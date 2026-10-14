package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestGenerateAnnotatedSchemas(t *testing.T) {
	t.Parallel()
	spec, err := os.ReadFile("testdata/schema-annotations.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := libopenapi.NewDocument(spec)
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	generator := New(Config{OutputDir: out, Namespace: "SumUp"})
	if err := generator.Run(&model.Model); err != nil {
		t.Fatal(err)
	}
	checks := map[string][]string{
		"Models/Address.g.cs":         {"string Country"},
		"Models/Merchant.g.cs":        {"string PhoneNumber", "Guid Id", "LegalType? LegalType"},
		"Models/ExtendedAddress.g.cs": {"string Country", "string? City", "string? Label"},
	}
	for file, snippets := range checks {
		data, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(data), snippet) {
				t.Errorf("%s missing %q:\n%s", file, snippet, data)
			}
		}
	}
}

func TestAnnotationNormalizationPreservesConstraints(t *testing.T) {
	t.Parallel()
	document, err := libopenapi.NewDocument([]byte(`{"openapi":"3.1.0","info":{"title":"test","version":"1"},"paths":{},"components":{"schemas":{"Code":{"type":"string"},"Constrained":{"allOf":[{"$ref":"#/components/schemas/Code"},{"minLength":2}]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	normalizeSchemaAnnotations(&model.Model)
	constrained := model.Model.Components.Schemas.GetOrZero("Constrained").Schema()
	if len(constrained.AllOf) != 2 {
		t.Fatal("normalization removed a validation constraint")
	}
}

func TestAnnotationNormalizationPreservesNamedReferences(t *testing.T) {
	t.Parallel()
	document, err := libopenapi.NewDocument([]byte(`{"openapi":"3.1.0","info":{"title":"test","version":"1"},"paths":{},"components":{"schemas":{"Code":{"type":"string"},"Alias":{"allOf":[{"description":"Alias documentation."},{"$ref":"#/components/schemas/Code"}]},"Holder":{"type":"object","properties":{"code":{"$ref":"#/components/schemas/Alias"}}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	normalizeSchemaAnnotations(&model.Model)
	property := model.Model.Components.Schemas.GetOrZero("Holder").Schema().Properties.GetOrZero("code")
	if got := property.GetReference(); got != "#/components/schemas/Alias" {
		t.Fatalf("reference = %q, want Alias", got)
	}
	if got := property.Schema().Description; got != "Alias documentation." {
		t.Fatalf("description = %q", got)
	}
}

func TestAnnotationNormalizationPreservesObjectDependencies(t *testing.T) {
	t.Parallel()
	document, err := libopenapi.NewDocument([]byte(`{"openapi":"3.1.0","info":{"title":"test","version":"1"},"paths":{},"components":{"schemas":{"Base":{"type":"object","properties":{"id":{"type":"string"}}},"Derived":{"allOf":[{"$ref":"#/components/schemas/Base","description":"Inherited fields."}]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}
	normalizeSchemaAnnotations(&model.Model)
	derived := model.Model.Components.Schemas.GetOrZero("Derived").Schema()
	if len(derived.AllOf) != 1 || derived.AllOf[0].GetReference() != "#/components/schemas/Base" {
		t.Fatal("object dependency was removed")
	}
}
