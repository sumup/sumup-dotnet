package generator

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Use the same model metadata as SDK generation so property names, inline
// models, enums, and request-specific types stay aligned with the public API.
type sampleRenderer struct {
	models map[string]modelTemplateData
}

func (r sampleRenderer) value(typeName string, value any) (string, error) {
	if value == nil {
		return "null", nil
	}
	typeName = strings.TrimSuffix(typeName, "?")
	if model, ok := r.models[typeName]; ok {
		if model.Kind == schemaKindEnum {
			for _, member := range model.EnumValues {
				if member.Value == fmt.Sprint(value) {
					return typeName + "." + member.Name, nil
				}
			}
			return "", fmt.Errorf("unknown %s value %v", typeName, value)
		}
		if model.IsDictionaryModel {
			return r.dictionary(typeName, model.DictionaryValueType, value)
		}
		object, ok := value.(map[string]any)
		if !ok {
			return "", fmt.Errorf("expected object for %s, got %T", typeName, value)
		}
		var entries []string
		known := make(map[string]bool, len(model.Properties))
		for _, property := range model.Properties {
			known[property.JsonName] = true
			item, exists := object[property.JsonName]
			if !exists || property.IsReadOnly {
				continue
			}
			expression, err := r.value(property.TypeName, item)
			if err != nil {
				return "", fmt.Errorf("render %s.%s: %w", typeName, property.PropertyName, err)
			}
			entries = append(entries, property.PropertyName+" = "+expression)
		}
		if model.HasExtensionData {
			extra := make(map[string]any)
			for key, item := range object {
				if !known[key] {
					extra[key] = item
				}
			}
			if len(extra) > 0 {
				expression, err := r.dictionary("Dictionary<string, "+model.ExtensionDataValueType+">", model.ExtensionDataValueType, extra)
				if err != nil {
					return "", err
				}
				entries = append(entries, "AdditionalProperties = "+expression)
			}
		}
		return sampleInitializer(typeName, entries), nil
	}
	if strings.HasPrefix(typeName, "IEnumerable<") {
		inner := strings.TrimSuffix(strings.TrimPrefix(typeName, "IEnumerable<"), ">")
		items, ok := value.([]any)
		if !ok {
			return "", fmt.Errorf("expected array for %s, got %T", typeName, value)
		}
		var entries []string
		for _, item := range items {
			expression, err := r.value(inner, item)
			if err != nil {
				return "", err
			}
			entries = append(entries, expression)
		}
		return sampleInitializer(inner+"[]", entries), nil
	}
	if strings.HasPrefix(typeName, "IDictionary<string, ") {
		inner := strings.TrimSuffix(strings.TrimPrefix(typeName, "IDictionary<string, "), ">")
		return r.dictionary("Dictionary<string, "+inner+">", inner, value)
	}
	switch typeName {
	case "JsonObject":
		return r.dictionary(typeName, "object?", value)
	case "object":
		switch value.(type) {
		case map[string]any:
			return r.value("JsonObject", value)
		case []any:
			return r.value("IEnumerable<object?>", value)
		case json.Number:
			return r.value("decimal", value)
		default:
			encoded, err := json.Marshal(value)
			return string(encoded), err
		}
	case "JsonDocument", "JsonElement":
		encoded, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		expression := "JsonDocument.Parse(" + sampleQuote(string(encoded)) + ")"
		if typeName == "JsonElement" {
			expression += ".RootElement.Clone()"
		}
		return expression, nil
	case "string":
		return sampleQuote(fmt.Sprint(value)), nil
	case "Guid", "DateOnly", "TimeOnly", "DateTimeOffset":
		return typeName + ".Parse(" + sampleQuote(fmt.Sprint(value)) + ")", nil
	case "byte[]":
		return "Convert.FromBase64String(" + sampleQuote(fmt.Sprint(value)) + ")", nil
	case "bool", "int", "long", "decimal", "float", "double":
		suffix := map[string]string{"long": "L", "decimal": "m", "float": "f", "double": "d"}[typeName]
		return fmt.Sprint(value) + suffix, nil
	default:
		return "", fmt.Errorf("unsupported sample type %s", typeName)
	}
}

func (r sampleRenderer) dictionary(typeName, valueType string, value any) (string, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return "", fmt.Errorf("expected dictionary for %s, got %T", typeName, value)
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		expression, err := r.value(valueType, object[key])
		if err != nil {
			return "", fmt.Errorf("render dictionary key %s: %w", key, err)
		}
		entries = append(entries, "["+sampleQuote(key)+"] = "+expression)
	}
	return sampleInitializer(typeName, entries), nil
}

func sampleInitializer(typeName string, entries []string) string {
	if len(entries) == 0 {
		return "new " + typeName + " { }"
	}
	return "new " + typeName + "\n{\n    " + strings.ReplaceAll(strings.Join(entries, ",\n"), "\n", "\n    ") + ",\n}"
}

func sampleQuote(value string) string {
	// JSON string escaping is also valid in ordinary C# string literals.
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
