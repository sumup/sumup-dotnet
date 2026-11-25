package generator

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/pb33f/libopenapi"
)

func TestNestedReaderResources(t *testing.T) {
	t.Parallel()

	doc, err := libopenapi.NewDocument([]byte(`
openapi: 3.1.0
info: {title: Readers, version: 1.0.0}
tags:
  - name: Readers
  - name: Checkouts
  - name: ReaderCheckouts
    x-displayName: Checkouts
x-tagGroups:
  - name: Readers
    tags: [ReaderCheckouts]
paths:
  /readers/checkouts:
    get:
      operationId: ListReaders
      tags: [Readers]
      responses:
        '204': {description: OK}
    post:
      operationId: CreateReaderCheckout
      tags: [ReaderCheckouts]
      x-codegen: {method_name: create}
      responses:
        '204': {description: OK}
  /checkouts:
    get:
      operationId: ListCheckouts
      tags: [Checkouts]
      responses:
        '204': {description: OK}
components: {schemas: {}}
`))
	if err != nil {
		t.Fatal(err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatal(err)
	}

	generator := New(Config{Namespace: "SumUp"})
	clients, err := generator.buildClients(&model.Model)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(clients, func(client clientTemplateData) bool {
		return client.ClientName == "Checkouts" && client.Namespace == "SumUp.Checkouts" && client.Directory == "Checkouts"
	}) {
		t.Fatal("expected a namespace and directory for the standalone Checkouts client")
	}
	for _, client := range clients {
		if client.ClientName != "Readers" {
			continue
		}
		if len(client.Children) != 1 || len(client.Operations) != 1 || client.Operations[0].MethodName != "ListReaders" {
			t.Fatalf("expected the parent to expose Checkouts and its own operation: %+v", client)
		}
		child := client.Children[0]
		if client.Namespace != "SumUp.Readers" || child.Namespace != client.Namespace {
			t.Fatalf("expected one feature namespace, got parent %q and child %q", client.Namespace, child.Namespace)
		}
		if client.Directory != "Readers" || child.Directory != filepath.Join("Readers", "Checkouts") {
			t.Fatalf("expected resource folders, got parent %q and child %q", client.Directory, child.Directory)
		}
		if child.AccessPath != "Readers.Checkouts" || len(child.Operations) != 1 || child.Operations[0].MethodName != "Create" {
			t.Fatalf("expected only the checkout operation on the nested client: %+v", child)
		}
		return
	}
	t.Fatal("missing Readers client")
}
