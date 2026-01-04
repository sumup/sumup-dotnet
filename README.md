# SumUp .NET SDK

This repository hosts the official .NET SDK for the SumUp API. The public surface is generated from the OpenAPI specification and wrapped in a modern HTTP client that prioritises ergonomics and resilience.

## Getting started

```sh
dotnet add package SumUp --prerelease
```

See `examples/BasicConsole` and `examples/CardReaderCheckout` for runnable projects.

## Supported .NET versions

We target every currently supported .NET release line. Continuous Integration runs restore/build/test on .NET 8.x, 9.x, and 10.x to ensure the SDK works across all supported versions.

Authenticate by exporting an access token (or by assigning `SumUpClientOptions.AccessToken` directly):

```sh
export SUMUP_ACCESS_TOKEN="my-token"
```

Then call the API:

```csharp
using SumUp;

using var client = new SumUpClient();
var response = await client.Checkouts.List();

foreach (var checkout in response.Data ?? Array.Empty<CheckoutSuccess>())
{
    Console.WriteLine($"Checkout {checkout.Id}: {checkout.Amount} {checkout.Currency}");
}
```

## Repository layout

- `src/SumUp` – the SDK (handwritten runtime + generated clients)
  - `*.g.cs` files under `src/SumUp` – per-tag clients (`CheckoutsClient`, `MerchantsClient`, …)
  - `src/SumUp/Models` – strongly typed request/response models from the spec
- `src/SumUp.Tests` – xUnit tests covering helpers and request builders
- `tools/codegen` – Go-based generator that transforms `openapi.json`
- `examples/BasicConsole` – console project demonstrating SDK usage

## Development

1. Install dependencies and create the generated clients:

   ```sh
   just generate
   ```

2. Run the usual validation pipeline locally prior to opening a PR:

   ```sh
   just ci
   ```

CI mirrors these steps via GitHub Actions (see `.github/workflows/ci.yml`).

## Code generation

The SDK clients live next to the handwritten runtime in `src/SumUp`, and the strongly typed models they depend on live in `src/SumUp/Models`. Both are generated through the Go CLI located in `tools/codegen`. You can regenerate them at any time with:

```sh
cd tools/codegen
go run ./cmd/sumup-dotnet \
  --spec ../../openapi.json \
  --output ../../src/SumUp \
  --namespace SumUp
```

The generator produces:

- One client per OpenAPI tag wired into `SumUpClient`.
- Request/response models (records, enums, dictionaries) sourced from `components.schemas`, so every method returns `ApiResponse<T>` where `T` is a concrete type such as `CheckoutSuccess`.

Custom runtime code (authentication, resiliency, convenience helpers) should be added under `src/SumUp` but outside of the generated `.g.cs` files.

## Examples

- `examples/BasicConsole` – lists recent checkouts to sanity check your API token.
- `examples/CardReaderCheckout` – mirrors the `../sumup-rs/examples/card_reader_checkout.rs` sample by listing the merchant’s paired readers and creating a €10 checkout on the first available device.

To run the card reader example:

```sh
export SUMUP_ACCESS_TOKEN="your_api_key"
export SUMUP_MERCHANT_CODE="your_merchant_code"
# Optional: set a specific reader, otherwise the first paired reader is chosen
# export SUMUP_READER_ID="your_reader_id"
dotnet run --project examples/CardReaderCheckout
```
