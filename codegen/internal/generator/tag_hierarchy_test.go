package generator

import (
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
	for _, client := range clients {
		if client.ClientName != "Readers" {
			continue
		}
		if len(client.Children) != 1 || len(client.Operations) != 1 || client.Operations[0].MethodName != "ListReaders" {
			t.Fatalf("expected the parent to expose Checkouts and its own operation: %+v", client)
		}
		child := client.Children[0]
		if child.AccessPath != "Readers.Checkouts" || len(child.Operations) != 1 || child.Operations[0].MethodName != "Create" {
			t.Fatalf("expected only the checkout operation on the nested client: %+v", child)
		}
		return
	}
	t.Fatal("missing Readers client")
}
