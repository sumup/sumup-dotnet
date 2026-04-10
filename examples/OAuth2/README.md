# OAuth2 Example

This example demonstrates the SumUp OAuth2 Authorization Code flow with PKCE in a minimal local callback flow.

It uses `Duende.IdentityModel` for the OAuth2 protocol work. For production integrations, prefer a well-maintained OAuth2/OIDC library or your platform's standard authentication stack over a custom implementation.

## Prerequisites

Set the client credentials and redirect URI before running:

```sh
export CLIENT_ID="your_client_id"
export CLIENT_SECRET="your_client_secret"
export REDIRECT_URI="http://localhost:8080/callback"
```

## Run

```sh
dotnet run --project examples/OAuth2
```

The example starts a local HTTP listener for the callback, opens the browser to the SumUp authorization page, validates the returned `state`, exchanges the authorization code via `Duende.IdentityModel`, and then fetches the merchant data for the `merchant_code` returned in the callback.
