package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/sumup/sumup-dotnet/codegen/internal/spec"
	"go.yaml.in/yaml/v4"
)

func TestSamples(t *testing.T) {
	repositoryRoot, catalog := testSampleCatalog(t)
	if catalog.SchemaVersion != 1 {
		t.Fatalf("SchemaVersion = %d, want 1", catalog.SchemaVersion)
	}
	if catalog.Language != "csharp" {
		t.Fatalf("Language = %q, want csharp", catalog.Language)
	}
	if catalog.SDK.Module != "SumUp" {
		t.Fatalf("SDK.Module = %q, want SumUp", catalog.SDK.Module)
	}
	if catalog.OpenAPIVersion != "1.0.0" {
		t.Fatalf("OpenAPIVersion = %q, want 1.0.0", catalog.OpenAPIVersion)
	}
	if len(catalog.Samples) == 0 {
		t.Fatal("catalog contains no samples")
	}
	if !sort.SliceIsSorted(catalog.Samples, func(i, j int) bool {
		return catalog.Samples[i].ID < catalog.Samples[j].ID
	}) {
		t.Fatal("samples are not sorted by ID")
	}

	seen := make(map[string]struct{}, len(catalog.Samples))
	for _, sample := range catalog.Samples {
		if _, exists := seen[sample.ID]; exists {
			t.Fatalf("duplicate sample ID %q", sample.ID)
		}
		seen[sample.ID] = struct{}{}
		if !strings.Contains(sample.Source, "public static async Task Main()") {
			t.Errorf("sample %q is not a complete program", sample.ID)
		}
	}

	createCheckout := sampleByID(t, catalog.Samples, "CreateCheckout.HostedCheckout")
	if !strings.Contains(createCheckout.Source, "client.Checkouts.CreateAsync(") {
		t.Fatalf("CreateCheckout sample does not call the generated SDK method:\n%s", createCheckout.Source)
	}
	if !strings.Contains(createCheckout.Source, `b50pr914-6k0e-3091-a592-890010285b3d`) {
		t.Fatalf("CreateCheckout sample does not preserve the OpenAPI example:\n%s", createCheckout.Source)
	}
	encoded, err := json.Marshal(createCheckout)
	if err != nil {
		t.Fatalf("marshal sample: %v", err)
	}
	if !strings.Contains(string(encoded), `"sample":`) || strings.Contains(string(encoded), `"source":`) {
		t.Fatalf("sample JSON does not preserve the portal contract: %s", encoded)
	}

	compileSamples(t, repositoryRoot, catalog.Samples)
}

func TestSamplesDeterministic(t *testing.T) {
	_, first := testSampleCatalog(t)
	_, second := testSampleCatalog(t)
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first catalog: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second catalog: %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("sample generation is not deterministic")
	}
}

func TestRequestExamplesPreserveWholeRequestExample(t *testing.T) {
	var node yaml.Node
	if err := node.Encode(map[string]any{"selected": "request-example"}); err != nil {
		t.Fatalf("encode request example: %v", err)
	}
	content := orderedmap.New[string, *v3.MediaType]()
	content.Set("application/json", &v3.MediaType{Example: &node})
	operation := &v3.Operation{RequestBody: &v3.RequestBody{Content: content}}

	examples := requestExamples(operation)
	if len(examples) != 1 || examples[0].json != `{"selected":"request-example"}` {
		t.Fatalf("request example was expanded with schema values: %#v", examples)
	}
}

func testSampleCatalog(t *testing.T) (string, *SampleCatalog) {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	doc, err := spec.Load(t.Context(), filepath.Join(repositoryRoot, "openapi.json"))
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	catalog, err := New(Config{Namespace: "SumUp"}).Samples(doc, "test")
	if err != nil {
		t.Fatalf("generate samples: %v", err)
	}
	return repositoryRoot, catalog
}

func sampleByID(t *testing.T, samples []Sample, id string) Sample {
	t.Helper()
	for _, sample := range samples {
		if sample.ID == id {
			return sample
		}
	}
	t.Fatalf("sample %q not found", id)
	return Sample{}
}

func compileSamples(t *testing.T, repositoryRoot string, samples []Sample) {
	t.Helper()
	if _, err := exec.LookPath("dotnet"); err != nil {
		t.Fatalf("dotnet is required to compile generated samples: %v", err)
	}
	dir := t.TempDir()
	project := fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <OutputType>Library</OutputType>
    <Nullable>enable</Nullable>
    <TreatWarningsAsErrors>true</TreatWarningsAsErrors>
  </PropertyGroup>
  <ItemGroup>
    <ProjectReference Include="%s" />
  </ItemGroup>
</Project>
`, filepath.Join(repositoryRoot, "src", "SumUp", "SumUp.csproj"))
	if err := os.WriteFile(filepath.Join(dir, "GeneratedSamples.csproj"), []byte(project), 0o600); err != nil {
		t.Fatalf("write sample project: %v", err)
	}
	for i, sample := range samples {
		className := fmt.Sprintf("Sample%03d", i)
		source := strings.Replace(sample.Source, "public static class Program", "public static class "+className, 1)
		if err := os.WriteFile(filepath.Join(dir, className+".cs"), []byte(source), 0o600); err != nil {
			t.Fatalf("write sample %q: %v", sample.ID, err)
		}
	}

	command := exec.CommandContext(t.Context(), "dotnet", "build", "GeneratedSamples.csproj", "--nologo", "--verbosity", "quiet")
	command.Dir = dir
	command.Env = append(os.Environ(), "DOTNET_CLI_TELEMETRY_OPTOUT=1", "DOTNET_NOLOGO=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compile generated samples: %v\n%s", err, output)
	}
}
