package spec

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// Load returns the parsed OpenAPI specification at the given path.
func Load(ctx context.Context, path string) (*openapi3.T, error) {
	loader := &openapi3.Loader{
		IsExternalRefsAllowed: true,
		Context:               ctx,
	}

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}

	return doc, nil
}
