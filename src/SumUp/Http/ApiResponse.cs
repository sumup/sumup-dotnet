using System;
using System.Net;
using System.Net.Http.Headers;

namespace SumUp.Http;

/// <summary>
/// Represents a SumUp API response.
/// </summary>
public sealed class ApiResponse<T>
{
    private ApiResponse(T? data, HttpStatusCode statusCode, HttpResponseHeaders headers, Uri? requestUri)
    {
        Data = data;
        StatusCode = statusCode;
        Headers = headers;
        RequestUri = requestUri;
    }

    public T? Data { get; }

    public HttpStatusCode StatusCode { get; }

    public HttpResponseHeaders Headers { get; }

    public Uri? RequestUri { get; }

    public bool IsSuccess => ((int)StatusCode is >= 200 and < 300);

    internal static ApiResponse<T> From(T? payload, HttpStatusCode statusCode, HttpResponseHeaders headers, Uri? requestUri)
        => new(payload, statusCode, headers, requestUri);
}
