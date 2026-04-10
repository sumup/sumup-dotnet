using System.Diagnostics;
using System.Net;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using Duende.IdentityModel;
using Duende.IdentityModel.Client;
using SumUp;

const string AuthorizationEndpoint = "https://api.sumup.com/authorize";
const string TokenEndpoint = "https://api.sumup.com/token";
const string Scopes = "email profile";

var clientId = GetRequiredEnvironmentVariable("CLIENT_ID");
var clientSecret = GetRequiredEnvironmentVariable("CLIENT_SECRET");
var redirectUri = Environment.GetEnvironmentVariable("REDIRECT_URI") ?? "http://localhost:8080/callback";

if (!Uri.TryCreate(redirectUri, UriKind.Absolute, out var redirectUriValue))
{
    throw new InvalidOperationException("REDIRECT_URI must be an absolute URL.");
}

var callbackPath = redirectUriValue.AbsolutePath;
var listenerPrefix = $"{redirectUriValue.Scheme}://{redirectUriValue.Authority}/";
var jsonOptions = new JsonSerializerOptions { WriteIndented = true };

// Create the state token and PKCE verifier before redirecting the user.
// When the browser comes back to our local callback URL, we validate the
// state value and use the verifier during the code exchange.
var state = GenerateUrlSafeRandomString();
var codeVerifier = GenerateCodeVerifier();
var authorizationUrl = BuildAuthorizationUrl(
    clientId,
    redirectUri,
    state,
    CreateCodeChallenge(codeVerifier));

using var listener = new HttpListener();
listener.Prefixes.Add(listenerPrefix);
listener.Start();

Console.WriteLine($"Listening on {listenerPrefix}");
Console.WriteLine($"Callback URL: {redirectUri}");
Console.WriteLine("Opening the browser for authorization...");
OpenBrowser(authorizationUrl);

using var httpClient = new HttpClient();
while (true)
{
    var context = await listener.GetContextAsync();
    var path = context.Request.Url?.AbsolutePath ?? string.Empty;

    if (!string.Equals(path, callbackPath, StringComparison.Ordinal))
    {
        context.Response.StatusCode = (int)HttpStatusCode.NotFound;
        await WriteHtmlAsync(context.Response, "<p>Not Found</p>");
        continue;
    }

    var code = context.Request.QueryString["code"];
    var returnedState = context.Request.QueryString["state"];
    var merchantCode = context.Request.QueryString["merchant_code"];

    if (string.IsNullOrWhiteSpace(code) || string.IsNullOrWhiteSpace(returnedState))
    {
        context.Response.StatusCode = (int)HttpStatusCode.BadRequest;
        await WriteHtmlAsync(context.Response, "<p>Missing OAuth callback parameters.</p>");
        break;
    }

    if (!string.Equals(returnedState, state, StringComparison.Ordinal))
    {
        context.Response.StatusCode = (int)HttpStatusCode.BadRequest;
        await WriteHtmlAsync(context.Response, "<p>Invalid OAuth state.</p>");
        break;
    }

    var token = await ExchangeTokenAsync(
        httpClient,
        clientId,
        clientSecret,
        redirectUri,
        code,
        codeVerifier);

    // SumUp returns the default merchant account in the callback.
    // A production integration may want to let the user choose among
    // all available merchants using the memberships API instead.
    if (string.IsNullOrWhiteSpace(merchantCode))
    {
        context.Response.StatusCode = (int)HttpStatusCode.BadRequest;
        await WriteHtmlAsync(context.Response, "<p>Missing merchant_code query parameter.</p>");
        break;
    }

    using var client = new SumUpClient(new SumUpClientOptions
    {
        AccessToken = token.AccessToken,
    });

    var merchantResponse = await client.Merchants.GetAsync(merchantCode);
    var merchant = merchantResponse.Data
        ?? throw new InvalidOperationException("Merchant response did not include any data.");

    Console.WriteLine($"Merchant code: {merchantCode}");
    Console.WriteLine(JsonSerializer.Serialize(merchant, jsonOptions));

    var merchantJson = JsonSerializer.Serialize(merchant, jsonOptions);
    var html = $"""
        <html>
          <body>
            <h1>OAuth2 Success</h1>
            <p>Access token obtained successfully.</p>
            <p><strong>Merchant code:</strong> {WebUtility.HtmlEncode(merchantCode)}</p>
            <h2>Merchant Information</h2>
            <pre>{WebUtility.HtmlEncode(merchantJson)}</pre>
          </body>
        </html>
        """;

    await WriteHtmlAsync(context.Response, html);
    break;
}

static string BuildAuthorizationUrl(
    string clientId,
    string redirectUri,
    string state,
    string codeChallenge)
{
    var requestUrl = new RequestUrl(AuthorizationEndpoint);
    return requestUrl.CreateAuthorizeUrl(
        clientId: clientId,
        responseType: "code",
        // Request only the scopes your application actually needs.
        scope: Scopes,
        redirectUri: redirectUri,
        state: state,
        codeChallenge: codeChallenge,
        codeChallengeMethod: "S256");
}

static async Task<TokenResponse> ExchangeTokenAsync(
    HttpClient httpClient,
    string clientId,
    string clientSecret,
    string redirectUri,
    string code,
    string codeVerifier)
{
    var response = await httpClient.RequestAuthorizationCodeTokenAsync(new AuthorizationCodeTokenRequest
    {
        Address = TokenEndpoint,
        ClientId = clientId,
        ClientSecret = clientSecret,
        Code = code,
        RedirectUri = redirectUri,
        CodeVerifier = codeVerifier,
    });

    if (response.IsError)
    {
        throw new InvalidOperationException($"Token exchange failed: {response.Error}");
    }

    return response;
}

static string CreateCodeChallenge(string codeVerifier)
{
    var verifierBytes = Encoding.ASCII.GetBytes(codeVerifier);
    var challengeBytes = SHA256.HashData(verifierBytes);
    return Base64UrlEncode(challengeBytes);
}

static string GenerateCodeVerifier()
{
    return GenerateUrlSafeRandomString(32);
}

static string GenerateUrlSafeRandomString(int length = 32)
{
    var buffer = RandomNumberGenerator.GetBytes(length);
    return Base64UrlEncode(buffer);
}

static string Base64UrlEncode(byte[] bytes)
{
    return Convert.ToBase64String(bytes)
        .TrimEnd('=')
        .Replace('+', '-')
        .Replace('/', '_');
}

static string GetRequiredEnvironmentVariable(string name)
{
    var value = Environment.GetEnvironmentVariable(name);
    if (string.IsNullOrWhiteSpace(value))
    {
        throw new InvalidOperationException($"Environment variable '{name}' must be set.");
    }

    return value;
}

static void OpenBrowser(string url)
{
    try
    {
        Process.Start(new ProcessStartInfo(url) { UseShellExecute = true });
    }
    catch (Exception ex)
    {
        Console.WriteLine($"Unable to open browser automatically: {ex.Message}");
        Console.WriteLine($"Open this URL manually: {url}");
    }
}

static async Task WriteHtmlAsync(HttpListenerResponse response, string html)
{
    var bytes = Encoding.UTF8.GetBytes(html);
    response.ContentType = "text/html; charset=utf-8";
    response.ContentLength64 = bytes.Length;
    await response.OutputStream.WriteAsync(bytes);
    response.Close();
}
