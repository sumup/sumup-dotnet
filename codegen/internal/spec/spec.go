package spec

import (
	"context"
	"fmt"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Load returns the parsed OpenAPI specification at the given path.
func Load(ctx context.Context, path string) (*v3.Document, error) {
	_ = ctx

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}

	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("build v3 model: %w", err)
	}

	return &model.Model, nil
}
