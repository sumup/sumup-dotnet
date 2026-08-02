package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"github.com/sumup/sumup-dotnet/codegen/internal/generator"
	"github.com/sumup/sumup-dotnet/codegen/internal/spec"
)

var sdkVersionPattern = regexp.MustCompile(`<Version>\s*([^<]+?)\s*</Version>`)

func main() {
	log.SetFlags(0)
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, stdout io.Writer) error {
	if len(args) > 0 && args[0] == "samples" {
		return runSamples(args[1:], stdout)
	}
	return runSDK(args, stdout)
}

func runSDK(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("codegen", flag.ContinueOnError)
	flags.SetOutput(stdout)
	var specPath, output, namespace string
	flags.StringVar(&specPath, "spec", "", "Path to the OpenAPI specification (JSON or YAML).")
	flags.StringVar(&output, "output", "src/SumUp", "Directory where generated files will be written.")
	flags.StringVar(&namespace, "namespace", "SumUp", "Root namespace for generated code.")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if specPath == "" {
		return fmt.Errorf("spec path is required (pass --spec)")
	}

	doc, err := loadSpec(specPath)
	if err != nil {
		return err
	}
	outputDir, err := absolutePath(output)
	if err != nil {
		return err
	}
	gen := generator.New(generator.Config{OutputDir: outputDir, Namespace: namespace})
	if err := gen.Run(doc); err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	_, err = fmt.Fprintf(stdout, "Generated SDK files at %s\n", outputDir)
	return err
}

func runSamples(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("codegen samples", flag.ContinueOnError)
	flags.SetOutput(stdout)
	var specPath, output, sdkVersion, sdkVersionFile, namespace string
	flags.StringVar(&specPath, "spec", "", "Path to the OpenAPI specification (JSON or YAML).")
	flags.StringVar(&output, "output", "", "Path to the output JSON file (defaults to stdout).")
	flags.StringVar(&sdkVersion, "sdk-version", "", "SumUp .NET SDK version represented by the samples.")
	flags.StringVar(&sdkVersionFile, "sdk-version-file", "", "MSBuild project containing the SDK Version property.")
	flags.StringVar(&namespace, "namespace", "SumUp", "Root namespace used by generated samples.")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if specPath == "" {
		return fmt.Errorf("spec path is required (pass --spec)")
	}
	if sdkVersion == "" && sdkVersionFile != "" {
		version, err := readSDKVersion(sdkVersionFile)
		if err != nil {
			return err
		}
		sdkVersion = version
	}
	if sdkVersion == "" {
		return fmt.Errorf("sdk version is required (pass --sdk-version or --sdk-version-file)")
	}

	doc, err := loadSpec(specPath)
	if err != nil {
		return err
	}
	gen := generator.New(generator.Config{Namespace: namespace})
	catalog, err := gen.Samples(doc, sdkVersion)
	if err != nil {
		return fmt.Errorf("generate samples: %w", err)
	}
	encoded, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode samples: %w", err)
	}
	encoded = append(encoded, '\n')
	if output == "" {
		_, err = stdout.Write(encoded)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create sample output directory: %w", err)
	}
	if err := os.WriteFile(output, encoded, 0o644); err != nil {
		return fmt.Errorf("write samples: %w", err)
	}
	return nil
}

func loadSpec(path string) (*v3.Document, error) {
	absSpec, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve spec path: %w", err)
	}
	doc, err := spec.Load(context.Background(), absSpec)
	if err != nil {
		return nil, fmt.Errorf("load spec: %w", err)
	}
	return doc, nil
}

func absolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cwd: %w", err)
	}
	return filepath.Join(cwd, path), nil
}

func readSDKVersion(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read SDK version file: %w", err)
	}
	match := sdkVersionPattern.FindSubmatch(content)
	if len(match) != 2 {
		return "", fmt.Errorf("find Version property in %q", path)
	}
	return strings.TrimSpace(string(match[1])), nil
}
