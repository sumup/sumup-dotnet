package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sumup/sumup-dotnet/codegen/internal/generator"
	"github.com/sumup/sumup-dotnet/codegen/internal/spec"
)

func main() {
	log.SetFlags(0)

	var (
		specPath string
		output   string
		ns       string
	)
	flag.StringVar(&specPath, "spec", "", "Path to the OpenAPI specification (JSON or YAML).")
	flag.StringVar(&output, "output", "src/SumUp", "Directory where generated files will be written.")
	flag.StringVar(&ns, "namespace", "SumUp", "Root namespace for generated code.")
	flag.Parse()

	if specPath == "" {
		log.Fatal("spec path is required (pass --spec)")
	}

	absSpec, err := filepath.Abs(specPath)
	if err != nil {
		log.Fatalf("resolve spec path: %v", err)
	}

	ctx := context.Background()
	doc, err := spec.Load(ctx, absSpec)
	if err != nil {
		log.Fatalf("load spec: %v", err)
	}

	outputDir := output
	if !filepath.IsAbs(outputDir) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("cwd: %v", err)
		}
		outputDir = filepath.Join(cwd, output)
	}

	gen := generator.New(generator.Config{
		OutputDir: outputDir,
		Namespace: ns,
	})

	if err := gen.Run(doc); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Printf("Generated SDK files at %s\n", outputDir)
}
