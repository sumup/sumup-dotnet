using System;
using System.Globalization;
using System.Runtime.Serialization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading;
using System.Threading.Tasks;

namespace SumUp;

/// <summary>
/// Shared constants for SumUp webhook verification.
/// </summary>
public static class WebhookConstants
{
    /// <summary>
    /// The header containing the signed payload digest.
    /// </summary>
    public const string SignatureHeader = "X-SumUp-Webhook-Signature";

    /// <summary>
    /// The header containing the Unix timestamp used for signing.
    /// </summary>
    public const string TimestampHeader = "X-SumUp-Webhook-Timestamp";

    /// <summary>
    /// The current webhook signing version.
    /// </summary>
    public const string SignatureVersion = "v1";

    /// <summary>
    /// The environment variable used when no webhook secret is passed explicitly.
    /// </summary>
    public const string SecretEnvironmentVariable = "SUMUP_WEBHOOK_SECRET";

    /// <summary>
    /// The default tolerance applied when validating webhook timestamps.
    /// </summary>
    public static readonly TimeSpan DefaultTolerance = TimeSpan.FromMinutes(5);
}

/// <summary>
/// Base exception for webhook verification and parsing failures.
/// </summary>
public class WebhookException : Exception
{
    public WebhookException(string message)
        : base(message)
    {
    }

    public WebhookException(string message, Exception innerException)
        : base(message, innerException)
    {
    }
}

/// <summary>
/// Raised when webhook verification is attempted without a configured secret.
/// </summary>
public sealed class WebhookSecretMissingException : WebhookException
{
    public WebhookSecretMissingException()
        : base($"Webhook secret is not configured. Pass a secret explicitly or set {WebhookConstants.SecretEnvironmentVariable}.")
    {
    }
}

/// <summary>
/// Raised when the webhook timestamp header is missing or malformed.
/// </summary>
public class WebhookTimestampException : WebhookException
{
    public WebhookTimestampException(string message)
        : base(message)
    {
    }

    public WebhookTimestampException(string message, Exception innerException)
        : base(message, innerException)
    {
    }
}

/// <summary>
/// Raised when the webhook signature header is missing or invalid.
/// </summary>
public class WebhookSignatureException : WebhookException
{
    public WebhookSignatureException(string message)
        : base(message)
    {
    }
}

/// <summary>
/// Raised when the webhook timestamp falls outside the configured tolerance.
/// </summary>
public sealed class WebhookSignatureExpiredException : WebhookTimestampException
{
    public WebhookSignatureExpiredException()
        : base("Webhook timestamp is outside the allowed tolerance.")
    {
    }
}

/// <summary>
/// Raised when the webhook body cannot be parsed into a valid event notification.
/// </summary>
public sealed class WebhookPayloadException : WebhookException
{
    public WebhookPayloadException(string message)
        : base(message)
    {
    }

    public WebhookPayloadException(string message, Exception innerException)
        : base(message, innerException)
    {
    }
}

/// <summary>
/// Known SumUp webhook event types.
/// </summary>
[JsonConverter(typeof(EnumMemberJsonConverterFactory))]
public enum WebhookEventType
{
    [EnumMember(Value = "checkout.created")]
    CheckoutCreated,
    [EnumMember(Value = "checkout.processed")]
    CheckoutProcessed,
    [EnumMember(Value = "checkout.failed")]
    CheckoutFailed,
    [EnumMember(Value = "checkout.terminated")]
    CheckoutTerminated,
    [EnumMember(Value = "member.created")]
    MemberCreated,
    [EnumMember(Value = "member.removed")]
    MemberRemoved,
}

/// <summary>
/// Reference to the SumUp resource associated with a webhook event.
/// </summary>
public sealed class WebhookObjectReference
{
    /// <summary>
    /// Gets the resource identifier.
    /// </summary>
    [JsonPropertyName("id")]
    public string Id { get; init; } = default!;

    /// <summary>
    /// Gets the resource type.
    /// </summary>
    [JsonPropertyName("type")]
    public string Type { get; init; } = default!;

    /// <summary>
    /// Gets the absolute URL for the referenced resource.
    /// </summary>
    [JsonPropertyName("url")]
    public string Url { get; init; } = default!;
}

/// <summary>
/// Generic SumUp webhook event envelope.
/// </summary>
public class WebhookEvent
{
    private SumUpClient? _client;

    /// <summary>
    /// Gets the webhook event identifier.
    /// </summary>
    [JsonPropertyName("id")]
    public string Id { get; init; } = default!;

    /// <summary>
    /// Gets the raw webhook event type.
    /// </summary>
    [JsonPropertyName("type")]
    public string Type { get; init; } = default!;

    /// <summary>
    /// Gets the timestamp at which the webhook event was created.
    /// </summary>
    [JsonPropertyName("created_at")]
    public DateTimeOffset CreatedAt { get; init; }

    /// <summary>
    /// Gets the referenced SumUp object.
    /// </summary>
    [JsonPropertyName("object")]
    public WebhookObjectReference Object { get; init; } = default!;

    internal void BindClient(SumUpClient? client)
    {
        _client = client;
    }

    internal SumUpClient RequireClient()
    {
        return _client ?? throw new InvalidOperationException("This webhook event is not bound to a SumUpClient instance.");
    }
}

/// <summary>
/// Base class for webhook events whose referenced object can be fetched from the API.
/// </summary>
/// <typeparam name="TModel">The API model returned by the referenced object URL.</typeparam>
public abstract class WebhookEvent<TModel> : WebhookEvent where TModel : class
{
    /// <summary>
    /// Fetches the resource referenced by this webhook event.
    /// </summary>
    /// <param name="requestOptions">Optional per-request overrides.</param>
    /// <param name="cancellationToken">Token used to cancel the request.</param>
    public TModel? FetchObject(RequestOptions? requestOptions = null, CancellationToken cancellationToken = default)
    {
        return RequireClient().FetchWebhookObject<TModel>(Object.Url, requestOptions, cancellationToken);
    }

    /// <summary>
    /// Fetches the resource referenced by this webhook event asynchronously.
    /// </summary>
    /// <param name="requestOptions">Optional per-request overrides.</param>
    /// <param name="cancellationToken">Token used to cancel the request.</param>
    public Task<TModel?> FetchObjectAsync(RequestOptions? requestOptions = null, CancellationToken cancellationToken = default)
    {
        return RequireClient().FetchWebhookObjectAsync<TModel>(Object.Url, requestOptions, cancellationToken);
    }
}

/// <summary>
/// Event emitted when a checkout is created.
/// </summary>
public sealed class CheckoutCreatedEvent : WebhookEvent<CheckoutSuccess>
{
}

/// <summary>
/// Event emitted when a checkout is processed.
/// </summary>
public sealed class CheckoutProcessedEvent : WebhookEvent<CheckoutSuccess>
{
}

/// <summary>
/// Event emitted when a checkout processing attempt fails.
/// </summary>
public sealed class CheckoutFailedEvent : WebhookEvent<CheckoutSuccess>
{
}

/// <summary>
/// Event emitted when a checkout is terminated.
/// </summary>
public sealed class CheckoutTerminatedEvent : WebhookEvent<CheckoutSuccess>
{
}

/// <summary>
/// Event emitted when a member is created.
/// </summary>
public sealed class MemberCreatedEvent : WebhookEvent<Member>
{
}

/// <summary>
/// Event emitted when a member is removed.
/// </summary>
public sealed class MemberRemovedEvent : WebhookEvent<Member>
{
}

/// <summary>
/// Verifies and parses incoming SumUp webhook notifications.
/// </summary>
public sealed class WebhookHandler
{
    private static readonly JsonSerializerOptions SerializerOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly byte[]? _secretBytes;
    private readonly SumUpClient? _client;

    /// <summary>
    /// Initializes a new webhook handler.
    /// </summary>
    /// <param name="secret">The webhook signing secret. Falls back to <see cref="WebhookConstants.SecretEnvironmentVariable"/> when omitted.</param>
    /// <param name="tolerance">The allowed clock skew when validating the timestamp header.</param>
    /// <param name="client">Optional client bound to parsed events for fetch operations.</param>
    public WebhookHandler(string? secret = null, TimeSpan? tolerance = null, SumUpClient? client = null)
    {
        Secret = secret ?? Environment.GetEnvironmentVariable(WebhookConstants.SecretEnvironmentVariable);
        Tolerance = tolerance ?? WebhookConstants.DefaultTolerance;
        _secretBytes = Secret is null ? null : Encoding.UTF8.GetBytes(Secret);
        _client = client;
    }

    /// <summary>
    /// Gets the configured webhook signing secret, if any.
    /// </summary>
    public string? Secret { get; }

    /// <summary>
    /// Gets the allowed clock skew applied during timestamp verification.
    /// </summary>
    public TimeSpan Tolerance { get; }

    /// <summary>
    /// Verifies the webhook signature and timestamp headers for a payload.
    /// </summary>
    /// <param name="signatureHeader">The value of <see cref="WebhookConstants.SignatureHeader"/>.</param>
    /// <param name="timestampHeader">The value of <see cref="WebhookConstants.TimestampHeader"/>.</param>
    /// <param name="body">The raw request body.</param>
    /// <param name="now">Optional current time override used for testing.</param>
    public void Verify(string? signatureHeader, string? timestampHeader, string body, DateTimeOffset? now = null)
    {
        Verify(signatureHeader, timestampHeader, Encoding.UTF8.GetBytes(body), now);
    }

    /// <summary>
    /// Verifies the webhook signature and timestamp headers for a payload.
    /// </summary>
    /// <param name="signatureHeader">The value of <see cref="WebhookConstants.SignatureHeader"/>.</param>
    /// <param name="timestampHeader">The value of <see cref="WebhookConstants.TimestampHeader"/>.</param>
    /// <param name="body">The raw request body.</param>
    /// <param name="now">Optional current time override used for testing.</param>
    public void Verify(string? signatureHeader, string? timestampHeader, ReadOnlySpan<byte> body, DateTimeOffset? now = null)
    {
        if (_secretBytes is null)
        {
            throw new WebhookSecretMissingException();
        }

        if (string.IsNullOrWhiteSpace(signatureHeader))
        {
            throw new WebhookSignatureException("Missing webhook signature header.");
        }

        if (string.IsNullOrWhiteSpace(timestampHeader))
        {
            throw new WebhookTimestampException("Missing webhook timestamp header.");
        }

        if (!long.TryParse(timestampHeader, NumberStyles.Integer, CultureInfo.InvariantCulture, out var unixTimestamp))
        {
            throw new WebhookTimestampException("Webhook timestamp header is invalid.");
        }

        var timestamp = DateTimeOffset.FromUnixTimeSeconds(unixTimestamp);
        var effectiveNow = now ?? DateTimeOffset.UtcNow;
        if ((effectiveNow - timestamp).Duration() > Tolerance)
        {
            throw new WebhookSignatureExpiredException();
        }

        var separatorIndex = signatureHeader.IndexOf('=');
        if (separatorIndex <= 0 || separatorIndex == signatureHeader.Length - 1)
        {
            throw new WebhookSignatureException("Webhook signature header has an invalid format.");
        }

        var version = signatureHeader[..separatorIndex];
        var providedDigest = signatureHeader[(separatorIndex + 1)..];
        if (!string.Equals(version, WebhookConstants.SignatureVersion, StringComparison.Ordinal))
        {
            throw new WebhookSignatureException("Unsupported webhook signature version.");
        }

        var expectedDigest = SignPayload(_secretBytes, unixTimestamp, body);
        var providedDigestBytes = Encoding.ASCII.GetBytes(providedDigest);
        var expectedDigestBytes = Encoding.ASCII.GetBytes(expectedDigest);
        if (!CryptographicOperations.FixedTimeEquals(providedDigestBytes, expectedDigestBytes))
        {
            throw new WebhookSignatureException("Webhook signature is invalid.");
        }
    }

    /// <summary>
    /// Parses a webhook payload into the most specific known event model.
    /// </summary>
    /// <param name="body">The raw webhook payload.</param>
    public WebhookEvent Parse(string body)
    {
        try
        {
            using var document = JsonDocument.Parse(body);
            var eventType = document.RootElement.TryGetProperty("type", out var typeProperty) && typeProperty.ValueKind == JsonValueKind.String
                ? typeProperty.GetString()
                : null;

            var notification = eventType switch
            {
                "checkout.created" => DeserializeAndValidate<CheckoutCreatedEvent>(body, "checkout"),
                "checkout.processed" => DeserializeAndValidate<CheckoutProcessedEvent>(body, "checkout"),
                "checkout.failed" => DeserializeAndValidate<CheckoutFailedEvent>(body, "checkout"),
                "checkout.terminated" => DeserializeAndValidate<CheckoutTerminatedEvent>(body, "checkout"),
                "member.created" => DeserializeAndValidate<MemberCreatedEvent>(body, "member"),
                "member.removed" => DeserializeAndValidate<MemberRemovedEvent>(body, "member"),
                _ => JsonSerializer.Deserialize<WebhookEvent>(body, SerializerOptions)
            };

            if (notification is null)
            {
                throw new WebhookPayloadException("Webhook payload could not be deserialized.");
            }

            if (notification.Object is null)
            {
                throw new WebhookPayloadException("Webhook payload is missing the referenced object.");
            }

            notification.BindClient(_client);
            return notification;
        }
        catch (JsonException exception)
        {
            throw new WebhookPayloadException("Webhook payload is not valid JSON.", exception);
        }
    }

    /// <summary>
    /// Verifies the webhook signature and then parses the payload.
    /// </summary>
    /// <param name="signatureHeader">The value of <see cref="WebhookConstants.SignatureHeader"/>.</param>
    /// <param name="timestampHeader">The value of <see cref="WebhookConstants.TimestampHeader"/>.</param>
    /// <param name="body">The raw request body.</param>
    /// <param name="now">Optional current time override used for testing.</param>
    public WebhookEvent VerifyAndParse(string? signatureHeader, string? timestampHeader, string body, DateTimeOffset? now = null)
    {
        Verify(signatureHeader, timestampHeader, body, now);
        return Parse(body);
    }

    private static string SignPayload(byte[] secret, long unixTimestamp, ReadOnlySpan<byte> body)
    {
        var prefix = Encoding.UTF8.GetBytes($"{WebhookConstants.SignatureVersion}:{unixTimestamp}:");
        var payload = new byte[prefix.Length + body.Length];
        prefix.CopyTo(payload, 0);
        body.CopyTo(payload.AsSpan(prefix.Length));

        using var hmac = new HMACSHA256(secret);
        var digest = hmac.ComputeHash(payload);
        return Convert.ToHexString(digest).ToLowerInvariant();
    }

    private static TEvent DeserializeAndValidate<TEvent>(string body, string expectedObjectType)
        where TEvent : WebhookEvent
    {
        var notification = JsonSerializer.Deserialize<TEvent>(body, SerializerOptions)
            ?? throw new WebhookPayloadException("Webhook payload could not be deserialized.");

        if (notification.Object is null)
        {
            throw new WebhookPayloadException("Webhook payload is missing the referenced object.");
        }

        if (!string.Equals(notification.Object.Type, expectedObjectType, StringComparison.OrdinalIgnoreCase))
        {
            throw new WebhookPayloadException(
                $"Webhook payload object type '{notification.Object.Type}' does not match event '{notification.Type}'.");
        }

        return notification;
    }
}
