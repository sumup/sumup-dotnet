using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;

namespace SumUp;

/// <summary>
/// Configures how <see cref="SumUpClient"/> connects to the API.
/// </summary>
public sealed class SumUpClientOptions
{
    private const string AccessTokenEnvironmentVariable = "SUMUP_ACCESS_TOKEN";

    /// <summary>
    /// Gets or sets the base address for API requests.
    /// </summary>
    public Uri BaseAddress { get; set; } = SumUpEnvironment.Production;

    /// <summary>
    /// Gets or sets the timeout applied to the underlying <see cref="HttpClient"/>.
    /// </summary>
    public TimeSpan Timeout { get; set; } = TimeSpan.FromSeconds(100);

    /// <summary>
    /// Gets or sets an optional access token used for Bearer authentication.
    /// </summary>
    public string? AccessToken { get; set; }

    /// <summary>
    /// Optional asynchronous token provider invoked when <see cref="AccessToken"/> is not set.
    /// </summary>
    public Func<CancellationToken, Task<string?>>? AccessTokenProvider { get; set; }

    /// <summary>
    /// Gets or sets the user agent sent with every request.
    /// </summary>
    public string UserAgent { get; set; } = PackageInfo.UserAgent;

    /// <summary>
    /// Supply a custom <see cref="HttpClient"/> instance when you need to control its lifecycle (for DI scenarios).
    /// </summary>
    public HttpClient? HttpClient { get; set; }

    /// <summary>
    /// Creates a new instance populated with defaults and the environment access token (if available).
    /// </summary>
    public static SumUpClientOptions FromEnvironment()
    {
        var options = new SumUpClientOptions();
        options.AccessToken = options.TryReadTokenFromEnvironment();
        return options;
    }

    internal async Task<string?> GetAccessTokenAsync(CancellationToken cancellationToken)
    {
        if (!string.IsNullOrWhiteSpace(AccessToken))
        {
            return AccessToken;
        }

        if (AccessTokenProvider is not null)
        {
            var token = await AccessTokenProvider(cancellationToken).ConfigureAwait(false);
            if (!string.IsNullOrWhiteSpace(token))
            {
                return token;
            }
        }

        return TryReadTokenFromEnvironment();
    }

    private string? TryReadTokenFromEnvironment()
    {
        return Environment.GetEnvironmentVariable(AccessTokenEnvironmentVariable);
    }
}
