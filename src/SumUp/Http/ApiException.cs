using System;
using System.Net;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;

namespace SumUp.Http;

/// <summary>
/// Raised when the SumUp API replies with a non-successful status code.
/// </summary>
public class ApiException : Exception
{
    public ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri)
        : this(statusCode, error, payload, requestUri, (HttpResponseHeaders?)null) { }

    public ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri, HttpResponseHeaders? headers)
        : this(statusCode, error, payload, requestUri, headers, ParseProblem(payload)) { }

    private ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri, HttpResponseHeaders? headers, Problem? problem)
        : this(statusCode, error, payload, requestUri, BuildMessage(statusCode, error, problem), headers)
    {
        Problem = problem;
    }

    /// <summary>Typed Problem details, when the API returned a Problem response.</summary>
    public virtual Problem? Problem { get; }

    public HttpResponseHeaders? Headers { get; }

    private static Problem? ParseProblem(string? payload)
    {
        if (string.IsNullOrEmpty(payload))
        {
            return null;
        }
        try
        {
            using var json = JsonDocument.Parse(payload);
            if (json.RootElement.ValueKind != JsonValueKind.Object ||
                !json.RootElement.TryGetProperty("type", out var type) || type.ValueKind != JsonValueKind.String)
            {
                return null;
            }
            return JsonSerializer.Deserialize<Problem>(payload);
        }
        catch (JsonException)
        {
            return null;
        }
    }

    public HttpStatusCode StatusCode { get; }

    public ApiError? Error { get; }

    public string? ResponseBody { get; }

    public Uri? RequestUri { get; }

    public override string ToString()
    {
        var builder = new StringBuilder(base.ToString());
        builder.AppendLine();
        AppendDetails(builder);
        return builder.ToString();
    }

    protected virtual void AppendDetails(StringBuilder builder)
    {
        builder.Append("StatusCode: ").Append((int)StatusCode).Append(' ').Append(StatusCode).AppendLine();
        if (RequestUri is not null)
        {
            builder.Append("RequestUri: ").Append(RequestUri).AppendLine();
        }
        if (!string.IsNullOrWhiteSpace(Error?.Code))
        {
            builder.Append("ErrorCode: ").Append(Error!.Code).AppendLine();
        }
        if (!string.IsNullOrWhiteSpace(Error?.Message))
        {
            builder.Append("ErrorMessage: ").Append(Error!.Message).AppendLine();
        }
        if (Error?.Details is JsonElement details)
        {
            builder.Append("ErrorDetails: ").Append(details.ToString()).AppendLine();
        }
        if (!string.IsNullOrWhiteSpace(ResponseBody))
        {
            builder.Append("ResponseBody: ").Append(ResponseBody).AppendLine();
        }
    }

    protected static string BuildMessage(HttpStatusCode statusCode, ApiError? error, Problem? problem)
    {
        var detail = !string.IsNullOrWhiteSpace(problem?.Detail) ? problem!.Detail : problem?.Title;
        if (!string.IsNullOrWhiteSpace(detail))
        {
            return $"SumUp API request failed with status code {(int)statusCode}. {detail}";
        }
        if (!string.IsNullOrWhiteSpace(error?.Code) || !string.IsNullOrWhiteSpace(error?.Message))
        {
            var code = string.IsNullOrWhiteSpace(error?.Code) ? "" : $" ({error!.Code})";
            var message = string.IsNullOrWhiteSpace(error?.Message) ? "" : $" {error!.Message}";
            return $"SumUp API request failed with status code {(int)statusCode}.{code}{message}";
        }

        return $"SumUp API request failed with status code {(int)statusCode}.";
    }

    protected ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri, string message)
        : this(statusCode, error, payload, requestUri, message, null) { }

    protected ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri, string message, HttpResponseHeaders? headers)
        : base(message)
    {
        StatusCode = statusCode;
        Error = error;
        ResponseBody = payload;
        RequestUri = requestUri;
        Headers = headers;
    }
}

public class ApiException<TError> : ApiException where TError : class
{
    public ApiException(HttpStatusCode statusCode, TError? error, string? payload, Uri? requestUri)
        : this(statusCode, error, payload, requestUri, (HttpResponseHeaders?)null) { }

    public ApiException(HttpStatusCode statusCode, TError? error, string? payload, Uri? requestUri, HttpResponseHeaders? headers)
        : base(statusCode, null, payload, requestUri, BuildMessage(statusCode, null, error as Problem), headers)
    {
        Error = error;
    }

    public new TError? Error { get; }

    public override Problem? Problem => Error is Problem { Type: not null } problem ? problem : null;

    protected override void AppendDetails(StringBuilder builder)
    {
        base.AppendDetails(builder);
        if (Error is not null)
        {
            builder.Append("Error: ").Append(Error).AppendLine();
        }
    }
}

public sealed class ApiError
{
    public string? Code { get; set; }

    public string? Message { get; set; }

    public JsonElement? Details { get; set; }
}
