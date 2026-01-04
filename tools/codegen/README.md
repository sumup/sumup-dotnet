# sumup-dotnet codegen

This Go module transforms the SumUp OpenAPI specification into the generated portion of the .NET SDK.

## Usage

```sh
cd tools/codegen
go run ./cmd/sumup-dotnet \
  --spec ../../openapi.json \
  --output ../../src/SumUp \
  --namespace SumUp
```

The CLI accepts:

| Flag | Description |
| --- | --- |
| `--spec` | Path to the source OpenAPI document (`.json` or `.yaml`). |
| `--output` | Directory that will host the generated `.cs` files (existing `.g.cs` files inside are overwritten). |
| `--namespace` | Root C# namespace (defaults to `SumUp`). |

The generator emits:

- A client per OpenAPI tag (for example `CheckoutsClient`, `MerchantsClient`, …) in the configured root namespace (default `SumUp`).
- `SumUpClient.g.cs`, which wires the generated clients into the public `SumUpClient` partial class.
- POCO models and enums for every `components.schemas` entry, placed under `src/SumUp/Models`.
- XML docs sourced from the spec summaries / descriptions.

Each client method returns `Task<ApiResponse<T>>`, where `T` is the concrete response model inferred from the operation’s schema (falling back to `JsonDocument` only when the spec omits a schema). The plumbing around `ApiClient` and the handwritten surfaces live in `src/SumUp` and will evolve independently of this generator.
