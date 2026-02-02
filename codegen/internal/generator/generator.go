package generator

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"

	"github.com/sumup/sumup-dotnet/codegen/internal/naming"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type schemaKind int

const (
	schemaKindAlias schemaKind = iota
	schemaKindObject
	schemaKindEnum
)

type schemaTypeInfo struct {
	Name             string
	TypeName         string
	Kind             schemaKind
	Schema           *base.SchemaProxy
	EnumValues       []string
	AliasType        string
	AliasIsValueType bool
}

// Config holds runtime settings for the generator.
type Config struct {
	OutputDir string
	Namespace string
}

// Generator renders the SumUp .NET SDK from the OpenAPI spec.
type Generator struct {
	config       Config
	schemaTypes  map[string]*schemaTypeInfo
	inlineModels []modelTemplateData
	modelNames   map[string]struct{}
	errorModels  map[string]struct{}
}

// New returns a new Generator.
func New(config Config) *Generator {
	return &Generator{
		config:      config,
		schemaTypes: map[string]*schemaTypeInfo{},
		modelNames:  map[string]struct{}{},
		errorModels: map[string]struct{}{},
	}
}

// Run executes the generator.
func (g *Generator) Run(doc *v3.Document) error {
	if g.config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	g.inlineModels = nil
	g.modelNames = map[string]struct{}{}
	g.errorModels = map[string]struct{}{}

	if g.config.Namespace == "" {
		g.config.Namespace = "SumUp"
	}

	if err := os.MkdirAll(g.config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	if err := g.cleanOutputDir(); err != nil {
		return fmt.Errorf("clean output: %w", err)
	}

	tmpl, err := template.New("clients").ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return fmt.Errorf("parse templates: %w", err)
	}

	models, err := g.buildModels(doc)
	if err != nil {
		return err
	}

	clients, err := g.buildClients(doc)
	if err != nil {
		return err
	}

	if len(g.inlineModels) > 0 {
		models = append(models, g.inlineModels...)
		sort.Slice(models, func(i, j int) bool {
			return models[i].Name < models[j].Name
		})
	}

	for i := range models {
		if _, ok := g.errorModels[models[i].Name]; ok {
			models[i].EmitToString = true
			models[i].UsesJson = true
		}
	}

	if err := g.renderModels(tmpl, models); err != nil {
		return err
	}

	for _, client := range clients {
		if err := g.renderClient(tmpl, client); err != nil {
			return err
		}
	}

	rootData := rootTemplateData{
		Namespace: g.config.Namespace,
		Clients:   clients,
	}
	if err := g.renderRoot(tmpl, rootData); err != nil {
		return err
	}
	if err := g.renderApiVersion(tmpl, apiVersionTemplateData{
		Namespace:  g.config.Namespace,
		ApiVersion: apiVersionFromSpec(doc),
	}); err != nil {
		return err
	}

	return nil
}

func (g *Generator) renderClient(t *template.Template, client clientTemplateData) (err error) {
	filePath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%sClient.g.cs", client.ClientName))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close client file: %w", closeErr)
		}
	}()

	if err := t.ExecuteTemplate(file, "client.tmpl", client); err != nil {
		return fmt.Errorf("render template %s: %w", client.ClientName, err)
	}
	return nil
}

func (g *Generator) renderRoot(t *template.Template, data rootTemplateData) (err error) {
	filePath := filepath.Join(g.config.OutputDir, "SumUpClient.g.cs")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create root file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close root file: %w", closeErr)
		}
	}()

	if err := t.ExecuteTemplate(file, "root_client.tmpl", data); err != nil {
		return fmt.Errorf("render root template: %w", err)
	}
	return nil
}

func (g *Generator) renderApiVersion(t *template.Template, data apiVersionTemplateData) (err error) {
	targetDir := filepath.Join(g.config.OutputDir, "Http")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create http directory: %w", err)
	}
	filePath := filepath.Join(targetDir, "ApiVersion.g.cs")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create api version file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close api version file: %w", closeErr)
		}
	}()

	if err := t.ExecuteTemplate(file, "api_version.tmpl", data); err != nil {
		return fmt.Errorf("render api version template: %w", err)
	}
	return nil
}

func (g *Generator) renderModels(t *template.Template, models []modelTemplateData) error {
	for _, model := range models {
		var templateName string
		switch model.Kind {
		case schemaKindEnum:
			templateName = "model_enum.tmpl"
		default:
			templateName = "model_class.tmpl"
		}

		targetDir := filepath.Join(g.config.OutputDir, "Models")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return fmt.Errorf("create models directory: %w", err)
		}
		filePath := filepath.Join(targetDir, fmt.Sprintf("%s.g.cs", model.Name))
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("create model file: %w", err)
		}
		if err := t.ExecuteTemplate(file, templateName, model); err != nil {
			closeErr := file.Close()
			if closeErr != nil {
				return errors.Join(
					fmt.Errorf("render model template %s: %w", model.Name, err),
					fmt.Errorf("close model file %s: %w", model.Name, closeErr),
				)
			}
			return fmt.Errorf("render model template %s: %w", model.Name, err)
		}
		if err := file.Close(); err != nil {
			return fmt.Errorf("close model file %s: %w", model.Name, err)
		}
	}
	return nil
}

func (g *Generator) buildModels(doc *v3.Document) ([]modelTemplateData, error) {
	g.schemaTypes = map[string]*schemaTypeInfo{}
	if doc.Components == nil || doc.Components.Schemas == nil || doc.Components.Schemas.Len() == 0 {
		return nil, nil
	}

	names := []string{}
	for name := range doc.Components.Schemas.KeysFromOldest() {
		names = append(names, name)
		schemaProxy := doc.Components.Schemas.GetOrZero(name)
		g.schemaTypes[name] = &schemaTypeInfo{
			Name:     name,
			TypeName: naming.PascalIdentifier(name),
			Schema:   schemaProxy,
		}
		g.modelNames[g.schemaTypes[name].TypeName] = struct{}{}
	}
	sort.Strings(names)

	for _, name := range names {
		info := g.schemaTypes[name]
		schema := g.schemaFromProxy(info.Schema)
		if schema == nil {
			continue
		}
		info.Kind = g.classifySchema(schema)
	}

	for _, name := range names {
		info := g.schemaTypes[name]
		if info.Kind != schemaKindAlias {
			continue
		}
		schema := g.schemaFromProxy(info.Schema)
		if schema == nil {
			continue
		}
		typeInfo := g.resolveType(base.CreateSchemaProxy(schema), true)
		info.AliasType = strings.TrimSuffix(typeInfo.TypeName, "?")
		info.AliasIsValueType = typeInfo.IsValueType
	}

	models := make([]modelTemplateData, 0, len(names))
	for _, name := range names {
		info := g.schemaTypes[name]
		schema := g.schemaFromProxy(info.Schema)
		if schema == nil {
			continue
		}
		switch info.Kind {
		case schemaKindEnum:
			enumValues := g.buildEnumValues(schema)
			models = append(models, modelTemplateData{
				Namespace:       g.config.Namespace,
				Name:            info.TypeName,
				Description:     sanitizeText(schema.Description),
				Kind:            schemaKindEnum,
				EnumValues:      enumValues,
				UsesCollections: false,
			})
			info.AliasType = info.TypeName
			info.AliasIsValueType = true
		case schemaKindObject:
			model, err := g.buildClassModel(info.TypeName, schema)
			if err != nil {
				return nil, err
			}
			model.Kind = schemaKindObject
			models = append(models, model)
			info.AliasType = info.TypeName
			info.AliasIsValueType = false
		}
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	return models, nil
}

func (g *Generator) buildEnumValues(schema *base.Schema) []enumValueTemplateData {
	if schema == nil || len(schema.Enum) == 0 {
		return nil
	}
	names := map[string]int{}
	values := make([]enumValueTemplateData, 0, len(schema.Enum))
	for _, raw := range schema.Enum {
		str := yamlNodeToString(raw)
		name := naming.PascalIdentifier(str)
		if count, exists := names[name]; exists {
			count++
			names[name] = count
			name = fmt.Sprintf("%s%d", name, count)
		} else {
			names[name] = 0
		}
		values = append(values, enumValueTemplateData{
			Name:  name,
			Value: str,
		})
	}
	return values
}

func (g *Generator) buildClassModel(typeName string, schema *base.Schema) (modelTemplateData, error) {
	props, usesCollections, usesJson, err := g.collectProperties(typeName, schema)
	if err != nil {
		return modelTemplateData{}, err
	}
	extensionType := ""
	if schema.AdditionalProperties != nil {
		if schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			val := g.resolveType(schema.AdditionalProperties.A, true)
			extensionType = strings.TrimSuffix(val.TypeName, "?")
			usesCollections = true
			if strings.Contains(extensionType, "Json") {
				usesJson = true
			}
		} else if schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			extensionType = "JsonElement"
			usesCollections = true
			usesJson = true
		}
	}
	return modelTemplateData{
		Namespace:              g.config.Namespace,
		Name:                   typeName,
		Description:            sanitizeText(schema.Description),
		Properties:             props,
		Kind:                   schemaKindObject,
		HasProperties:          len(props) > 0,
		UsesCollections:        usesCollections,
		UsesJson:               usesJson,
		HasExtensionData:       extensionType != "",
		ExtensionDataValueType: extensionType,
	}, nil
}

func (g *Generator) collectProperties(ownerName string, schema *base.Schema) ([]modelPropertyTemplateData, bool, bool, error) {
	if schema == nil {
		return nil, false, false, nil
	}
	propMap := map[string]modelPropertyTemplateData{}
	usesCollections := false
	usesJson := false

	addProps := func(source *base.Schema) error {
		if source == nil {
			return nil
		}
		requiredSet := make(map[string]struct{}, len(source.Required))
		for _, name := range source.Required {
			requiredSet[name] = struct{}{}
		}
		names := make([]string, 0)
		if source.Properties != nil {
			for name := range source.Properties.KeysFromOldest() {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			if source.Properties == nil {
				continue
			}
			propRef := source.Properties.GetOrZero(name)
			required := false
			if _, ok := requiredSet[name]; ok {
				required = true
			}
			typeInfo, err := g.resolvePropertyType(ownerName, name, propRef, required)
			if err != nil {
				return err
			}
			desc := sanitizeText(g.schemaDescription(propRef))
			prop := modelPropertyTemplateData{
				PropertyName:     naming.PascalIdentifier(name),
				JsonName:         name,
				TypeName:         typeInfo.TypeName,
				Description:      desc,
				Required:         required,
				NeedsInitializer: required && !typeInfo.IsValueType && !strings.HasSuffix(typeInfo.TypeName, "?"),
				IsValueType:      typeInfo.IsValueType,
				IsNullable:       strings.HasSuffix(typeInfo.TypeName, "?"),
			}
			propMap[name] = prop
			if typeInfo.IsCollection {
				usesCollections = true
			}
			if strings.Contains(typeInfo.TypeName, "JsonDocument") || strings.Contains(typeInfo.TypeName, "JsonElement") {
				usesJson = true
			}
		}
		return nil
	}

	if err := addProps(schema); err != nil {
		return nil, false, false, err
	}
	for _, allOf := range schema.AllOf {
		sub := g.schemaFromProxy(allOf)
		if sub == nil {
			continue
		}
		if err := addProps(sub); err != nil {
			return nil, false, false, err
		}
	}

	keys := make([]string, 0, len(propMap))
	for key := range propMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	properties := make([]modelPropertyTemplateData, 0, len(keys))
	for _, key := range keys {
		properties = append(properties, propMap[key])
	}
	return properties, usesCollections, usesJson, nil
}

func (g *Generator) schemaFromProxy(proxy *base.SchemaProxy) *base.Schema {
	if proxy == nil {
		return nil
	}
	if schema := proxy.Schema(); schema != nil {
		return schema
	}
	if proxy.IsReference() {
		name := componentName(proxy.GetReference())
		if info, ok := g.schemaTypes[name]; ok {
			return g.schemaFromProxy(info.Schema)
		}
	}
	return nil
}

func (g *Generator) schemaDescription(proxy *base.SchemaProxy) string {
	if proxy == nil {
		return ""
	}
	if schema := g.schemaFromProxy(proxy); schema != nil {
		return schema.Description
	}
	return ""
}

func (g *Generator) classifySchema(schema *base.Schema) schemaKind {
	if schema == nil {
		return schemaKindAlias
	}
	if len(schema.Enum) > 0 {
		return schemaKindEnum
	}
	if (schema.Properties != nil && schema.Properties.Len() > 0) || len(schema.AllOf) > 0 {
		return schemaKindObject
	}
	if schemaHasType(schema, "object") {
		return schemaKindObject
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
		return schemaKindObject
	}
	return schemaKindAlias
}

func (g *Generator) buildClients(doc *v3.Document) ([]clientTemplateData, error) {
	clientMap := map[string]*clientTemplateData{}
	nameCounts := map[string]int{}

	if doc.Paths == nil || doc.Paths.PathItems == nil || doc.Paths.PathItems.Len() == 0 {
		return nil, fmt.Errorf("spec contains no paths")
	}

	for rawPath, pathItem := range doc.Paths.PathItems.FromOldest() {
		if pathItem == nil {
			continue
		}
		ops := pathItem.GetOperations()
		if ops == nil {
			continue
		}
		for methodName, operation := range ops.FromOldest() {
			if operation == nil {
				continue
			}
			tag := "Core"
			if len(operation.Tags) > 0 {
				tag = operation.Tags[0]
			}
			clientName := naming.PascalIdentifier(tag)
			ct, ok := clientMap[clientName]
			if !ok {
				ct = &clientTemplateData{
					Namespace:    g.config.Namespace,
					ClientName:   clientName,
					PropertyName: clientName,
				}
				clientMap[clientName] = ct
			}

			httpMethod := canonicalMethodName(methodName)
			baseName := g.operationBaseName(httpMethod, rawPath, operation)
			pascalName := naming.PascalIdentifier(baseName)
			if pascalName == "" {
				pascalName = generateOperationName(httpMethod, rawPath)
			}
			key := fmt.Sprintf("%s.%s", clientName, pascalName)
			count := nameCounts[key]
			nameCounts[key] = count + 1
			methodNameFinal := pascalName
			if count > 0 {
				methodNameFinal = fmt.Sprintf("%s%d", pascalName, count+1)
			}

			method, err := g.buildOperation(rawPath, httpMethod, methodNameFinal, clientName, operation, pathItem)
			if err != nil {
				return nil, err
			}
			ct.Operations = append(ct.Operations, method)
			ct.UsesCollections = ct.UsesCollections || method.UsesCollections
			ct.UsesJson = true // responses default to JSON
			if method.HasErrorResponses {
				ct.UsesErrorResponses = true
			}
		}
	}

	clients := make([]clientTemplateData, 0, len(clientMap))
	for _, client := range clientMap {
		sort.Slice(client.Operations, func(i, j int) bool {
			return client.Operations[i].MethodName < client.Operations[j].MethodName
		})
		clients = append(clients, *client)
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].ClientName < clients[j].ClientName
	})
	return clients, nil
}

func (g *Generator) buildOperation(path, method, methodName, clientName string, op *v3.Operation, pathItem *v3.PathItem) (operationTemplateData, error) {
	parameters := mergeParameters(pathItem.Parameters, op.Parameters)
	var (
		pathParams   []parameterTemplateData
		queryParams  []parameterTemplateData
		headerParams []parameterTemplateData
	)
	pathIndex := map[string]int{}
	queryIndex := map[string]int{}
	headerIndex := map[string]int{}

	for _, param := range parameters {
		if param == nil {
			continue
		}
		parameter, err := g.convertParameter(param)
		if err != nil {
			return operationTemplateData{}, err
		}
		switch parameter.Location {
		case "path":
			if idx, ok := pathIndex[parameter.Name]; ok {
				pathParams[idx] = parameter
			} else {
				pathIndex[parameter.Name] = len(pathParams)
				pathParams = append(pathParams, parameter)
			}
		case "query":
			if idx, ok := queryIndex[parameter.Name]; ok {
				queryParams[idx] = parameter
			} else {
				queryIndex[parameter.Name] = len(queryParams)
				queryParams = append(queryParams, parameter)
			}
		case "header":
			if idx, ok := headerIndex[parameter.Name]; ok {
				headerParams[idx] = parameter
			} else {
				headerIndex[parameter.Name] = len(headerParams)
				headerParams = append(headerParams, parameter)
			}
		}
	}

	body, err := g.buildRequestBody(clientName, methodName, op.RequestBody)
	if err != nil {
		return operationTemplateData{}, err
	}
	responseInfo, err := g.resolveResponseType(op, clientName, methodName)
	if err != nil {
		return operationTemplateData{}, err
	}
	responseMode, err := g.resolveResponseMode(op)
	if err != nil {
		return operationTemplateData{}, err
	}
	errorResponses, err := g.resolveErrorResponses(op, clientName, methodName)
	if err != nil {
		return operationTemplateData{}, err
	}

	allParams := append(append([]methodParameter{}, toMethodParameters(pathParams)...), toMethodParameters(queryParams)...)
	allParams = append(allParams, toMethodParameters(headerParams)...)
	if body != nil {
		descriptionText := body.Description
		if descriptionText == "" {
			descriptionText = "Request body payload."
		}
		allParams = append(allParams, methodParameter{
			Name:        body.ArgName,
			Signature:   body.Signature,
			Description: descriptionText,
		})
	}

	summary := sanitizeText(firstNonEmpty(op.Summary, fmt.Sprintf("%s %s", method, path)))
	description := sanitizeText(op.Description)
	usesCollections := hasCollections(pathParams) || hasCollections(queryParams) || hasCollections(headerParams)
	if body != nil && body.IsCollection {
		usesCollections = true
	}
	if responseInfo.IsCollection {
		usesCollections = true
	}
	data := operationTemplateData{
		MethodName:      methodName,
		HttpMethod:      method,
		HttpMethodExpr:  httpMethodExpression(method),
		Path:            path,
		Summary:         summary,
		Description:     description,
		PathParams:      pathParams,
		QueryParams:     queryParams,
		HeaderParams:    headerParams,
		Parameters:      allParams,
		HasParameters:   len(allParams) > 0,
		HasRequestBody:  body != nil,
		Body:            body,
		ResponseType:    responseInfo.TypeName,
		HasBuilder:      len(pathParams)+len(queryParams)+len(headerParams) > 0,
		HasPathParams:   len(pathParams) > 0,
		HasQueryParams:  len(queryParams) > 0,
		HasHeaderParams: len(headerParams) > 0,
		UsesCollections: usesCollections,
		HasErrorResponses: len(errorResponses) > 0,
		ErrorResponses: errorResponses,
		ResponseMode:   responseMode,
	}
	return data, nil
}

func (g *Generator) convertParameter(param *v3.Parameter) (parameterTemplateData, error) {
	if param == nil {
		return parameterTemplateData{}, fmt.Errorf("parameter is nil")
	}
	required := param.Required != nil && *param.Required
	typeInfo := g.resolveType(param.Schema, required)
	defaultValue := ""
	if !required {
		defaultValue = " = null"
	}

	argName := naming.Identifier(param.Name)
	return parameterTemplateData{
		Location:     param.In,
		Name:         param.Name,
		ArgName:      argName,
		Declaration:  fmt.Sprintf("%s %s%s", typeInfo.TypeName, argName, defaultValue),
		Description:  sanitizeText(param.Description),
		Required:     required,
		BuilderCall:  builderCall(param.In, param.Name, argName),
		IsCollection: typeInfo.IsCollection,
	}, nil
}

func (g *Generator) buildRequestBody(clientName, methodName string, body *v3.RequestBody) (*bodyTemplateData, error) {
	if body == nil {
		return nil, nil
	}

	contentType := firstContentType(body.Content)
	if contentType == "" {
		return nil, nil
	}

	schemaRef := preferredSchema(body.Content)
	if schemaRef == nil {
		return nil, nil
	}

	required := body.Required != nil && *body.Required
	typeInfo, err := g.resolveRequestBodyType(schemaRef, required, clientName, methodName)
	if err != nil {
		return nil, err
	}
	signature := fmt.Sprintf("%s body", typeInfo.TypeName)
	if !required {
		signature = fmt.Sprintf("%s body = null", typeInfo.TypeName)
	}

	return &bodyTemplateData{
		ArgName:      "body",
		Signature:    signature,
		Description:  sanitizeText(body.Description),
		Required:     required,
		ContentType:  contentType,
		TypeName:     typeInfo.TypeName,
		IsCollection: typeInfo.IsCollection,
	}, nil
}

func (g *Generator) resolveRequestBodyType(schemaRef *base.SchemaProxy, required bool, clientName, methodName string) (typeInfo, error) {
	return g.resolveInlineSchemaType(schemaRef, required, fmt.Sprintf("%s%sRequest", clientName, methodName))
}

func (g *Generator) resolvePropertyType(ownerName, propertyName string, schemaRef *base.SchemaProxy, required bool) (typeInfo, error) {
	inlineBase := fmt.Sprintf("%s%s", ownerName, naming.PascalIdentifier(propertyName))
	return g.resolveInlineSchemaType(schemaRef, required, inlineBase)
}

func (g *Generator) resolveInlineSchemaType(schemaRef *base.SchemaProxy, required bool, inlineBase string) (typeInfo, error) {
	if schemaRef == nil {
		return g.nullableType("JsonDocument", false, required), nil
	}
	if inlineBase != "" && !schemaRef.IsReference() {
		schema := g.schemaFromProxy(schemaRef)
		if schema == nil {
			return g.nullableType("JsonDocument", false, required), nil
		}
		if schemaDefinesStructuredObject(schema) {
			typeName, err := g.createInlineModel(inlineBase, schema)
			if err != nil {
				return typeInfo{}, err
			}
			return g.nullableType(typeName, false, required), nil
		}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			valueInfo, err := g.resolveInlineSchemaType(schema.AdditionalProperties.A, true, inlineBase+"Value")
			if err != nil {
				return typeInfo{}, err
			}
			valueName := strings.TrimSuffix(valueInfo.TypeName, "?")
			typeName := fmt.Sprintf("IDictionary<string, %s>", valueName)
			return g.nullableType(typeName, false, required, true), nil
		}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			return g.nullableType("IDictionary<string, JsonElement>", false, required, true), nil
		}
		if schema.Items != nil && schema.Items.IsA() {
			itemInfo, err := g.resolveInlineSchemaType(schema.Items.A, true, inlineBase+"Item")
			if err != nil {
				return typeInfo{}, err
			}
			itemName := strings.TrimSuffix(itemInfo.TypeName, "?")
			typeName := fmt.Sprintf("IEnumerable<%s>", itemName)
			return g.nullableType(typeName, false, required, true), nil
		}
	}
	return g.resolveType(schemaRef, required), nil
}

func schemaDefinesStructuredObject(schema *base.Schema) bool {
	if schema == nil {
		return false
	}
	if (schema.Properties != nil && schema.Properties.Len() > 0) || len(schema.AllOf) > 0 {
		return true
	}
	return false
}

func (g *Generator) createInlineModel(baseName string, schema *base.Schema) (string, error) {
	typeName := g.reserveModelName(baseName)
	model, err := g.buildClassModel(typeName, schema)
	if err != nil {
		return "", err
	}
	g.inlineModels = append(g.inlineModels, model)
	return typeName, nil
}

func (g *Generator) reserveModelName(base string) string {
	name := base
	index := 2
	if g.modelNames == nil {
		g.modelNames = map[string]struct{}{}
	}
	for {
		if _, exists := g.modelNames[name]; !exists {
			g.modelNames[name] = struct{}{}
			return name
		}
		name = fmt.Sprintf("%s%d", base, index)
		index++
	}
}

func (g *Generator) resolveResponseType(op *v3.Operation, clientName, methodName string) (typeInfo, error) {
	if op == nil || op.Responses == nil || op.Responses.Codes == nil || op.Responses.Codes.Len() == 0 {
		return g.nullableType("JsonDocument", false, true), nil
	}
	codes := make([]string, 0, op.Responses.Codes.Len())
	for code := range op.Responses.Codes.KeysFromOldest() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if !strings.HasPrefix(code, "2") {
			continue
		}
		resp := op.Responses.Codes.GetOrZero(code)
		if info, err := g.responseTypeForResponse(resp, fmt.Sprintf("%s%sResponse", clientName, methodName)); err != nil {
			return typeInfo{}, err
		} else if info.TypeName != "" {
			return info, nil
		}
	}
	if resp := op.Responses.Default; resp != nil {
		if info, err := g.responseTypeForResponse(resp, fmt.Sprintf("%s%sResponseDefault", clientName, methodName)); err != nil {
			return typeInfo{}, err
		} else if info.TypeName != "" {
			return info, nil
		}
	}
	return g.nullableType("JsonDocument", false, true), nil
}

func (g *Generator) resolveErrorResponses(op *v3.Operation, clientName, methodName string) ([]errorResponseTemplateData, error) {
	if op == nil || op.Responses == nil || op.Responses.Codes == nil || op.Responses.Codes.Len() == 0 {
		if op != nil && op.Responses != nil && op.Responses.Default != nil {
			return g.resolveDefaultErrorResponse(op.Responses.Default, clientName, methodName)
		}
		return nil, nil
	}

	codes := make([]string, 0, op.Responses.Codes.Len())
	for code := range op.Responses.Codes.KeysFromOldest() {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	var responses []errorResponseTemplateData
	for _, code := range codes {
		if strings.HasPrefix(code, "2") {
			continue
		}
		resp := op.Responses.Codes.GetOrZero(code)
		parsed, err := g.errorResponseForCode(resp, code, clientName, methodName)
		if err != nil {
			return nil, err
		}
		if parsed != nil {
			responses = append(responses, *parsed)
		}
	}

	if op.Responses.Default != nil {
		defaultResponses, err := g.resolveDefaultErrorResponse(op.Responses.Default, clientName, methodName)
		if err != nil {
			return nil, err
		}
		responses = append(responses, defaultResponses...)
	}

	return responses, nil
}

func (g *Generator) resolveDefaultErrorResponse(resp *v3.Response, clientName, methodName string) ([]errorResponseTemplateData, error) {
	parsed, err := g.errorResponseForCode(resp, "default", clientName, methodName)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, nil
	}
	return []errorResponseTemplateData{*parsed}, nil
}

func (g *Generator) errorResponseForCode(resp *v3.Response, code, clientName, methodName string) (*errorResponseTemplateData, error) {
	if resp == nil {
		return nil, nil
	}
	statusSuffix := statusCodeSuffix(code)
	inlineBase := fmt.Sprintf("%s%sError%s", clientName, methodName, statusSuffix)
	info, err := g.responseTypeForResponse(resp, inlineBase)
	if err != nil {
		return nil, err
	}
	if info.TypeName == "" {
		return nil, nil
	}
	errorType := strings.TrimSuffix(info.TypeName, "?")
	if errorType != "" {
		g.errorModels[errorType] = struct{}{}
	}
	isDefault := strings.EqualFold(code, "default")
	statusLiteral := statusCodeLiteral(code)
	if !isDefault && statusLiteral == "" {
		return nil, nil
	}
	return &errorResponseTemplateData{
		StatusCodeLiteral: statusLiteral,
		ErrorType:         errorType,
		IsDefault:         isDefault,
	}, nil
}

func statusCodeLiteral(code string) string {
	if strings.EqualFold(code, "default") {
		return ""
	}
	if isNumericStatusCode(code) {
		return code
	}
	return ""
}

func statusCodeSuffix(code string) string {
	if strings.EqualFold(code, "default") {
		return "Default"
	}
	if isNumericStatusCode(code) {
		return code
	}
	parsed := naming.PascalIdentifier(code)
	if parsed == "" {
		return "Unknown"
	}
	return parsed
}

func isNumericStatusCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (g *Generator) responseTypeForResponse(resp *v3.Response, inlineBase string) (typeInfo, error) {
	if resp == nil || resp.Content == nil || resp.Content.Len() == 0 {
		return typeInfo{}, nil
	}
	schemaRef := preferredSchema(resp.Content)
	if schemaRef == nil {
		return typeInfo{}, nil
	}
	return g.resolveInlineSchemaType(schemaRef, true, inlineBase)
}

func (g *Generator) resolveResponseMode(op *v3.Operation) (string, error) {
	if op == nil || op.Responses == nil || op.Responses.Codes == nil || op.Responses.Codes.Len() == 0 {
		return "none", nil
	}
	codes := make([]string, 0, op.Responses.Codes.Len())
	for code := range op.Responses.Codes.KeysFromOldest() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if !strings.HasPrefix(code, "2") {
			continue
		}
		resp := op.Responses.Codes.GetOrZero(code)
		return g.responseModeForResponse(resp)
	}
	if resp := op.Responses.Default; resp != nil {
		return g.responseModeForResponse(resp)
	}
	return "none", nil
}

func (g *Generator) responseModeForResponse(resp *v3.Response) (string, error) {
	if resp == nil || resp.Content == nil || resp.Content.Len() == 0 {
		return "none", nil
	}
	contentType := firstContentType(resp.Content)
	schemaRef := preferredSchema(resp.Content)
	if schemaRef == nil {
		if strings.Contains(strings.ToLower(contentType), "text/") {
			return "string", nil
		}
		return "json", nil
	}

	typeInfo, err := g.resolveInlineSchemaType(schemaRef, true, "")
	if err != nil {
		return "", err
	}
	typeName := strings.TrimSuffix(typeInfo.TypeName, "?")
	if typeName == "string" {
		return "string", nil
	}
	if typeName == "JsonDocument" {
		if strings.Contains(strings.ToLower(contentType), "text/") {
			return "string", nil
		}
		return "json-document", nil
	}
	return "json", nil
}

func (g *Generator) operationBaseName(method, path string, op *v3.Operation) string {
	if name := methodNameFromExtension(op); name != "" {
		return name
	}
	if op != nil && op.OperationId != "" {
		return op.OperationId
	}
	return generateOperationName(method, path)
}

func methodNameFromExtension(op *v3.Operation) string {
	if op == nil || op.Extensions == nil {
		return ""
	}
	node := op.Extensions.GetOrZero("x-codegen")
	if node == nil || len(node.Content) == 0 {
		return ""
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		if keyNode != nil && keyNode.Value == "method_name" {
			if valueNode != nil && strings.TrimSpace(valueNode.Value) != "" {
				return valueNode.Value
			}
		}
	}
	return ""
}

func generateOperationName(method, path string) string {
	combined := fmt.Sprintf("%s_%s", method, path)
	combined = strings.ReplaceAll(combined, "{", "")
	combined = strings.ReplaceAll(combined, "}", "")
	return naming.PascalIdentifier(combined)
}

func mergeParameters(pathParams, opParams []*v3.Parameter) []*v3.Parameter {
	result := make([]*v3.Parameter, 0, len(pathParams)+len(opParams))
	result = appendParameters(result, pathParams)
	result = appendParameters(result, opParams)
	return result
}

func appendParameters(dst []*v3.Parameter, src []*v3.Parameter) []*v3.Parameter {
	for _, param := range src {
		if param == nil {
			continue
		}
		dst = append(dst, param)
	}
	return dst
}

func builderCall(location, name, arg string) string {
	switch location {
	case "path":
		return fmt.Sprintf(`builder.AddPath("%s", %s);`, name, arg)
	case "query":
		return fmt.Sprintf(`builder.AddQuery("%s", %s);`, name, arg)
	case "header":
		return fmt.Sprintf(`builder.AddHeader("%s", %s);`, name, arg)
	default:
		return ""
	}
}

func httpMethodExpression(method string) string {
	switch strings.ToLower(method) {
	case "get":
		return "HttpMethod.Get"
	case "post":
		return "HttpMethod.Post"
	case "put":
		return "HttpMethod.Put"
	case "delete":
		return "HttpMethod.Delete"
	case "options":
		return "HttpMethod.Options"
	case "head":
		return "HttpMethod.Head"
	case "trace":
		return "HttpMethod.Trace"
	case "patch":
		return `new HttpMethod("PATCH")`
	default:
		return fmt.Sprintf(`new HttpMethod("%s")`, strings.ToUpper(method))
	}
}

func (g *Generator) resolveType(schemaRef *base.SchemaProxy, required bool) typeInfo {
	if schemaRef == nil {
		return g.nullableType("JsonDocument", false, required)
	}
	if schemaRef.IsReference() {
		name := componentName(schemaRef.GetReference())
		if info, ok := g.schemaTypes[name]; ok {
			typeName := info.TypeName
			isValueType := info.AliasIsValueType
			if info.Kind == schemaKindAlias && info.AliasType != "" {
				typeName = info.AliasType
			}
			if info.Kind == schemaKindEnum {
				isValueType = true
			}
			if info.Kind == schemaKindObject {
				isValueType = false
			}
			return g.nullableType(typeName, isValueType, required)
		}
	}
	schema := g.schemaFromProxy(schemaRef)
	if schema == nil {
		return g.nullableType("JsonDocument", false, required)
	}
	if schema.Nullable != nil && *schema.Nullable {
		required = false
	}

	switch {
	case schemaHasType(schema, "string"):
		typeName := "string"
		switch schema.Format {
		case "date-time":
			typeName = "DateTimeOffset"
		case "date":
			typeName = "DateTime"
		case "uuid":
			typeName = "Guid"
		case "byte", "binary":
			typeName = "byte[]"
		}
		return g.nullableType(typeName, false, required)
	case schemaHasType(schema, "integer"):
		typeName := "int"
		if schema.Format == "int64" {
			typeName = "long"
		}
		return g.nullableType(typeName, true, required)
	case schemaHasType(schema, "number"):
		typeName := "decimal"
		if schema.Format == "float" {
			typeName = "float"
		}
		if schema.Format == "double" {
			typeName = "double"
		}
		return g.nullableType(typeName, true, required)
	case schemaHasType(schema, "boolean"):
		return g.nullableType("bool", true, required)
	case schemaHasType(schema, "array") || (schema.Items != nil && schema.Items.IsA()):
		if schema.Items != nil && schema.Items.IsA() {
			elem := g.resolveType(schema.Items.A, true)
			elemName := strings.TrimSuffix(elem.TypeName, "?")
			typeName := fmt.Sprintf("IEnumerable<%s>", elemName)
			return g.nullableType(typeName, false, required, true)
		}
		return g.nullableType("IEnumerable<JsonDocument>", false, required, true)
	case schemaHasType(schema, "object"):
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() && schema.AdditionalProperties.A != nil {
			valueType := g.resolveType(schema.AdditionalProperties.A, true)
			valueName := strings.TrimSuffix(valueType.TypeName, "?")
			typeName := fmt.Sprintf("IDictionary<string, %s>", valueName)
			return g.nullableType(typeName, false, required, true)
		}
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsB() && schema.AdditionalProperties.B {
			return g.nullableType("IDictionary<string, JsonElement>", false, required, true)
		}
		if (schema.Properties == nil || schema.Properties.Len() == 0) && len(schema.AllOf) == 0 {
			return g.nullableType("JsonDocument", false, required)
		}
		return g.nullableType("JsonDocument", false, required)
	default:
		return g.nullableType("JsonDocument", false, required)
	}
}

func (g *Generator) nullableType(typeName string, isValueType bool, required bool, collection ...bool) typeInfo {
	isCollection := false
	if len(collection) > 0 {
		isCollection = collection[0]
	}
	if !required && !strings.HasSuffix(typeName, "?") {
		typeName += "?"
	}
	return typeInfo{
		TypeName:     typeName,
		IsCollection: isCollection || strings.Contains(typeName, "IEnumerable") || strings.Contains(typeName, "IDictionary"),
		IsValueType:  isValueType,
	}
}

func firstContentType(content *orderedmap.Map[string, *v3.MediaType]) string {
	if content == nil {
		return ""
	}
	for k := range content.KeysFromOldest() {
		return k
	}
	return ""
}

func preferredSchema(content *orderedmap.Map[string, *v3.MediaType]) *base.SchemaProxy {
	if content == nil {
		return nil
	}
	if media := content.GetOrZero("application/json"); media != nil && media.Schema != nil {
		return media.Schema
	}
	for _, media := range content.FromOldest() {
		if media != nil && media.Schema != nil {
			return media.Schema
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sanitizeText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func yamlNodeToString(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	data, err := yaml.Marshal(node)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func schemaHasType(schema *base.Schema, target string) bool {
	if schema == nil {
		return false
	}
	for _, t := range schema.Type {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}

func canonicalMethodName(method string) string {
	if method == "" {
		return ""
	}
	lower := strings.ToLower(method)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

type typeInfo struct {
	TypeName     string
	IsCollection bool
	IsValueType  bool
}

type modelTemplateData struct {
	Namespace              string
	Name                   string
	Description            string
	Kind                   schemaKind
	Properties             []modelPropertyTemplateData
	EnumValues             []enumValueTemplateData
	HasProperties          bool
	UsesCollections        bool
	UsesJson               bool
	HasExtensionData       bool
	ExtensionDataValueType string
	EmitToString           bool
}

type modelPropertyTemplateData struct {
	PropertyName     string
	JsonName         string
	TypeName         string
	Description      string
	Required         bool
	NeedsInitializer bool
	IsValueType      bool
	IsNullable       bool
}

type enumValueTemplateData struct {
	Name  string
	Value string
}

type clientTemplateData struct {
	Namespace       string
	ClientName      string
	PropertyName    string
	Operations      []operationTemplateData
	UsesCollections bool
	UsesJson        bool
	UsesErrorResponses bool
}

type operationTemplateData struct {
	MethodName      string
	HttpMethod      string
	HttpMethodExpr  string
	Path            string
	Summary         string
	Description     string
	PathParams      []parameterTemplateData
	QueryParams     []parameterTemplateData
	HeaderParams    []parameterTemplateData
	Parameters      []methodParameter
	HasParameters   bool
	HasRequestBody  bool
	Body            *bodyTemplateData
	ResponseType    string
	HasBuilder      bool
	HasPathParams   bool
	HasQueryParams  bool
	HasHeaderParams bool
	UsesCollections bool
	HasErrorResponses bool
	ErrorResponses []errorResponseTemplateData
	ResponseMode   string
}

type errorResponseTemplateData struct {
	StatusCodeLiteral string
	ErrorType         string
	IsDefault         bool
}

type parameterTemplateData struct {
	Location     string
	Name         string
	ArgName      string
	Declaration  string
	Description  string
	Required     bool
	BuilderCall  string
	IsCollection bool
}

type methodParameter struct {
	Name        string
	Signature   string
	Description string
}

type bodyTemplateData struct {
	ArgName      string
	Signature    string
	Description  string
	Required     bool
	ContentType  string
	TypeName     string
	IsCollection bool
}

type rootTemplateData struct {
	Namespace string
	Clients   []clientTemplateData
}

type apiVersionTemplateData struct {
	Namespace  string
	ApiVersion string
}

func toMethodParameters(params []parameterTemplateData) []methodParameter {
	result := make([]methodParameter, 0, len(params))
	for _, param := range params {
		result = append(result, methodParameter{
			Name:        param.ArgName,
			Signature:   param.Declaration,
			Description: param.Description,
		})
	}
	return result
}

func apiVersionFromSpec(doc *v3.Document) string {
	if doc != nil && doc.Info != nil {
		version := strings.TrimSpace(doc.Info.Version)
		if version != "" {
			return version
		}
	}
	return "1.0.0"
}

func hasCollections(params []parameterTemplateData) bool {
	for _, param := range params {
		if param.IsCollection {
			return true
		}
	}
	return false
}

func (g *Generator) cleanOutputDir() error {
	return filepath.WalkDir(g.config.OutputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == g.config.OutputDir {
				return nil
			}
			name := d.Name()
			if name == "bin" || name == "obj" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".g.cs") {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func componentName(ref string) string {
	if ref == "" {
		return ""
	}
	segments := strings.Split(ref, "/")
	return segments[len(segments)-1]
}
