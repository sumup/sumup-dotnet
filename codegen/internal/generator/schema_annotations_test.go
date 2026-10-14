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
		"Models/Merchant.g.cs":        {"string PhoneNumber"},
		"Models/ExtendedAddress.g.cs": {"string Country", "string? Label"},
	}
	for file, snippets := range checks {
		data, err := os.ReadFile(filepath.Join(out, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(data), snippet) {
				t.Errorf("%s missing %q", file, snippet)
			}
		}
	}
}
