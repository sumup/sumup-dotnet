package generator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

const sampleCatalogSchemaVersion = 1

// SampleCatalog is the versioned JSON contract consumed by documentation sites.
type SampleCatalog struct {
	SchemaVersion  int       `json:"schemaVersion"`
	Language       string    `json:"language"`
	SDK            SampleSDK `json:"sdk"`
	OpenAPIVersion string    `json:"openAPIVersion"`
	Samples        []Sample  `json:"samples"`
}

// SampleSDK identifies the package used by every generated sample.
type SampleSDK struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// Sample is a complete C# program for one OpenAPI operation example.
type Sample struct {
	ID          string `json:"id"`
	OperationID string `json:"operationId"`
	Example     string `json:"example,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	HTTPMethod  string `json:"httpMethod"`
	Path        string `json:"path"`
	Source      string `json:"sample"`
}

type requestExample struct {
	name        string
	summary     string
	description string
	json        string
}

// Samples builds a deterministic catalog of compile-ready C# programs.
func (g *Generator) Samples(doc *v3.Document, sdkVersion string) (*SampleCatalog, error) {
	if doc == nil || doc.Info == nil {
		return nil, fmt.Errorf("openapi document info is required")
	}
	if strings.TrimSpace(sdkVersion) == "" {
		return nil, fmt.Errorf("sdk version is required")
	}

	g.inlineModels = nil
	g.modelNames = map[string]struct{}{}
	g.errorModels = map[string]struct{}{}
	g.optionNames = map[string]struct{}{}
	if _, err := g.buildModels(doc); err != nil {
		return nil, fmt.Errorf("build models: %w", err)
	}
	clients, err := g.buildClients(doc)
	if err != nil {
		return nil, fmt.Errorf("build clients: %w", err)
	}

	samples := make([]Sample, 0)
	for _, client := range clients {
		for _, operation := range client.Operations {
			if operation.OperationID == "" {
				return nil, fmt.Errorf("missing operationId for %s %s", strings.ToUpper(operation.HttpMethod), operation.Path)
			}
			for _, example := range operation.RequestExamples {
				id := operation.OperationID
				if example.name != "" {
					id += "." + example.name
				}
				summary := strings.TrimSpace(operation.Summary)
				if example.summary != "" {
					summary = strings.TrimSpace(example.summary)
				}
				description := strings.TrimSpace(operation.Description)
				if example.description != "" {
					description = strings.TrimSpace(example.description)
				}
				samples = append(samples, Sample{
					ID:          id,
					OperationID: operation.OperationID,
					Example:     example.name,
					Summary:     summary,
					Description: description,
					HTTPMethod:  strings.ToUpper(operation.HttpMethod),
					Path:        operation.Path,
					Source:      g.renderSample(client, operation, example),
				})
			}
		}
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].ID < samples[j].ID })

	return &SampleCatalog{
		SchemaVersion: sampleCatalogSchemaVersion,
		Language:      "csharp",
		SDK: SampleSDK{
			Module:  "SumUp",
			Version: strings.TrimSpace(sdkVersion),
		},
		OpenAPIVersion: strings.TrimSpace(doc.Info.Version),
		Samples:        samples,
	}, nil
}

func (g *Generator) renderSample(client clientTemplateData, operation operationTemplateData, example requestExample) string {
	arguments := make([]string, 0, len(operation.PathParams)+2)
	for _, parameter := range operation.PathParams {
		arguments = append(arguments, g.sampleValue(parameter.TypeName, parameter.Name))
	}
	if operation.Body != nil {
		bodyType := strings.TrimSuffix(operation.Body.TypeName, "?")
		payload := example.json
		if payload == "" {
			payload = "{}"
		}
		arguments = append(arguments, fmt.Sprintf(
			"JsonSerializer.Deserialize<%s>(@\"%s\")!",
			bodyType,
			strings.ReplaceAll(payload, "\"", "\"\""),
		))
	}
	if operation.OperationOptions != nil {
		properties := make([]string, 0, len(operation.OperationOptions.Properties))
		for _, property := range operation.OperationOptions.Properties {
			properties = append(properties, fmt.Sprintf(
				"    %s = %s,",
				property.PropertyName,
				g.sampleValue(property.TypeName, property.PropertyName),
			))
		}
		arguments = append(arguments, fmt.Sprintf("new %s\n{\n%s\n}", operation.OperationOptions.Name, strings.Join(properties, "\n")))
	}

	call := fmt.Sprintf("client.%s.%sAsync()", client.PropertyName, operation.MethodName)
	if len(arguments) > 0 {
		indented := make([]string, len(arguments))
		for i, argument := range arguments {
			indented[i] = "            " + strings.ReplaceAll(argument, "\n", "\n            ")
		}
		call = fmt.Sprintf("client.%s.%sAsync(\n%s)", client.PropertyName, operation.MethodName, strings.Join(indented, ",\n"))
	}

	return fmt.Sprintf(`using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Threading.Tasks;
using SumUp;

public static class Program
{
    public static async Task Main()
    {
        using var client = new SumUpClient();
        var response = await %s;

        Console.WriteLine(response.StatusCode);
    }
}
`, call)
}

func (g *Generator) sampleValue(typeName, name string) string {
	typeName = strings.TrimSuffix(strings.TrimSpace(typeName), "?")
	if strings.HasPrefix(typeName, "OptionalQuery<") && strings.HasSuffix(typeName, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(typeName, "OptionalQuery<"), ">")
		return fmt.Sprintf("%s.From(%s)", typeName, g.sampleValue(inner, name))
	}
	if strings.HasPrefix(typeName, "IEnumerable<") && strings.HasSuffix(typeName, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(typeName, "IEnumerable<"), ">")
		return fmt.Sprintf("Array.Empty<%s>()", inner)
	}
	if member, ok := g.firstEnumMember(typeName); ok {
		return typeName + "." + member
	}

	switch typeName {
	case "string":
		return fmt.Sprintf("%q", sampleString(name))
	case "bool":
		return "true"
	case "byte[]":
		return "Array.Empty<byte>()"
	case "int":
		return "10"
	case "long":
		return "10L"
	case "decimal":
		return "10.00m"
	case "float":
		return "10.0f"
	case "double":
		return "10.0d"
	case "Guid":
		return `Guid.Parse("00000000-0000-0000-0000-000000000001")`
	case "DateOnly":
		return `DateOnly.Parse("2025-01-01")`
	case "TimeOnly":
		return `TimeOnly.Parse("12:00:00")`
	case "DateTimeOffset":
		return `DateTimeOffset.Parse("2025-01-01T12:00:00Z")`
	case "JsonDocument":
		return `JsonDocument.Parse("{}")`
	case "JsonObject":
		return `JsonObject.Parse("{}")`
	default:
		return fmt.Sprintf("default(%s)", typeName)
	}
}

func (g *Generator) firstEnumMember(typeName string) (string, bool) {
	for _, info := range g.schemaTypes {
		if info.TypeName != typeName || info.Kind != schemaKindEnum {
			continue
		}
		schema := g.schemaFromProxy(info.Schema)
		values := g.buildEnumValues(schema)
		if len(values) > 0 {
			return values[0].Name, true
		}
	}
	for _, model := range g.inlineModels {
		if model.Name == typeName && model.Kind == schemaKindEnum && len(model.EnumValues) > 0 {
			return model.EnumValues[0].Name, true
		}
	}
	return "", false
}

func sampleString(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "merchant") && strings.Contains(lower, "code"):
		return "your-merchant-code"
	case strings.Contains(lower, "reader") && strings.Contains(lower, "id"):
		return "your-reader-id"
	case strings.Contains(lower, "email"):
		return "merchant@example.com"
	case strings.Contains(lower, "url"):
		return "https://example.com/callback"
	case strings.Contains(lower, "id"):
		return "example-id"
	default:
		return "example"
	}
}

func requestExamples(operation *v3.Operation) []requestExample {
	if operation == nil || operation.RequestBody == nil || operation.RequestBody.Content == nil {
		return []requestExample{{}}
	}
	mediaType := preferredSampleMediaType(operation.RequestBody.Content)
	if mediaType == nil {
		return []requestExample{{}}
	}
	fallback := schemaSampleJSON(mediaType.Schema)
	if mediaType.Examples != nil && mediaType.Examples.Len() > 0 {
		names := make([]string, 0, mediaType.Examples.Len())
		for name := range mediaType.Examples.KeysFromOldest() {
			names = append(names, name)
		}
		sort.Strings(names)
		examples := make([]requestExample, 0, len(names))
		for _, name := range names {
			example := mediaType.Examples.GetOrZero(name)
			if example == nil {
				continue
			}
			value := nodeJSON(example.Value)
			if value == "" {
				value = fallback
			}
			examples = append(examples, requestExample{
				name:        name,
				summary:     example.Summary,
				description: example.Description,
				json:        value,
			})
		}
		if len(examples) > 0 {
			return examples
		}
	}
	if value := nodeJSON(mediaType.Example); value != "" {
		return []requestExample{{json: value}}
	}
	return []requestExample{{json: fallback}}
}

func preferredSampleMediaType(content *orderedmap.Map[string, *v3.MediaType]) *v3.MediaType {
	if content == nil {
		return nil
	}
	for contentType, mediaType := range content.FromOldest() {
		lower := strings.ToLower(contentType)
		if lower == "application/json" || strings.HasSuffix(lower, "+json") {
			return mediaType
		}
	}
	for _, mediaType := range content.FromOldest() {
		return mediaType
	}
	return nil
}

func schemaSampleJSON(proxy *base.SchemaProxy) string {
	value := schemaSampleValue(proxy, 0)
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func schemaSampleValue(proxy *base.SchemaProxy, depth int) any {
	if proxy == nil || depth > 6 {
		return map[string]any{}
	}
	schema := proxy.Schema()
	if schema == nil {
		return map[string]any{}
	}
	for _, node := range append(append([]*yaml.Node{}, schema.Examples...), schema.Example, schema.Default, schema.Const) {
		if value, ok := decodeNode(node); ok {
			return value
		}
	}
	if len(schema.Enum) > 0 {
		if value, ok := decodeNode(schema.Enum[0]); ok {
			return value
		}
	}
	if len(schema.AllOf) > 0 {
		merged := map[string]any{}
		for _, part := range schema.AllOf {
			if value, ok := schemaSampleValue(part, depth+1).(map[string]any); ok {
				for key, item := range value {
					merged[key] = item
				}
			}
		}
		if len(merged) > 0 {
			return merged
		}
	}
	if len(schema.OneOf) > 0 {
		return schemaSampleValue(schema.OneOf[0], depth+1)
	}
	if len(schema.AnyOf) > 0 {
		return schemaSampleValue(schema.AnyOf[0], depth+1)
	}
	if schemaHasType(schema, "array") && schema.Items != nil && schema.Items.IsA() {
		return []any{schemaSampleValue(schema.Items.A, depth+1)}
	}
	if schemaHasType(schema, "object") || (schema.Properties != nil && schema.Properties.Len() > 0) {
		value := map[string]any{}
		required := make(map[string]struct{}, len(schema.Required))
		for _, name := range schema.Required {
			required[name] = struct{}{}
		}
		if schema.Properties != nil {
			for name, property := range schema.Properties.FromOldest() {
				propertySchema := property.Schema()
				if propertySchema != nil && propertySchema.ReadOnly != nil && *propertySchema.ReadOnly {
					continue
				}
				if _, ok := required[name]; ok || schemaHasExample(propertySchema) {
					value[name] = schemaSampleValue(property, depth+1)
				}
			}
		}
		return value
	}
	switch {
	case schemaHasType(schema, "string"):
		return "example"
	case schemaHasType(schema, "integer"):
		return 1
	case schemaHasType(schema, "number"):
		return 10.0
	case schemaHasType(schema, "boolean"):
		return true
	default:
		return map[string]any{}
	}
}

func schemaHasExample(schema *base.Schema) bool {
	return schema != nil && (schema.Example != nil || len(schema.Examples) > 0 || schema.Default != nil || schema.Const != nil || len(schema.Enum) > 0)
}

func nodeJSON(node *yaml.Node) string {
	value, ok := decodeNode(node)
	if !ok {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func decodeNode(node *yaml.Node) (any, bool) {
	if node == nil {
		return nil, false
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, false
	}
	return value, true
}
