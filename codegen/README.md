# sumup-dotnet codegen

Tiny Go tool that transforms the SumUp OpenAPI specification into the generated portion of the .NET SDK.

## Usage

```sh
cd codegen
go run ./... \
  --spec ../openapi.json \
  --output ../src/SumUp \
  --namespace SumUp
```

The CLI accepts:

| Flag | Description |
| --- | --- |
| `--spec` | Path to the source OpenAPI document (`.json` or `.yaml`). |
| `--output` | Directory that will host the generated `.cs` files (existing `.g.cs` files inside are overwritten). |
| `--namespace` | Root C# namespace (defaults to `SumUp`). |

## Generate code samples

Generate the deterministic, versioned catalog of complete C# programs from the repository root:

```sh
just generate-codesamples
```

The command writes `code-samples.json` at the repository root. Override the destination when needed:

```sh
just generate-codesamples /tmp/dotnet.json
```

The catalog is generated from the same OpenAPI model and SDK method signatures as the client. Every emitted program is compiled against the current SDK by the codegen test suite.

The equivalent direct command is:

```sh
cd codegen
go run . samples \
  --spec ../openapi.json \
  --sdk-version-file ../src/SumUp/SumUp.csproj \
  --output ../code-samples.json
```
