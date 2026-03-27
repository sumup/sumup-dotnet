package generator

import (
	"testing"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func TestBuildModels_GeneratesInlineEnumForLinksRelation(t *testing.T) {
	const spec = `{
	  "openapi": "3.0.3",
	  "info": {
	    "title": "test",
	    "version": "1.0.0"
	  },
	  "paths": {},
	  "components": {
	    "schemas": {
	      "TransactionsListResponse": {
	        "type": "object",
	        "properties": {
	          "links": {
	            "type": "array",
	            "items": {
	              "type": "object",
	              "properties": {
	                "rel": {
	                  "type": "string",
	                  "enum": ["next", "previous"]
	                },
	                "href": {
	                  "type": "string"
	                }
	              },
	              "required": ["rel", "href"]
	            }
	          }
	        }
	      }
	    }
	  }
	}`

	doc := mustBuildV3Document(t, spec)

	g := New(Config{Namespace: "SumUp"})
	models, err := g.buildModels(doc)
	if err != nil {
		t.Fatalf("buildModels() error = %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("buildModels() returned %d top-level models, want 1", len(models))
	}

	response := models[0]
	if response.Name != "TransactionsListResponse" {
		t.Fatalf("top-level model name = %q, want %q", response.Name, "TransactionsListResponse")
	}

	if got := propertyType(response.Properties, "Links"); got != "IEnumerable<TransactionsListResponseLinksItem>?" {
		t.Fatalf("Links property type = %q, want %q", got, "IEnumerable<TransactionsListResponseLinksItem>?")
	}

	linkItem := findModel(t, g.inlineModels, "TransactionsListResponseLinksItem")
	if got := propertyType(linkItem.Properties, "Rel"); got != "TransactionsListResponseLinksItemRel" {
		t.Fatalf("Rel property type = %q, want %q", got, "TransactionsListResponseLinksItemRel")
	}

	relEnum := findModel(t, g.inlineModels, "TransactionsListResponseLinksItemRel")
	if relEnum.Kind != schemaKindEnum {
		t.Fatalf("inline rel model kind = %v, want enum", relEnum.Kind)
	}
	if len(relEnum.EnumValues) != 2 {
		t.Fatalf("inline rel enum values = %d, want 2", len(relEnum.EnumValues))
	}
	if relEnum.EnumValues[0].Value != "next" || relEnum.EnumValues[1].Value != "previous" {
		t.Fatalf("inline rel enum values = %#v, want next/previous", relEnum.EnumValues)
	}
}

func mustBuildV3Document(t *testing.T, raw string) *v3.Document {
	t.Helper()

	document, err := libopenapi.NewDocument([]byte(raw))
	if err != nil {
		t.Fatalf("NewDocument() error = %v", err)
	}

	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatalf("BuildV3Model() error = %v", err)
	}

	return &model.Model
}

func findModel(t *testing.T, models []modelTemplateData, name string) modelTemplateData {
	t.Helper()

	for _, model := range models {
		if model.Name == name {
			return model
		}
	}

	t.Fatalf("model %q not found", name)
	return modelTemplateData{}
}

func propertyType(properties []modelPropertyTemplateData, name string) string {
	for _, property := range properties {
		if property.PropertyName == name {
			return property.TypeName
		}
	}

	return ""
}
