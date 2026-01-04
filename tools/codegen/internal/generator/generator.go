package generator

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/sumup/sumup-dotnet/tools/codegen/internal/naming"
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
	Schema           *openapi3.SchemaRef
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
}

// New returns a new Generator.
func New(config Config) *Generator {
	return &Generator{
		config:      config,
		schemaTypes: map[string]*schemaTypeInfo{},
		modelNames:  map[string]struct{}{},
	}
}

// Run executes the generator.
func (g *Generator) Run(doc *openapi3.T) error {
	if g.config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	g.inlineModels = nil
	g.modelNames = map[string]struct{}{}

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

	return nil
}

func (g *Generator) renderClient(t *template.Template, client clientTemplateData) error {
	filePath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%sClient.g.cs", client.ClientName))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := t.ExecuteTemplate(file, "client.tmpl", client); err != nil {
		return fmt.Errorf("render template %s: %w", client.ClientName, err)
	}
	return nil
}

func (g *Generator) renderRoot(t *template.Template, data rootTemplateData) error {
	filePath := filepath.Join(g.config.OutputDir, "SumUpClient.g.cs")
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create root file: %w", err)
	}
	defer file.Close()

	if err := t.ExecuteTemplate(file, "root_client.tmpl", data); err != nil {
		return fmt.Errorf("render root template: %w", err)
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
			file.Close()
			return fmt.Errorf("render model template %s: %w", model.Name, err)
		}
		file.Close()
	}
	return nil
}

func (g *Generator) buildModels(doc *openapi3.T) ([]modelTemplateData, error) {
	g.schemaTypes = map[string]*schemaTypeInfo{}
	if doc.Components.Schemas == nil {
		return nil, nil
	}

	names := make([]string, 0, len(doc.Components.Schemas))
	for name, schemaRef := range doc.Components.Schemas {
		names = append(names, name)
		g.schemaTypes[name] = &schemaTypeInfo{
			Name:     name,
			TypeName: naming.PascalIdentifier(name),
			Schema:   schemaRef,
		}
		g.modelNames[g.schemaTypes[name].TypeName] = struct{}{}
	}
	sort.Strings(names)

	for _, name := range names {
		info := g.schemaTypes[name]
		schema := g.schemaFromRef(info.Schema)
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
		schema := g.schemaFromRef(info.Schema)
		if schema == nil {
			continue
		}
		typeInfo := g.resolveType(&openapi3.SchemaRef{Value: schema}, true)
		info.AliasType = strings.TrimSuffix(typeInfo.TypeName, "?")
		info.AliasIsValueType = typeInfo.IsValueType
	}

	models := make([]modelTemplateData, 0, len(names))
	for _, name := range names {
		info := g.schemaTypes[name]
		schema := g.schemaFromRef(info.Schema)
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

func (g *Generator) buildEnumValues(schema *openapi3.Schema) []enumValueTemplateData {
	if schema == nil || len(schema.Enum) == 0 {
		return nil
	}
	names := map[string]int{}
	values := make([]enumValueTemplateData, 0, len(schema.Enum))
	for _, raw := range schema.Enum {
		str := fmt.Sprintf("%v", raw)
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

func (g *Generator) buildClassModel(typeName string, schema *openapi3.Schema) (modelTemplateData, error) {
	props, usesCollections, usesJson, err := g.collectProperties(typeName, schema)
	if err != nil {
		return modelTemplateData{}, err
	}
	extensionType := ""
	if schema.AdditionalProperties.Schema != nil {
		val := g.resolveType(schema.AdditionalProperties.Schema, true)
		extensionType = strings.TrimSuffix(val.TypeName, "?")
		usesCollections = true
		if strings.Contains(extensionType, "Json") {
			usesJson = true
		}
	} else if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		extensionType = "JsonElement"
		usesCollections = true
		usesJson = true
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

func (g *Generator) collectProperties(ownerName string, schema *openapi3.Schema) ([]modelPropertyTemplateData, bool, bool, error) {
	if schema == nil {
		return nil, false, false, nil
	}
	propMap := map[string]modelPropertyTemplateData{}
	usesCollections := false
	usesJson := false

	addProps := func(source *openapi3.Schema) error {
		if source == nil {
			return nil
		}
		requiredSet := make(map[string]struct{}, len(source.Required))
		for _, name := range source.Required {
			requiredSet[name] = struct{}{}
		}
		names := make([]string, 0, len(source.Properties))
		for name := range source.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			propRef := source.Properties[name]
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
		sub := g.schemaFromRef(allOf)
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

func (g *Generator) schemaFromRef(ref *openapi3.SchemaRef) *openapi3.Schema {
	if ref == nil {
		return nil
	}
	if ref.Value != nil {
		return ref.Value
	}
	if ref.Ref == "" {
		return nil
	}
	name := componentName(ref.Ref)
	if info, ok := g.schemaTypes[name]; ok && info.Schema != nil && info.Schema.Value != nil {
		return info.Schema.Value
	}
	return nil
}

func (g *Generator) schemaDescription(ref *openapi3.SchemaRef) string {
	if ref == nil {
		return ""
	}
	if ref.Value != nil && ref.Value.Description != "" {
		return ref.Value.Description
	}
	if ref.Ref != "" {
		name := componentName(ref.Ref)
		if info, ok := g.schemaTypes[name]; ok {
			if schema := g.schemaFromRef(info.Schema); schema != nil {
				return schema.Description
			}
		}
	}
	return ""
}

func (g *Generator) classifySchema(schema *openapi3.Schema) schemaKind {
	if schema == nil {
		return schemaKindAlias
	}
	if len(schema.Enum) > 0 {
		return schemaKindEnum
	}
	if len(schema.Properties) > 0 || len(schema.AllOf) > 0 {
		return schemaKindObject
	}
	if schema.Type == "object" || schema.AdditionalProperties.Schema != nil {
		return schemaKindObject
	}
	return schemaKindAlias
}

func (g *Generator) buildClients(doc *openapi3.T) ([]clientTemplateData, error) {
	clientMap := map[string]*clientTemplateData{}
	nameCounts := map[string]int{}

	if doc.Paths == nil {
		return nil, fmt.Errorf("spec contains no paths")
	}

	for rawPath, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, entry := range []struct {
			Method string
			Op     *openapi3.Operation
		}{
			{"Get", pathItem.Get},
			{"Put", pathItem.Put},
			{"Post", pathItem.Post},
			{"Delete", pathItem.Delete},
			{"Options", pathItem.Options},
			{"Head", pathItem.Head},
			{"Patch", pathItem.Patch},
			{"Trace", pathItem.Trace},
		} {
			if entry.Op == nil {
				continue
			}
			tag := "Core"
			if len(entry.Op.Tags) > 0 {
				tag = entry.Op.Tags[0]
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

			baseName := g.operationBaseName(entry.Method, rawPath, entry.Op)
			pascalName := naming.PascalIdentifier(baseName)
			if pascalName == "" {
				pascalName = generateOperationName(entry.Method, rawPath)
			}
			key := fmt.Sprintf("%s.%s", clientName, pascalName)
			count := nameCounts[key]
			nameCounts[key] = count + 1
			methodName := pascalName
			if count > 0 {
				methodName = fmt.Sprintf("%s%d", pascalName, count+1)
			}

			method, err := g.buildOperation(doc, rawPath, entry.Method, methodName, clientName, entry.Op, pathItem)
			if err != nil {
				return nil, err
			}
			ct.Operations = append(ct.Operations, method)
			ct.UsesCollections = ct.UsesCollections || method.UsesCollections
			ct.UsesJson = true // responses default to JSON
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

func (g *Generator) buildOperation(doc *openapi3.T, path, method, methodName, clientName string, op *openapi3.Operation, pathItem *openapi3.PathItem) (operationTemplateData, error) {
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
		parameter, err := g.convertParameter(doc, param)
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

	body, err := g.buildRequestBody(doc, clientName, methodName, op.RequestBody)
	if err != nil {
		return operationTemplateData{}, err
	}
	responseInfo, err := g.resolveResponseType(op, clientName, methodName)
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
	}
	return data, nil
}

func (g *Generator) convertParameter(doc *openapi3.T, paramRef *openapi3.ParameterRef) (parameterTemplateData, error) {
	param, err := resolveParameter(doc, paramRef)
	if err != nil {
		return parameterTemplateData{}, err
	}
	typeInfo := g.resolveType(param.Schema, param.Required)
	defaultValue := ""
	if !param.Required {
		defaultValue = " = null"
	}

	argName := naming.Identifier(param.Name)
	return parameterTemplateData{
		Location:     param.In,
		Name:         param.Name,
		ArgName:      argName,
		Declaration:  fmt.Sprintf("%s %s%s", typeInfo.TypeName, argName, defaultValue),
		Description:  sanitizeText(param.Description),
		Required:     param.Required,
		BuilderCall:  builderCall(param.In, param.Name, argName),
		IsCollection: typeInfo.IsCollection,
	}, nil
}

func (g *Generator) buildRequestBody(doc *openapi3.T, clientName, methodName string, bodyRef *openapi3.RequestBodyRef) (*bodyTemplateData, error) {
	if bodyRef == nil {
		return nil, nil
	}

	body, err := resolveRequestBody(doc, bodyRef)
	if err != nil {
		return nil, err
	}

	contentType := firstContentType(body.Content)
	if contentType == "" {
		return nil, nil
	}

	schemaRef := preferredSchema(body.Content)
	if schemaRef == nil {
		return nil, nil
	}

	typeInfo, err := g.resolveRequestBodyType(schemaRef, body.Required, clientName, methodName)
	if err != nil {
		return nil, err
	}
	signature := fmt.Sprintf("%s body", typeInfo.TypeName)
	if !body.Required {
		signature = fmt.Sprintf("%s body = null", typeInfo.TypeName)
	}

	return &bodyTemplateData{
		ArgName:      "body",
		Signature:    signature,
		Description:  sanitizeText(body.Description),
		Required:     body.Required,
		ContentType:  contentType,
		TypeName:     typeInfo.TypeName,
		IsCollection: typeInfo.IsCollection,
	}, nil
}

func (g *Generator) resolveRequestBodyType(schemaRef *openapi3.SchemaRef, required bool, clientName, methodName string) (typeInfo, error) {
	return g.resolveInlineSchemaType(schemaRef, required, fmt.Sprintf("%s%sRequest", clientName, methodName))
}

func (g *Generator) resolvePropertyType(ownerName, propertyName string, schemaRef *openapi3.SchemaRef, required bool) (typeInfo, error) {
	inlineBase := fmt.Sprintf("%s%s", ownerName, naming.PascalIdentifier(propertyName))
	return g.resolveInlineSchemaType(schemaRef, required, inlineBase)
}

func (g *Generator) resolveInlineSchemaType(schemaRef *openapi3.SchemaRef, required bool, inlineBase string) (typeInfo, error) {
	if schemaRef == nil {
		return g.nullableType("JsonDocument", false, required), nil
	}
	if inlineBase != "" && schemaRef.Ref == "" && schemaRef.Value != nil {
		schema := schemaRef.Value
		if schemaDefinesStructuredObject(schema) {
			typeName, err := g.createInlineModel(inlineBase, schema)
			if err != nil {
				return typeInfo{}, err
			}
			return g.nullableType(typeName, false, required), nil
		}
		if schema.AdditionalProperties.Schema != nil {
			valueInfo, err := g.resolveInlineSchemaType(schema.AdditionalProperties.Schema, true, inlineBase+"Value")
			if err != nil {
				return typeInfo{}, err
			}
			valueName := strings.TrimSuffix(valueInfo.TypeName, "?")
			typeName := fmt.Sprintf("IDictionary<string, %s>", valueName)
			return g.nullableType(typeName, false, required, true), nil
		}
		if schema.Items != nil && (schema.Type == "array" || schema.Type == "") {
			itemInfo, err := g.resolveInlineSchemaType(schema.Items, true, inlineBase+"Item")
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

func schemaDefinesStructuredObject(schema *openapi3.Schema) bool {
	if schema == nil {
		return false
	}
	if len(schema.Properties) > 0 || len(schema.AllOf) > 0 {
		return true
	}
	return false
}

func (g *Generator) createInlineModel(baseName string, schema *openapi3.Schema) (string, error) {
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

func (g *Generator) resolveResponseType(op *openapi3.Operation, clientName, methodName string) (typeInfo, error) {
	if op == nil || op.Responses == nil || op.Responses.Len() == 0 {
		return g.nullableType("JsonDocument", false, true), nil
	}
	codes := make([]string, 0, op.Responses.Len())
	for code := range op.Responses.Map() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if !strings.HasPrefix(code, "2") {
			continue
		}
		if info, err := g.responseTypeForRef(op.Responses.Value(code), fmt.Sprintf("%s%sResponse", clientName, methodName)); err != nil {
			return typeInfo{}, err
		} else if info.TypeName != "" {
			return info, nil
		}
	}
	if resp := op.Responses.Default(); resp != nil {
		if info, err := g.responseTypeForRef(resp, fmt.Sprintf("%s%sResponseDefault", clientName, methodName)); err != nil {
			return typeInfo{}, err
		} else if info.TypeName != "" {
			return info, nil
		}
	}
	return g.nullableType("JsonDocument", false, true), nil
}

func (g *Generator) responseTypeForRef(respRef *openapi3.ResponseRef, inlineBase string) (typeInfo, error) {
	if respRef == nil || respRef.Value == nil || respRef.Value.Content == nil {
		return typeInfo{}, nil
	}
	schemaRef := preferredSchema(respRef.Value.Content)
	if schemaRef == nil {
		return typeInfo{}, nil
	}
	return g.resolveInlineSchemaType(schemaRef, true, inlineBase)
}

func (g *Generator) operationBaseName(method, path string, op *openapi3.Operation) string {
	if name := methodNameFromExtension(op); name != "" {
		return name
	}
	if op != nil && op.OperationID != "" {
		return op.OperationID
	}
	return generateOperationName(method, path)
}

func methodNameFromExtension(op *openapi3.Operation) string {
	if op == nil || op.Extensions == nil {
		return ""
	}
	raw, ok := op.Extensions["x-codegen"]
	if !ok {
		return ""
	}
	asMap, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	if value, ok := asMap["method_name"]; ok {
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return str
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

func mergeParameters(pathParams, opParams openapi3.Parameters) []*openapi3.ParameterRef {
	result := make([]*openapi3.ParameterRef, 0, len(pathParams)+len(opParams))
	result = append(result, pathParams...)
	result = append(result, opParams...)
	return result
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

func resolveParameter(doc *openapi3.T, ref *openapi3.ParameterRef) (*openapi3.Parameter, error) {
	if ref == nil {
		return nil, fmt.Errorf("parameter ref is nil")
	}
	if ref.Value != nil {
		return ref.Value, nil
	}
	if ref.Ref == "" {
		return nil, fmt.Errorf("parameter %v missing reference", ref)
	}
	name := strings.TrimPrefix(ref.Ref, "#/components/parameters/")
	if resolved, ok := doc.Components.Parameters[name]; ok && resolved != nil && resolved.Value != nil {
		return resolved.Value, nil
	}
	return nil, fmt.Errorf("parameter %s not found in components", ref.Ref)
}

func resolveRequestBody(doc *openapi3.T, ref *openapi3.RequestBodyRef) (*openapi3.RequestBody, error) {
	if ref == nil {
		return nil, nil
	}
	if ref.Value != nil {
		return ref.Value, nil
	}
	if ref.Ref == "" {
		return nil, fmt.Errorf("request body %v missing reference", ref)
	}
	name := strings.TrimPrefix(ref.Ref, "#/components/requestBodies/")
	if resolved, ok := doc.Components.RequestBodies[name]; ok && resolved != nil && resolved.Value != nil {
		return resolved.Value, nil
	}
	return nil, fmt.Errorf("request body %s not found in components", ref.Ref)
}

func (g *Generator) resolveType(schemaRef *openapi3.SchemaRef, required bool) typeInfo {
	if schemaRef == nil {
		return g.nullableType("JsonDocument", false, required)
	}
	if schemaRef.Ref != "" {
		name := componentName(schemaRef.Ref)
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
	if schemaRef.Value == nil {
		return g.nullableType("JsonDocument", false, required)
	}
	schema := schemaRef.Value
	if schema.Nullable {
		required = false
	}

	switch schema.Type {
	case "string":
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
	case "integer":
		typeName := "int"
		if schema.Format == "int64" {
			typeName = "long"
		}
		return g.nullableType(typeName, true, required)
	case "number":
		typeName := "decimal"
		if schema.Format == "float" {
			typeName = "float"
		}
		if schema.Format == "double" {
			typeName = "double"
		}
		return g.nullableType(typeName, true, required)
	case "boolean":
		return g.nullableType("bool", true, required)
	case "array":
		elem := g.resolveType(schema.Items, true)
		elemName := strings.TrimSuffix(elem.TypeName, "?")
		typeName := fmt.Sprintf("IEnumerable<%s>", elemName)
		return g.nullableType(typeName, false, required, true)
	case "object":
		if schema.AdditionalProperties.Schema != nil {
			valueType := g.resolveType(schema.AdditionalProperties.Schema, true)
			valueName := strings.TrimSuffix(valueType.TypeName, "?")
			typeName := fmt.Sprintf("IDictionary<string, %s>", valueName)
			return g.nullableType(typeName, false, required, true)
		}
		if len(schema.Properties) == 0 && len(schema.AllOf) == 0 {
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

func firstContentType(content openapi3.Content) string {
	for k := range content {
		return k
	}
	return ""
}

func preferredSchema(content openapi3.Content) *openapi3.SchemaRef {
	if media, ok := content["application/json"]; ok && media != nil && media.Schema != nil {
		return media.Schema
	}
	for _, media := range content {
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
}

type modelPropertyTemplateData struct {
	PropertyName     string
	JsonName         string
	TypeName         string
	Description      string
	Required         bool
	NeedsInitializer bool
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
