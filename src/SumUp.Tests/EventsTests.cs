using SumUp.Http;
using System;
using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace SumUp.Tests;

public class EventsTests
{
    private const string Secret = "test_secret";
    private const long Now = 1788600000;
    private static byte[] Body(string type = "members.updated", string url = "https://api.sumup.com/v0.1/members/123") =>
        JsonSerializer.SerializeToUtf8Bytes(new { id = "evt_123", type, created_at = "2026-09-05T09:30:00Z", @object = new { id = "123", type = "member", url } });

    private static string Sign(byte[] body, string? timestamp = null, string secret = Secret)
    {
        timestamp ??= DateTimeOffset.UtcNow.ToUnixTimeSeconds().ToString(CultureInfo.InvariantCulture);
        var prefix = Encoding.UTF8.GetBytes($"v1:{timestamp}:");
        var signed = new byte[prefix.Length + body.Length];
        prefix.CopyTo(signed, 0);
        body.CopyTo(signed, prefix.Length);
        return $"t={timestamp},v1={Convert.ToHexString(HMACSHA256.HashData(Encoding.UTF8.GetBytes(secret), signed))}";
    }

    [Theory]
    [InlineData(-300)]
    [InlineData(0)]
    [InlineData(300)]
    public void Verify_AcceptsBoundaryAndPreservesTimestampText(int offset)
    {
        var body = Encoding.UTF8.GetBytes("héllo");
        var signature = Sign(body, "00" + (Now + offset));
        EventSignature.Verify(body, "  " + signature + "  ", Secret, Now);
        EventSignature.Verify(body, signature.ToLowerInvariant(), Secret, Now);
    }

    [Theory]
    [InlineData(-301)]
    [InlineData(301)]
    public void Verify_RejectsExpiredSignatures(int offset) =>
        Assert.Throws<EventSignatureExpiredException>(() => EventSignature.Verify(Body(), Sign(Body(), (Now + offset).ToString()), Secret, Now));

    [Theory]
    [InlineData(null)]
    [InlineData("")]
    [InlineData("v1=ab")]
    [InlineData("t=1,v1=ab")]
    [InlineData("t=-1,v1=ab")]
    [InlineData("t=+1,v1=ab")]
    [InlineData("t=1, v1=ab")]
    [InlineData("t=1,v1=ab,t=1")]
    [InlineData("v1=ab,t=1")]
    public void Verify_RejectsMalformedHeaders(string? signature) =>
        Assert.Throws<EventSignatureException>(() => EventSignature.Verify(Body(), signature, Secret, Now));

    [Fact]
    public void Verify_RejectsChangedBytesWrongSecretAndOverflowTimestamp()
    {
        var body = Body();
        var signature = Sign(body, Now.ToString());
        body[0] ^= 1;
        Assert.Throws<EventSignatureException>(() => EventSignature.Verify(body, signature, Secret, Now));
        Assert.Throws<EventSignatureException>(() => EventSignature.Verify(Body(), signature, "wrong", Now));
        Assert.Throws<EventSignatureException>(() => EventSignature.Verify(Body(), Sign(Body(), "9223372036854775808"), Secret, Now));
        Assert.Throws<EventSignatureExpiredException>(() => EventSignature.Verify(Body(), Sign(Body(), long.MaxValue.ToString()), Secret, Now));
        Assert.Throws<ArgumentException>(() => EventSignature.Verify(Body(), signature, "", Now));
    }

    [Theory]
    [InlineData("members.created", typeof(MemberCreatedEvent))]
    [InlineData("members.updated", typeof(MemberUpdatedEvent))]
    [InlineData("members.deleted", typeof(MemberDeletedEvent))]
    [InlineData("readers.created", typeof(ReaderCreatedEvent))]
    [InlineData("readers.deleted", typeof(ReaderDeletedEvent))]
    [InlineData("future.event", typeof(EventNotification))]
    public void Parse_ReturnsTypedEvent(string type, Type model)
    {
        using var client = new SumUpClient();
        var body = Body(type);
        var notification = client.ParseEventNotification(body, Sign(body), Secret);
        Assert.IsType(model, notification);
        Assert.Equal("evt_123", notification.Id);
        Assert.Equal(type, notification.Type);
        Assert.Equal(2026, notification.CreatedAt.Year);
        Assert.IsType(model, client.ParseEventNotificationWithoutVerification(body));
    }

    [Theory]
    [InlineData("null")]
    [InlineData("[]")]
    [InlineData("{} {}")]
    [InlineData("{")]
    [InlineData("{\"type\":12}")]
    [InlineData("{\"created_at\":\"invalid\"}")]
    public void Parse_ReportsDeserializationErrors(string json)
    {
        using var client = new SumUpClient();
        Assert.Throws<EventPayloadException>(() => client.ParseEventNotificationWithoutVerification(Encoding.UTF8.GetBytes(json)));
    }

    [Fact]
    public void Parse_VerifiesFirstAndDoesNotAddSchemaValidation()
    {
        using var client = new SumUpClient();
        Assert.Throws<EventSignatureException>(() => client.ParseEventNotification("{"u8, null, Secret));
        Assert.IsType<EventNotification>(client.ParseEventNotificationWithoutVerification("{}"u8));
        Assert.IsType<MemberUpdatedEvent>(client.ParseEventNotificationWithoutVerification("{\"type\":\"members.updated\",\"object\":{\"type\":\"future\"}}"u8));
    }

    [Fact]
    public async Task Handle_AwaitsTypedCallbackAndUsesFallback()
    {
        using var client = new SumUpClient();
        var fallbackCalls = 0;
        var events = client.CreateEventsHandler(Secret, (_, _) => { fallbackCalls++; return Task.CompletedTask; });
        var completion = new TaskCompletionSource(TaskCreationOptions.RunContinuationsAsynchronously);
        using var cancellation = new CancellationTokenSource();
        events.OnMemberUpdated((notification, token) =>
        {
            Assert.Equal("evt_123", notification.Id);
            Assert.Equal(cancellation.Token, token);
            return completion.Task;
        });
        var body = Body();
        var handling = events.HandleAsync(body, Sign(body), cancellation.Token);
        Assert.False(handling.IsCompleted);
        completion.SetResult();
        await handling;
        foreach (var type in new[] { "future.event", "members.created" })
        {
            body = Body(type);
            await events.HandleAsync(body, Sign(body));
        }
        Assert.Equal(2, fallbackCalls);
    }

    [Fact]
    public async Task Handle_WrapsFailurePreservesCancellationAndReplacesCallback()
    {
        using var client = new SumUpClient();
        var cause = new InvalidOperationException("callback failed");
        var events = client.CreateEventsHandler(Secret, (_, _) => throw cause);
        var body = Body();
        var error = await Assert.ThrowsAsync<EventCallbackException>(() => events.HandleAsync(body, Sign(body)));
        Assert.Same(cause, error.InnerException);
        events.OnMemberUpdated((_, _) => throw cause);
        events.OnMemberUpdated((_, _) => Task.CompletedTask);
        await events.HandleAsync(body, Sign(body));
        using var cancellation = new CancellationTokenSource();
        events.OnMemberUpdated((_, token) => { cancellation.Cancel(); token.ThrowIfCancellationRequested(); return Task.CompletedTask; });
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => events.HandleAsync(body, Sign(body), cancellation.Token));
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => events.HandleAsync(body, null, cancellation.Token));
    }

    [Theory]
    [InlineData("https://evil.example/member")]
    [InlineData("http://api.sumup.com/member")]
    [InlineData("https://api.sumup.com:444/member")]
    [InlineData("https://api.sumup.com.evil.example/member")]
    [InlineData("/member")]
    [InlineData("file:///member")]
    public async Task Fetch_RejectsOtherOrigins(string url)
    {
        using var transport = new HttpClient(new StubHandler((_, _) => throw new Exception("Must not send")));
        using var client = new SumUpClient(new() { HttpClient = transport });
        var notification = Assert.IsType<MemberUpdatedEvent>(client.ParseEventNotificationWithoutVerification(Body(url: url)));
        await Assert.ThrowsAsync<EventObjectException>(() => notification.FetchObjectAsync());
    }

    [Fact]
    public async Task Fetch_UsesConfiguredOriginNormalizesUrlAndForwardsCredentials()
    {
        using var transport = new HttpClient(new StubHandler((request, token) =>
        {
            Assert.Equal("https://custom.example:8443/member?expand=role", request.RequestUri!.AbsoluteUri);
            Assert.Equal("override", request.Headers.Authorization!.Parameter);
            Assert.NotEmpty(request.Headers.UserAgent);
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent("{\"id\":\"123\"}") });
        }))
        { BaseAddress = new Uri("https://custom.example:8443/base/") };
        using var client = new SumUpClient(new() { HttpClient = transport, AccessToken = "default" });
        var notification = Assert.IsType<MemberUpdatedEvent>(client.ParseEventNotificationWithoutVerification(Body(url: "https://user:pass@CUSTOM.example:8443/member?expand=role#fragment")));
        var member = await notification.FetchObjectAsync(new() { AccessToken = "override" });
        Assert.Equal("123", member!.Id);
    }

    [Fact]
    public async Task Fetch_PropagatesApiErrorsAndCancellation()
    {
        using var transport = new HttpClient(new StubHandler((_, token) =>
        {
            token.ThrowIfCancellationRequested();
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.NotFound) { Content = new StringContent("{}") });
        }));
        using var client = new SumUpClient(new() { HttpClient = transport, AccessToken = "test" });
        var notification = Assert.IsType<MemberDeletedEvent>(client.ParseEventNotificationWithoutVerification(Body("members.deleted")));
        await Assert.ThrowsAsync<ApiException>(() => notification.FetchObjectAsync());
        using var cancellation = new CancellationTokenSource();
        cancellation.Cancel();
        await Assert.ThrowsAnyAsync<OperationCanceledException>(() => notification.FetchObjectAsync(cancellationToken: cancellation.Token));
    }

    private sealed class StubHandler(Func<HttpRequestMessage, CancellationToken, Task<HttpResponseMessage>> send) : HttpMessageHandler
    {
        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken) => send(request, cancellationToken);
    }
}
