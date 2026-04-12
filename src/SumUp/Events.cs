using SumUp.Http;
using System.Globalization;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace SumUp;

/// <summary>An event delivered by SumUp, including event types introduced after this SDK release.</summary>
public class EventNotification
{
    /// <summary>The event ID. Use it to deduplicate deliveries.</summary>
    [JsonPropertyName("id")]
    public string Id { get; init; } = "";

    /// <summary>The event name, such as <c>members.updated</c>.</summary>
    [JsonPropertyName("type")]
    public string Type { get; init; } = "";

    /// <summary>When the event occurred. Signature verification uses the delivery timestamp in the signature header.</summary>
    [JsonPropertyName("created_at")]
    public DateTimeOffset CreatedAt { get; init; }

    /// <summary>A reference to the affected resource.</summary>
    [JsonPropertyName("object")]
    public EventObjectReference Object { get; init; } = new();

    internal SumUpClient? Client { get; set; }
}

/// <summary>Identifies the resource affected by an event.</summary>
public sealed class EventObjectReference
{
    /// <summary>The resource ID, available without an API request.</summary>
    [JsonPropertyName("id")]
    public string Id { get; init; } = "";
    /// <summary>The resource type, such as <c>member</c> or <c>reader</c>.</summary>
    [JsonPropertyName("type")]
    public string Type { get; init; } = "";
    /// <summary>The API URL used to fetch the resource.</summary>
    [JsonPropertyName("url")]
    public string Url { get; init; } = "";
}

/// <summary>An event whose affected resource can be fetched through the originating client.</summary>
/// <typeparam name="T">The resource model.</typeparam>
public class EventNotification<T> : EventNotification where T : class
{
    /// <summary>Fetches the resource's current state, rather than its state when the event occurred.</summary>
    /// <param name="requestOptions">Optional authentication and timeout overrides.</param>
    /// <param name="cancellationToken">Cancels the API request.</param>
    /// <returns>The current resource, or null for an empty JSON result.</returns>
    /// <remarks>A deleted resource may no longer be available and can return an API error.</remarks>
    /// <exception cref="EventObjectException">The event is not bound to a client or the resource URL has a different origin.</exception>
    /// <exception cref="ApiException">The API returned an unsuccessful response.</exception>
    public Task<T?> FetchObjectAsync(RequestOptions? requestOptions = null, CancellationToken cancellationToken = default)
    {
        var client = Client ?? throw new EventObjectException("Parse the event through a SumUp client before fetching its resource.");
        return client.FetchEventObjectAsync<T>(Object?.Url ?? "", requestOptions, cancellationToken);
    }

    /// <summary>Synchronously fetches the resource's current state. Prefer <see cref="FetchObjectAsync"/> in asynchronous code.</summary>
    /// <param name="requestOptions">Optional authentication and timeout overrides.</param>
    /// <param name="cancellationToken">Cancels the API request.</param>
    /// <returns>The current resource, or null for an empty JSON result.</returns>
    public T? FetchObject(RequestOptions? requestOptions = null, CancellationToken cancellationToken = default) =>
        FetchObjectAsync(requestOptions, cancellationToken).GetAwaiter().GetResult();
}

/// <summary>Verifies event signatures using the original request bytes.</summary>
public static class EventSignature
{
    /// <summary>The HTTP header containing the delivery timestamp and signature.</summary>
    public const string HeaderName = "X-SumUp-Webhook-Signature";

    /// <summary>Checks the signature and enforces a five-minute delivery timestamp tolerance in either direction.</summary>
    /// <param name="body">The unmodified HTTP request body. Do not deserialize and serialize it before verification.</param>
    /// <param name="signature">The complete value of <see cref="HeaderName"/>.</param>
    /// <param name="secret">The endpoint's signing secret, not an API access token.</param>
    /// <exception cref="ArgumentException">The signing secret is empty.</exception>
    /// <exception cref="EventSignatureException">The signature header is missing, malformed, or does not match.</exception>
    /// <exception cref="EventSignatureExpiredException">The delivery timestamp is outside the accepted window.</exception>
    public static void Verify(ReadOnlySpan<byte> body, string? signature, string secret) =>
        Verify(body, signature, secret, DateTimeOffset.UtcNow.ToUnixTimeSeconds());

    internal static void Verify(ReadOnlySpan<byte> body, string? signature, string secret, long now)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(secret);
        var parts = signature?.Trim().Split(',');
        if (parts is not { Length: 2 } || !parts[0].StartsWith("t=", StringComparison.Ordinal) ||
            !parts[1].StartsWith("v1=", StringComparison.Ordinal))
            throw new EventSignatureException("Expected a signature header in the form t=<timestamp>,v1=<signature>.");

        var timestampText = parts[0][2..];
        if (!long.TryParse(timestampText, NumberStyles.None, CultureInfo.InvariantCulture, out var timestamp))
            throw new EventSignatureException("Invalid signature timestamp.");

        byte[] signatureBytes;
        try
        {
            signatureBytes = Convert.FromHexString(parts[1][3..]);
        }
        catch (FormatException cause)
        {
            throw new EventSignatureException("Invalid signature encoding.", cause);
        }
        if (signatureBytes.Length != 32)
            throw new EventSignatureException("Invalid signature length.");

        using var hmac = IncrementalHash.CreateHMAC(HashAlgorithmName.SHA256, Encoding.UTF8.GetBytes(secret));
        hmac.AppendData(Encoding.UTF8.GetBytes($"v1:{timestampText}:"));
        hmac.AppendData(body);
        if (!CryptographicOperations.FixedTimeEquals(hmac.GetHashAndReset(), signatureBytes))
            throw new EventSignatureException("The event signature does not match.");
        if (timestamp < now - 300 || timestamp > now + 300)
            throw new EventSignatureExpiredException("The event signature timestamp is outside the five-minute tolerance.");
    }
}

/// <summary>Verifies and dispatches events to typed callbacks.</summary>
/// <remarks>Register callbacks before serving requests. Each registration replaces the previous callback for that type.
/// Deliveries can repeat; make callbacks idempotent using the event ID.</remarks>
public sealed partial class EventsHandler
{
    private readonly SumUpClient _client;
    private readonly string _secret;
    private readonly Func<EventNotification, CancellationToken, Task> _fallback;
    private readonly Dictionary<string, Func<EventNotification, CancellationToken, Task>> _callbacks = new();

    internal EventsHandler(SumUpClient client, string secret, Func<EventNotification, CancellationToken, Task> fallback)
    {
        ArgumentException.ThrowIfNullOrWhiteSpace(secret);
        ArgumentNullException.ThrowIfNull(fallback);
        _client = client;
        _secret = secret;
        _fallback = fallback;
    }

    private EventsHandler Register<T>(string type, Func<T, CancellationToken, Task> callback) where T : EventNotification
    {
        ArgumentNullException.ThrowIfNull(callback);
        _callbacks[type] = (notification, token) => callback((T)notification, token);
        return this;
    }

    /// <summary>Verifies and parses an event without invoking callbacks.</summary>
    /// <param name="body">The unmodified HTTP request body.</param>
    /// <param name="signature">The complete signature header value.</param>
    /// <returns>A typed event, or a base notification for unknown event types.</returns>
    /// <exception cref="EventSignatureException">Signature verification failed.</exception>
    /// <exception cref="EventPayloadException">The body cannot be deserialized as an event.</exception>
    public EventNotification Parse(ReadOnlySpan<byte> body, string? signature) =>
        _client.ParseEventNotification(body, signature, _secret);

    /// <summary>Verifies, parses, and awaits the registered callback, or the fallback for unregistered event types.</summary>
    /// <param name="body">The unmodified HTTP request body.</param>
    /// <param name="signature">The complete signature header value.</param>
    /// <param name="cancellationToken">Cancels handling and is forwarded to the callback.</param>
    /// <returns>A task that completes when the callback finishes. Acknowledge delivery only after it succeeds.</returns>
    /// <exception cref="EventSignatureException">Signature verification failed.</exception>
    /// <exception cref="EventPayloadException">The body cannot be deserialized as an event.</exception>
    /// <exception cref="EventCallbackException">The callback failed; its exception is available as the inner exception.</exception>
    /// <exception cref="OperationCanceledException">Handling was canceled.</exception>
    public async Task HandleAsync(ReadOnlyMemory<byte> body, string? signature, CancellationToken cancellationToken = default)
    {
        cancellationToken.ThrowIfCancellationRequested();
        var notification = Parse(body.Span, signature);
        var callback = _callbacks.GetValueOrDefault(notification.Type ?? "") ?? _fallback;
        try
        {
            await callback(notification, cancellationToken).ConfigureAwait(false);
        }
        catch (OperationCanceledException) when (cancellationToken.IsCancellationRequested)
        {
            throw;
        }
        catch (Exception cause)
        {
            throw new EventCallbackException("Event callback failed.", cause);
        }
    }
}

public partial class SumUpClient
{
    /// <summary>Creates an event handler bound to this client's API credentials and configuration.</summary>
    /// <param name="secret">The endpoint's signing secret, not an API access token.</param>
    /// <param name="fallback">Called for unknown event types and types without a registered callback.</param>
    /// <returns>A handler on which to register typed callbacks before serving requests.</returns>
    public EventsHandler CreateEventsHandler(string secret, Func<EventNotification, CancellationToken, Task> fallback) => new(this, secret, fallback);

    /// <summary>Verifies the signature before deserializing an event.</summary>
    /// <param name="body">The unmodified HTTP request body.</param>
    /// <param name="signature">The complete value of <see cref="EventSignature.HeaderName"/>.</param>
    /// <param name="secret">The endpoint's signing secret, not an API access token.</param>
    /// <returns>A typed event bound to this client, or a base notification for an unknown event type.</returns>
    /// <exception cref="EventSignatureException">Signature verification failed.</exception>
    /// <exception cref="EventPayloadException">The body cannot be deserialized as an event.</exception>
    public EventNotification ParseEventNotification(ReadOnlySpan<byte> body, string? signature, string secret)
    {
        EventSignature.Verify(body, signature, secret);
        return ParseEvent(body);
    }

    /// <summary>Parses an event without checking its signature. Use only for already verified payloads from trusted storage.</summary>
    /// <param name="body">The JSON event body.</param>
    /// <returns>A typed event bound to this client, or a base notification for an unknown event type.</returns>
    /// <exception cref="EventPayloadException">The body cannot be deserialized as an event.</exception>
    public EventNotification ParseEventNotificationWithoutVerification(ReadOnlySpan<byte> body) => ParseEvent(body);

    private EventNotification ParseEvent(ReadOnlySpan<byte> body)
    {
        try
        {
            var reader = new Utf8JsonReader(body);
            using var document = JsonDocument.ParseValue(ref reader);
            if (reader.Read() || document.RootElement.ValueKind != JsonValueKind.Object)
                throw new JsonException("Expected a JSON object.");
            var root = document.RootElement;
            var type = root.TryGetProperty("type", out var value) ? value.GetString() : null;
            var notification = DeserializeEvent(root, type);
            notification.Client = this;
            return notification;
        }
        catch (Exception cause) when (cause is JsonException or InvalidOperationException)
        {
            throw new EventPayloadException("Cannot deserialize the event body.", cause);
        }
    }

    internal Task<T?> FetchEventObjectAsync<T>(string url, RequestOptions? requestOptions, CancellationToken cancellationToken) where T : class
    {
        var origin = _httpClient.BaseAddress!;
        if (!Uri.TryCreate(url, UriKind.Absolute, out var resource) ||
            (resource.Scheme != Uri.UriSchemeHttps && resource.Scheme != Uri.UriSchemeHttp) ||
            !string.Equals(resource.Scheme, origin.Scheme, StringComparison.OrdinalIgnoreCase) ||
            !string.Equals(resource.IdnHost, origin.IdnHost, StringComparison.OrdinalIgnoreCase) || resource.Port != origin.Port)
            throw new EventObjectException("The event resource URL must have the same origin as the API client.");
        var normalized = new UriBuilder(resource) { UserName = "", Password = "", Fragment = "" }.Uri;
        return _apiClient.GetAbsoluteAsync<T>(normalized.AbsoluteUri, requestOptions, cancellationToken);
    }
}
