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

        return request;
    }

    internal async Task<ApiResponse<T>> SendAsync<T>(
        HttpRequestMessage request,
        object? body,
        string? contentType,
        CancellationToken cancellationToken)
    {
        var token = await _options.GetAccessTokenAsync(cancellationToken).ConfigureAwait(false);
        if (!string.IsNullOrWhiteSpace(token))
        {
            request.Headers.Authorization = new AuthenticationHeaderValue("Bearer", token);
        }

        if (body is not null && request.Content is null)
        {
            request.Content = CreateContent(body, contentType);
        }

        using var response = await _httpClient.SendAsync(
            request,
            HttpCompletionOption.ResponseHeadersRead,
            cancellationToken).ConfigureAwait(false);

        if (!response.IsSuccessStatusCode)
        {
            var responseBody = response.Content is null
                ? null
                : await ReadContentAsStringAsync(response.Content, cancellationToken).ConfigureAwait(false);

            ApiError? error = null;
            if (!string.IsNullOrEmpty(responseBody))
            {
                error = TryDeserialize<ApiError>(responseBody!);
            }

            throw new ApiException(response.StatusCode, error, responseBody, response.RequestMessage?.RequestUri);
        }

        if (response.Content == null || response.Content.Headers.ContentLength == 0)
        {
            return ApiResponse<T>.From(default, response.StatusCode, response.Headers, response.RequestMessage?.RequestUri);
        }

        if (typeof(T) == typeof(JsonDocument))
        {
            using var jsonStream = await ReadContentAsStreamAsync(response.Content, cancellationToken).ConfigureAwait(false);
            var document = await JsonDocument.ParseAsync(jsonStream, cancellationToken: cancellationToken).ConfigureAwait(false);
            return ApiResponse<T>.From((T)(object)document, response.StatusCode, response.Headers, response.RequestMessage?.RequestUri);
        }

        if (typeof(T) == typeof(string))
        {
            var text = await ReadContentAsStringAsync(response.Content, cancellationToken).ConfigureAwait(false);
            return ApiResponse<T>.From((T)(object)text, response.StatusCode, response.Headers, response.RequestMessage?.RequestUri);
        }

        using var stream = await ReadContentAsStreamAsync(response.Content, cancellationToken).ConfigureAwait(false);
        var result = await JsonSerializer.DeserializeAsync<T>(stream, _serializerOptions, cancellationToken).ConfigureAwait(false);
        return ApiResponse<T>.From(result, response.StatusCode, response.Headers, response.RequestMessage?.RequestUri);
    }

    internal ApiResponse<T> Send<T>(
        HttpRequestMessage request,
        object? body,
        string? contentType,
        CancellationToken cancellationToken)
    {
        return SendAsync<T>(request, body, contentType, cancellationToken).GetAwaiter().GetResult();
    }

    private HttpContent CreateContent(object body, string? contentType)
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

    private TModel? TryDeserialize<TModel>(string payload)
    {
        try
        {
            return JsonSerializer.Deserialize<TModel>(payload, _serializerOptions);
        }
        catch
        {
            return default;
        }
    }

    private static Task<string> ReadContentAsStringAsync(HttpContent content, CancellationToken cancellationToken)
    {
#if NETSTANDARD2_0
        cancellationToken.ThrowIfCancellationRequested();
        return content.ReadAsStringAsync();
#else
        return content.ReadAsStringAsync(cancellationToken);
#endif
    }

    private static Task<Stream> ReadContentAsStreamAsync(HttpContent content, CancellationToken cancellationToken)
    {
#if NETSTANDARD2_0
        cancellationToken.ThrowIfCancellationRequested();
        return content.ReadAsStreamAsync();
#else
        return content.ReadAsStreamAsync(cancellationToken);
#endif
    }
}
