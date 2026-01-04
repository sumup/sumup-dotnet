<div align="center">

# SumUp .NET SDK

[![NuGet](https://img.shields.io/nuget/v/sumup.svg)](https://www.nuget.org/packages/SumUp/)
[![Documentation][docs-badge]](https://developer.sumup.com)
[![CI Status](https://github.com/sumup/sumup-dotnet/workflows/CI/badge.svg)](https://github.com/sumup/sumup-dotnet/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/sumup/sumup-py)](./LICENSE)

</div>

_**IMPORTANT:** This SDK is under development. We might still introduce minor breaking changes before reaching v1._

The .NET SDK for the SumUp [API](https://developer.sumup.com).

## Getting Started

```sh
dotnet add package SumUp --prerelease
```

See `examples/BasicConsole` and `examples/CardReaderCheckout` for runnable projects.

## Supported .NET Versions

We target every currently supported .NET release line. Continuous Integration runs restore/build/test on .NET 8.x, 9.x, and 10.x to ensure the SDK works across all supported versions.

Authenticate by exporting an access token (or by assigning `SumUpClientOptions.AccessToken` directly):

```sh
export SUMUP_ACCESS_TOKEN="my-token"
```

Then call the API:

```csharp
using System;
using SumUp;

using var client = new SumUpClient();
var response = await client.Checkouts.List();

foreach (var checkout in response.Data ?? Array.Empty<CheckoutSuccess>())
{
    Console.WriteLine($"Checkout {checkout.Id}: {checkout.Amount} {checkout.Currency}");
}
```

## Usage

### Creating a Checkout

```csharp
using System;
using SumUp;

using var client = new SumUpClient();

// Merchant profile contains the merchant code required when creating checkouts
var merchantResponse = await client.Merchant.GetAsync();
var merchantCode = merchantResponse.Data?.MerchantProfile?.MerchantCode
    ?? throw new InvalidOperationException("Merchant code not returned.");

var checkoutReference = $"checkout-{Guid.NewGuid():N}";

var checkoutResponse = await client.Checkouts.CreateAsync(new CheckoutCreateRequest
{
    Amount = 10.00f,
    Currency = Currency.Eur,
    CheckoutReference = checkoutReference,
    MerchantCode = merchantCode,
    Description = "Test payment",
    RedirectUrl = "https://example.com/success",
    ReturnUrl = "https://example.com/webhook",
});

Console.WriteLine($"Checkout ID: {checkoutResponse.Data?.Id}");
Console.WriteLine($"Checkout Reference: {checkoutResponse.Data?.CheckoutReference}");
```

### Creating a Reader Checkout

```csharp
using System;
using SumUp;

using var client = new SumUpClient();

var readerCheckout = await client.Readers.CreateCheckoutAsync(
    merchantCode: "your-merchant-code",
    readerId: "your-reader-id",
    body: new CreateReaderCheckoutRequest
    {
        Description = "Coffee purchase",
        ReturnUrl = "https://example.com/webhook",
        TotalAmount = new CreateReaderCheckoutRequestTotalAmount
        {
            Currency = "EUR",
            MinorUnit = 2,
            Value = 1000, // €10.00
        },
    });

Console.WriteLine($"Reader checkout created: {readerCheckout.Data?.Data?.ClientTransactionId}");
```

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

[docs-badge]: https://img.shields.io/badge/SumUp-documentation-white.svg?logo=data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgY29sb3I9IndoaXRlIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPgogICAgPHBhdGggZD0iTTIyLjI5IDBIMS43Qy43NyAwIDAgLjc3IDAgMS43MVYyMi4zYzAgLjkzLjc3IDEuNyAxLjcxIDEuN0gyMi4zYy45NCAwIDEuNzEtLjc3IDEuNzEtMS43MVYxLjdDMjQgLjc3IDIzLjIzIDAgMjIuMjkgMFptLTcuMjIgMTguMDdhNS42MiA1LjYyIDAgMCAxLTcuNjguMjQuMzYuMzYgMCAwIDEtLjAxLS40OWw3LjQ0LTcuNDRhLjM1LjM1IDAgMCAxIC40OSAwIDUuNiA1LjYgMCAwIDEtLjI0IDcuNjlabTEuNTUtMTEuOS03LjQ0IDcuNDVhLjM1LjM1IDAgMCAxLS41IDAgNS42MSA1LjYxIDAgMCAxIDcuOS03Ljk2bC4wMy4wM2MuMTMuMTMuMTQuMzUuMDEuNDlaIiBmaWxsPSJjdXJyZW50Q29sb3IiLz4KPC9zdmc+
