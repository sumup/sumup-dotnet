using System;
using System.Net;
using System.Text.Json;

namespace SumUp.Http;

/// <summary>
/// Raised when the SumUp API replies with a non-successful status code.
/// </summary>
public sealed class ApiException : Exception
{
    public ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri)
        : base(error?.Message ?? $"SumUp API request failed with status code {(int)statusCode}.")
    {
        StatusCode = statusCode;
        Error = error;
        ResponseBody = payload;
        RequestUri = requestUri;
    }

    public HttpStatusCode StatusCode { get; }

    public ApiError? Error { get; }

    public string? ResponseBody { get; }

    public Uri? RequestUri { get; }
}

public sealed class ApiError
{
    public string? Code { get; set; }

    public string? Message { get; set; }

    public JsonElement? Details { get; set; }
}
