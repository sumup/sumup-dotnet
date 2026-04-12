package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/sumup/sumup-dotnet/codegen/internal/naming"
)

type eventTemplateData struct {
	Name, TypeLiteral, Model, Description string
}

func (g *Generator) renderEvents(t *template.Template, doc *v3.Document) error {
	events := []eventTemplateData{}
	if doc.Webhooks != nil {
		for eventType, path := range doc.Webhooks.FromOldest() {
			if path == nil || path.Post == nil {
				continue
			}
			op := path.Post
			var object struct {
				Ref string `yaml:"$ref"`
			}
			if op.Extensions == nil || op.Extensions.GetOrZero("x-object") == nil {
				return fmt.Errorf("event %s: missing x-object", eventType)
			}
			if err := op.Extensions.GetOrZero("x-object").Decode(&object); err != nil {
				return fmt.Errorf("event %s: decode x-object: %w", eventType, err)
			}
			model, ok := g.schemaTypes[componentName(object.Ref)]
			if !strings.HasPrefix(object.Ref, "#/components/schemas/") || !ok || model.Kind != schemaKindObject {
				return fmt.Errorf("event %s: invalid object reference %q", eventType, object.Ref)
			}
			name := naming.PascalIdentifier(strings.TrimSuffix(op.OperationId, "Webhook"))
			if name == "" {
				return fmt.Errorf("event %s: missing operation ID", eventType)
			}
			events = append(events, eventTemplateData{name, strconv.Quote(eventType), model.TypeName, sanitizeText(op.Description)})
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].TypeLiteral < events[j].TypeLiteral })
	var output bytes.Buffer
	data := struct {
		Namespace string
		Events    []eventTemplateData
	}{g.config.Namespace, events}
	if err := t.ExecuteTemplate(&output, "events.tmpl", data); err != nil {
		return fmt.Errorf("render events: %w", err)
	}
	if err := os.WriteFile(filepath.Join(g.config.OutputDir, "Events.g.cs"), output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write events: %w", err)
	}
	return nil
}
