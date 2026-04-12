package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEvents(t *testing.T) {
	t.Parallel()
	const spec = `{
      "openapi":"3.1.0", "info":{"title":"test","version":"1"}, "paths":{"/widgets":{"get":{"operationId":"listWidgets","tags":["Widgets"],"responses":{"200":{"description":"ok"}}}}},
      "webhooks":{"widgets.updated":{"post":{
        "operationId":"WidgetUpdatedWebhook", "description":"Widget changed.",
        "x-object":{"$ref":"#/components/schemas/Widget"},
        "responses":{"200":{"description":"ok"}}
      }}},
      "components":{"schemas":{"Widget":{"type":"object","properties":{"id":{"type":"string"}}}}}
    }`
	t.Run("generates types callbacks and dispatch from spec", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		g := New(Config{OutputDir: dir, Namespace: "Example"})
		doc := mustBuildV3Document(t, spec)
		if err := g.Run(doc); err != nil {
			t.Fatal(err)
		}
		output, err := os.ReadFile(filepath.Join(dir, "Events.g.cs"))
		if err != nil {
			t.Fatal(err)
		}
		for _, expected := range []string{"namespace Example;", "class WidgetUpdatedEvent : EventNotification<Widget>", "OnWidgetUpdated", `"widgets.updated" => root.Deserialize<WidgetUpdatedEvent>()`, "Widget changed."} {
			if !strings.Contains(string(output), expected) {
				t.Errorf("missing %q in generated events", expected)
			}
		}
		if err := g.Run(doc); err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(filepath.Join(dir, "Events.g.cs"))
		if err != nil {
			t.Fatal(err)
		}
		if string(output) != string(again) {
			t.Error("generation is not deterministic")
		}
	})
	t.Run("rejects missing resource references", func(t *testing.T) {
		t.Parallel()
		doc := mustBuildV3Document(t, strings.Replace(spec, "x-object", "x-missing", 1))
		err := New(Config{OutputDir: t.TempDir()}).Run(doc)
		if err == nil || !strings.Contains(err.Error(), "missing x-object") {
			t.Fatalf("got %v, want missing x-object", err)
		}
	})
}
