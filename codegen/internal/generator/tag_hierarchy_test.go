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
		if len(client.Children) != 1 || len(client.Operations) != 2 {
			t.Fatalf("expected the parent to expose Checkouts and retain both operations: %+v", client)
		}
		child := client.Children[0]
		if child.AccessPath != "Readers.Checkouts" || len(child.Operations) != 1 {
			t.Fatalf("expected only the checkout operation on the nested client: %+v", child)
		}
		return
	}
	t.Fatal("missing Readers client")
}
