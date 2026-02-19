using System;
using System.Net;
using System.Text;
using System.Text.Json;

namespace SumUp.Http;

/// <summary>
/// Raised when the SumUp API replies with a non-successful status code.
/// </summary>
public class ApiException : Exception
{
    public ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri)
        : this(statusCode, error, payload, requestUri, BuildMessage(statusCode, error))
    {
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

    private static string BuildMessage(HttpStatusCode statusCode, ApiError? error)
    {
        if (!string.IsNullOrWhiteSpace(error?.Code) || !string.IsNullOrWhiteSpace(error?.Message))
        {
            var code = string.IsNullOrWhiteSpace(error?.Code) ? "" : $" ({error!.Code})";
            var message = string.IsNullOrWhiteSpace(error?.Message) ? "" : $" {error!.Message}";
            return $"SumUp API request failed with status code {(int)statusCode}.{code}{message}";
        }

        return $"SumUp API request failed with status code {(int)statusCode}.";
    }

    protected ApiException(HttpStatusCode statusCode, ApiError? error, string? payload, Uri? requestUri, string message)
        : base(message)
    {
        StatusCode = statusCode;
        Error = error;
        ResponseBody = payload;
        RequestUri = requestUri;
    }
}

public class ApiException<TError> : ApiException where TError : class
{
    public ApiException(HttpStatusCode statusCode, TError? error, string? payload, Uri? requestUri)
        : base(statusCode, null, payload, requestUri, BuildMessage(statusCode))
    {
        Error = error;
    }

    public new TError? Error { get; }

    protected override void AppendDetails(StringBuilder builder)
    {
        base.AppendDetails(builder);
        if (Error is not null)
        {
            builder.Append("Error: ").Append(Error).AppendLine();
        }
    }

    private static string BuildMessage(HttpStatusCode statusCode)
    {
        return $"SumUp API request failed with status code {(int)statusCode}.";
    }
}

public sealed class ApiError
{
    public string? Code { get; set; }

    public string? Message { get; set; }

    public JsonElement? Details { get; set; }
}
