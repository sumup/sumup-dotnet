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
      "components":{"schemas":{
        "Widget":{"type":"object","properties":{"id":{"type":"string"}}},
        "WebhooksEvent":{"type":"object","properties":{"type":{"$ref":"#/components/schemas/EventType"}}},
        "EventType":{"type":"string","enum":["widgets.updated"]},
        "EventObjectType":{"type":"string","enum":["widget"]}
      }}
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
		for _, expected := range []string{"namespace Example;", "class WidgetUpdatedEvent : EventNotification<Widget>", "OnWidgetUpdated", `value == EventType.WidgetsUpdated => root.Deserialize<WidgetUpdatedEvent>()`, "Register(EventType.WidgetsUpdated, callback)", "Widget changed."} {
			if !strings.Contains(string(output), expected) {
				t.Errorf("missing %q in generated events", expected)
			}
		}
		for file, expected := range map[string]string{
			"WebhooksEvent.g.cs":   "public partial class WebhooksEvent",
			"EventType.g.cs":       "public readonly record struct EventType(string Value)",
			"EventObjectType.g.cs": "public readonly record struct EventObjectType(string Value)",
		} {
			model, err := os.ReadFile(filepath.Join(dir, "Models", file))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(model), expected) {
				t.Errorf("missing %q in generated %s", expected, file)
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
