using System;

namespace SumUp;

/// <summary>
/// Per-request overrides for <see cref="SumUpClient"/> operations.
/// </summary>
public sealed class RequestOptions
{
    /// <summary>
    /// Optional access token overriding <see cref="SumUpClientOptions.AccessToken"/>.
    /// When set, the value is used to populate the Authorization header for the request.
    /// </summary>
    public string? AccessToken { get; set; }

    /// <summary>
    /// Overrides the request timeout for a single call. When specified, a cancellation token linked to the provided timeout is used to cancel the outbound HTTP call.
    /// </summary>
    public TimeSpan? Timeout { get; set; }
}
