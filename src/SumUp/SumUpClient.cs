using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using SumUp.Http;

namespace SumUp;

/// <summary>
/// Entry point for interacting with the SumUp API.
/// </summary>
public partial class SumUpClient : IDisposable
{
    private readonly HttpClient _httpClient;
    private readonly bool _ownsHttpClient;
    private readonly ApiClient _apiClient;
    private bool _disposed;

    /// <summary>
    /// Gets the configured options.
    /// </summary>
    public SumUpClientOptions Options { get; }

    public SumUpClient(SumUpClientOptions? options = null)
    {
        Options = options ?? SumUpClientOptions.FromEnvironment();

        _httpClient = Options.HttpClient ?? new HttpClient();
        _ownsHttpClient = Options.HttpClient is null;

        _httpClient.Timeout = Options.Timeout;
        _httpClient.BaseAddress ??= Options.BaseAddress;

        _apiClient = new ApiClient(_httpClient, Options);
        InitializeGeneratedClients(_apiClient);
    }

    partial void InitializeGeneratedClients(ApiClient apiClient);

    /// <summary>
    /// Creates a webhook handler bound to this client for typed event parsing and object fetches.
    /// </summary>
    /// <param name="secret">The webhook signing secret. Falls back to <see cref="WebhookConstants.SecretEnvironmentVariable"/> when omitted.</param>
    /// <param name="tolerance">The allowed clock skew when validating webhook timestamps.</param>
    public WebhookHandler CreateWebhookHandler(string? secret = null, TimeSpan? tolerance = null)
    {
        return new WebhookHandler(secret, tolerance, this);
    }

    internal TModel? FetchWebhookObject<TModel>(
        string absoluteUrl,
        RequestOptions? requestOptions = null,
        CancellationToken cancellationToken = default)
        where TModel : class
    {
        return _apiClient.GetAbsolute<TModel>(absoluteUrl, requestOptions, cancellationToken);
    }

    internal Task<TModel?> FetchWebhookObjectAsync<TModel>(
        string absoluteUrl,
        RequestOptions? requestOptions = null,
        CancellationToken cancellationToken = default)
        where TModel : class
    {
        return _apiClient.GetAbsoluteAsync<TModel>(absoluteUrl, requestOptions, cancellationToken);
    }

    public void Dispose()
    {
        if (_disposed)
        {
            return;
        }

        if (_ownsHttpClient)
        {
            _httpClient.Dispose();
        }

        _disposed = true;
        GC.SuppressFinalize(this);
    }
}
