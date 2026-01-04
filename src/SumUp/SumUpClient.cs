using System;
using System.Net.Http;
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
