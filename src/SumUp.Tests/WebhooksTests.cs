using System;
using System.Net;
using System.Net.Http;
using System.Security.Cryptography;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace SumUp.Tests;

public class WebhooksTests
{
    private static readonly DateTimeOffset FixedNow = new(2026, 4, 12, 10, 0, 0, TimeSpan.Zero);

    [Fact]
    public void Verify_AcceptsValidSignature()
    {
        const string body = """{"id":"evt_123","type":"checkout.created"}""";
        var signature = SignPayload("whsec_test", FixedNow.ToUnixTimeSeconds(), body);

        var handler = new WebhookHandler("whsec_test");

        handler.Verify(signature, FixedNow.ToUnixTimeSeconds().ToString(), body, FixedNow);
    }

    [Fact]
    public void Verify_RejectsExpiredTimestamp()
    {
        const string body = """{"id":"evt_123","type":"checkout.created"}""";
        var timestamp = FixedNow - WebhookConstants.DefaultTolerance - TimeSpan.FromSeconds(1);
        var signature = SignPayload("whsec_test", timestamp.ToUnixTimeSeconds(), body);

        var handler = new WebhookHandler("whsec_test");

        Assert.Throws<WebhookSignatureExpiredException>(() =>
            handler.Verify(signature, timestamp.ToUnixTimeSeconds().ToString(), body, FixedNow));
    }

    [Fact]
    public void Verify_RejectsInvalidSignature()
    {
        var handler = new WebhookHandler("whsec_test");

        Assert.Throws<WebhookSignatureException>(() =>
            handler.Verify("v1=deadbeef", FixedNow.ToUnixTimeSeconds().ToString(), "{}", FixedNow));
    }

    [Fact]
    public void Verify_RejectsInvalidTimestamp()
    {
        var handler = new WebhookHandler("whsec_test");

        Assert.Throws<WebhookTimestampException>(() =>
            handler.Verify("v1=deadbeef", "nope", "{}", FixedNow));
    }

    [Fact]
    public void Verify_RequiresConfiguredSecret()
    {
        var handler = new WebhookHandler();

        Assert.Throws<WebhookSecretMissingException>(() =>
            handler.Verify("v1=deadbeef", FixedNow.ToUnixTimeSeconds().ToString(), "{}", FixedNow));
    }

    [Fact]
    public void Parse_ReturnsTypedKnownEvent()
    {
        var eventNotification = new WebhookHandler("whsec_test").Parse(CheckoutCreatedPayload);

        var checkoutCreated = Assert.IsType<CheckoutCreatedEvent>(eventNotification);
        Assert.Equal("checkout.created", checkoutCreated.Type);
        Assert.Equal("checkout", checkoutCreated.Object.Type);
    }

    [Fact]
    public void Parse_ReturnsGenericEventForUnknownType()
    {
        const string body = """
        {
          "id": "evt_123",
          "type": "something.else",
          "created_at": "2026-04-11T10:00:00Z",
          "object": {
            "id": "obj_123",
            "type": "other",
            "url": "https://api.sumup.com/v0.1/other/obj_123"
          }
        }
        """;

        var eventNotification = new WebhookHandler("whsec_test").Parse(body);

        Assert.IsType<WebhookEvent>(eventNotification);
        Assert.Equal("something.else", eventNotification.Type);
    }

    [Fact]
    public void Parse_RejectsMalformedJson()
    {
        var handler = new WebhookHandler("whsec_test");

        Assert.Throws<WebhookPayloadException>(() => handler.Parse("{"));
    }

    [Fact]
    public void Parse_RejectsObjectTypeMismatch()
    {
        const string body = """
        {
          "id": "evt_123",
          "type": "checkout.created",
          "created_at": "2026-04-11T10:00:00Z",
          "object": {
            "id": "mem_123",
            "type": "member",
            "url": "https://api.sumup.com/v0.1/members/mem_123"
          }
        }
        """;

        var handler = new WebhookHandler("whsec_test");

        Assert.Throws<WebhookPayloadException>(() => handler.Parse(body));
    }

    [Fact]
    public void ClientCanCreateBoundWebhookHandler()
    {
        using var client = new SumUpClient(new SumUpClientOptions
        {
            AccessToken = "test-token"
        });

        var handler = client.CreateWebhookHandler("whsec_test");

        Assert.Equal("whsec_test", handler.Secret);
        Assert.Equal(WebhookConstants.DefaultTolerance, handler.Tolerance);
    }

    [Fact]
    public async Task VerifyAndParse_BindsClientAndFetchesObject()
    {
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var transport = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(
                    """
                    {
                      "id": "chk_123",
                      "amount": 10.0,
                      "checkout_reference": "ref_123",
                      "currency": "EUR",
                      "date": "2026-04-11T10:00:00Z",
                      "description": "Test payment",
                      "merchant_code": "MC123",
                      "status": "PENDING"
                    }
                    """,
                    Encoding.UTF8,
                    "application/json")
            };
        });

        using var httpClient = new HttpClient(transport)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        using var client = new SumUpClient(new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "test-token"
        });

        var signature = SignPayload("whsec_test", FixedNow.ToUnixTimeSeconds(), CheckoutCreatedPayload);
        var eventNotification = client.CreateWebhookHandler("whsec_test")
            .VerifyAndParse(signature, FixedNow.ToUnixTimeSeconds().ToString(), CheckoutCreatedPayload, FixedNow);

        var checkoutCreated = Assert.IsType<CheckoutCreatedEvent>(eventNotification);
        var checkout = await checkoutCreated.FetchObjectAsync();

        Assert.NotNull(checkout);
        Assert.Equal("chk_123", checkout!.Id);
        Assert.Equal(Currency.Eur, checkout.Currency);
        Assert.Equal("https://api.sumup.com/v0.1/checkouts/chk_123", transport.LastRequest?.RequestUri?.AbsoluteUri);
        Assert.Equal("Bearer test-token", transport.LastRequest?.Headers.Authorization?.ToString());
    }

    private const string CheckoutCreatedPayload = """
    {
      "id": "evt_123",
      "type": "checkout.created",
      "created_at": "2026-04-11T10:00:00Z",
      "object": {
        "id": "chk_123",
        "type": "checkout",
        "url": "https://api.sumup.com/v0.1/checkouts/chk_123"
      }
    }
    """;

    private static string SignPayload(string secret, long timestamp, string body)
    {
        var payload = Encoding.UTF8.GetBytes($"{WebhookConstants.SignatureVersion}:{timestamp}:{body}");
        using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(secret));
        var digest = hmac.ComputeHash(payload);
        return $"{WebhookConstants.SignatureVersion}={Convert.ToHexString(digest).ToLowerInvariant()}";
    }

    private sealed class EnvironmentVariableScope : IDisposable
    {
        private readonly string _name;
        private readonly string? _previousValue;

        internal EnvironmentVariableScope(string name, string? value)
        {
            _name = name;
            _previousValue = Environment.GetEnvironmentVariable(name);
            Environment.SetEnvironmentVariable(name, value);
        }

        public void Dispose()
        {
            Environment.SetEnvironmentVariable(_name, _previousValue);
        }
    }

    private sealed class RecordingHttpMessageHandler : HttpMessageHandler
    {
        private readonly Func<HttpRequestMessage, HttpResponseMessage> _handler;

        internal RecordingHttpMessageHandler(Func<HttpRequestMessage, HttpResponseMessage> handler)
        {
            _handler = handler;
        }

        internal HttpRequestMessage? LastRequest { get; private set; }

        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            LastRequest = request;
            return Task.FromResult(_handler(request));
        }
    }
}
