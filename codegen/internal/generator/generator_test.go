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

func TestBuildModels_UsesJsonObjectForFreeFormObjects(t *testing.T) {
	const spec = `{
	  "openapi": "3.0.3",
	  "info": {
	    "title": "test",
	    "version": "1.0.0"
	  },
	  "paths": {},
	  "components": {
	    "schemas": {
	      "Metadata": {
	        "type": "object",
	        "additionalProperties": true
	      },
	      "PaymentPayload": {
	        "type": "object",
	        "properties": {
	          "metadata": {
	            "type": "object",
	            "additionalProperties": true
	          },
	          "apple_pay": {
	            "type": "object"
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

	metadata := findModel(t, models, "Metadata")
	if !metadata.IsDictionaryModel {
		t.Fatalf("Metadata should be a dictionary-backed model")
	}
	if metadata.DictionaryBaseType != "JsonObject" {
		t.Fatalf("Metadata dictionary base type = %q, want %q", metadata.DictionaryBaseType, "JsonObject")
	}

	payload := findModel(t, models, "PaymentPayload")
	if got := propertyType(payload.Properties, "Metadata"); got != "JsonObject?" {
		t.Fatalf("Metadata property type = %q, want %q", got, "JsonObject?")
	}
	if got := propertyType(payload.Properties, "ApplePay"); got != "JsonObject?" {
		t.Fatalf("ApplePay property type = %q, want %q", got, "JsonObject?")
	}
}

func TestBuildClients_UsesJsonDocumentForOpaqueObjectResponses(t *testing.T) {
	const spec = `{
	  "openapi": "3.0.3",
	  "info": {
	    "title": "test",
	    "version": "1.0.0"
	  },
	  "paths": {
	    "/v0.2/checkouts/{id}/apple-pay-session": {
	      "put": {
	        "tags": ["Checkouts"],
	        "operationId": "CreateApplePaySession",
	        "parameters": [
	          {
	            "name": "id",
	            "in": "path",
	            "required": true,
	            "schema": { "type": "string" }
	          }
	        ],
	        "responses": {
	          "200": {
	            "description": "ok",
	            "content": {
	              "application/json": {
	                "schema": { "type": "object" }
	              }
	            }
	          }
	        }
	      }
	    }
	  }
	}`

	doc := mustBuildV3Document(t, spec)

	g := New(Config{Namespace: "SumUp"})
	clients, err := g.buildClients(doc)
	if err != nil {
		t.Fatalf("buildClients() error = %v", err)
	}

	if len(clients) != 1 || len(clients[0].Operations) != 1 {
		t.Fatalf("unexpected client/operation count")
	}

	operation := clients[0].Operations[0]
	if operation.ResponseType != "JsonDocument" {
		t.Fatalf("response type = %q, want %q", operation.ResponseType, "JsonDocument")
	}
	if operation.ResponseMode != "json-document" {
		t.Fatalf("response mode = %q, want %q", operation.ResponseMode, "json-document")
	}
}

func TestBuildClients_UsesOperationOptionsForQueryParameters(t *testing.T) {
	const spec = `{
	  "openapi": "3.0.3",
	  "info": {
	    "title": "test",
	    "version": "1.0.0"
	  },
	  "paths": {
	    "/v0.1/merchants/{merchant_code}/transactions/history": {
	      "get": {
	        "tags": ["Transactions"],
	        "operationId": "ListHistory",
	        "parameters": [
	          {
	            "name": "merchant_code",
	            "in": "path",
	            "required": true,
	            "schema": { "type": "string" }
	          },
	          {
	            "name": "order",
	            "in": "query",
	            "schema": { "type": "string" }
	          },
	          {
	            "name": "limit",
	            "in": "query",
	            "schema": { "type": "integer" }
	          }
	        ],
	        "responses": {
	          "200": {
	            "description": "ok",
	            "content": {
	              "application/json": {
	                "schema": {
	                  "type": "object"
	                }
	              }
	            }
	          }
	        }
	      }
	    }
	  }
	}`

	doc := mustBuildV3Document(t, spec)

	g := New(Config{Namespace: "SumUp"})
	clients, err := g.buildClients(doc)
	if err != nil {
		t.Fatalf("buildClients() error = %v", err)
	}

	if len(clients) != 1 {
		t.Fatalf("buildClients() returned %d clients, want 1", len(clients))
	}

	operation := clients[0].Operations[0]
	if !operation.HasOperationOptions {
		t.Fatalf("operation should use an options model")
	}
	if operation.OperationOptions == nil {
		t.Fatalf("operation options should not be nil")
	}
	if operation.OperationOptions.Name != "TransactionsListHistoryOptions" {
		t.Fatalf("options model name = %q, want %q", operation.OperationOptions.Name, "TransactionsListHistoryOptions")
	}
	if len(operation.Parameters) != 2 {
		t.Fatalf("method parameter count = %d, want 2", len(operation.Parameters))
	}
	if operation.Parameters[0].Name != "merchantCode" {
		t.Fatalf("first parameter name = %q, want %q", operation.Parameters[0].Name, "merchantCode")
	}
	if operation.Parameters[1].Name != "options" {
		t.Fatalf("second parameter name = %q, want %q", operation.Parameters[1].Name, "options")
	}
	if operation.Parameters[1].Signature != "TransactionsListHistoryOptions? options = null" {
		t.Fatalf("options signature = %q, want %q", operation.Parameters[1].Signature, "TransactionsListHistoryOptions? options = null")
	}
	if operation.QueryParams[0].OptionsBuilderCall != "builder.AddQuery(\"order\", operationOptions.Order);" {
		t.Fatalf("order builder call = %q", operation.QueryParams[0].OptionsBuilderCall)
	}
	if operation.QueryParams[1].OptionsBuilderCall != "builder.AddQuery(\"limit\", operationOptions.Limit);" {
		t.Fatalf("limit builder call = %q", operation.QueryParams[1].OptionsBuilderCall)
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
