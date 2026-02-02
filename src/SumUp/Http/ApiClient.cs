using System;
using System.IO;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading;
using System.Threading.Tasks;

namespace SumUp.Http;

internal sealed class ApiClient
{
    private readonly HttpClient _httpClient;
    private readonly SumUpClientOptions _options;
    private readonly JsonSerializerOptions _serializerOptions;

    internal HttpClient HttpClient => _httpClient;

    internal JsonSerializerOptions SerializerOptions => _serializerOptions;

    internal ApiClient(HttpClient httpClient, SumUpClientOptions options)
    {
        _httpClient = httpClient;
        _options = options;
        _serializerOptions = new JsonSerializerOptions(JsonSerializerDefaults.Web)
        {
            PropertyNameCaseInsensitive = true
        };
        _serializerOptions.Converters.Add(new JsonStringEnumConverter());
    }

    internal HttpRequestMessage CreateRequest(HttpMethod method, string pathTemplate, Action<RequestBuilder>? configure = null)
    {
        var builder = new RequestBuilder(method, pathTemplate, _httpClient.BaseAddress ?? _options.BaseAddress);
        configure?.Invoke(builder);
        var request = builder.Build();

        request.Headers.Accept.Clear();
        request.Headers.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
        request.Headers.UserAgent.ParseAdd(_options.UserAgent);
        RuntimeHeaders.Apply(request.Headers);

        return request;
    }

    internal HttpContent CreateContent(object body, string? contentType)
    {
        if (body is HttpContent httpContent)
        {
            return httpContent;
        }

        if (body is string text)
        {
            return new StringContent(text, Encoding.UTF8, contentType ?? "application/json");
        }

        if (body is JsonDocument document)
        {
            return new StringContent(document.RootElement.GetRawText(), Encoding.UTF8, contentType ?? "application/json");
        }

        if (body is Stream stream)
        {
            var streamContent = new StreamContent(stream);
            streamContent.Headers.ContentType = new MediaTypeHeaderValue(contentType ?? "application/octet-stream");
            return streamContent;
        }

        if (body is byte[] buffer)
        {
            var byteContent = new ByteArrayContent(buffer);
            byteContent.Headers.ContentType = new MediaTypeHeaderValue(contentType ?? "application/octet-stream");
            return byteContent;
        }

        var json = JsonSerializer.Serialize(body, _serializerOptions);
        return new StringContent(json, Encoding.UTF8, contentType ?? "application/json");
    }

    internal TModel? TryDeserialize<TModel>(string? payload)
    {
        if (string.IsNullOrEmpty(payload))
        {
            return default;
        }

        try
        {
            return JsonSerializer.Deserialize<TModel>(payload!, _serializerOptions);
        }
        catch
        {
            return default;
        }
    }

    internal static Task<string> ReadContentAsStringAsync(HttpContent content, CancellationToken cancellationToken)
    {
#if NETSTANDARD2_0
        cancellationToken.ThrowIfCancellationRequested();
        return content.ReadAsStringAsync();
#else
        return content.ReadAsStringAsync(cancellationToken);
#endif
    }

    internal static Task<Stream> ReadContentAsStreamAsync(HttpContent content, CancellationToken cancellationToken)
    {
#if NETSTANDARD2_0
        cancellationToken.ThrowIfCancellationRequested();
        return content.ReadAsStreamAsync();
#else
        return content.ReadAsStreamAsync(cancellationToken);
#endif
    }

    internal async Task ApplyAuthorizationHeaderAsync(
        HttpRequestMessage request,
        CancellationToken cancellationToken,
        RequestOptions? requestOptions)
    {
        if (requestOptions?.AccessToken is not null)
        {
            SetAuthorizationHeader(request, requestOptions.AccessToken);
            return;
        }

        if (request.Headers.Authorization is not null)
        {
            return;
        }

        var token = await _options.GetAccessTokenAsync(cancellationToken).ConfigureAwait(false);
        if (!string.IsNullOrWhiteSpace(token))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
        }
    }

    private static void SetAuthorizationHeader(HttpRequestMessage request, string token)
    {
        if (string.IsNullOrWhiteSpace(token))
        {
            request.Headers.Remove("Authorization");
            return;
        }

        request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
    }

    internal static CancellationToken CreateCancellationToken(
        CancellationToken cancellationToken,
        RequestOptions? requestOptions,
        out CancellationTokenSource? timeoutScope)
    {
        timeoutScope = null;
        if (requestOptions?.Timeout is not TimeSpan timeout)
        {
            return cancellationToken;
        }

        if (timeout == Timeout.InfiniteTimeSpan)
        {
            return cancellationToken;
        }

        if (timeout <= TimeSpan.Zero)
        {
            throw new ArgumentOutOfRangeException(nameof(RequestOptions.Timeout), "Timeout must be a positive duration.");
        }

        timeoutScope = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
        timeoutScope.CancelAfter(timeout);
        return timeoutScope.Token;
    }
}
