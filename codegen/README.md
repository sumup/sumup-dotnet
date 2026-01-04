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
