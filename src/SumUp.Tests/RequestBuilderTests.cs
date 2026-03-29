using System;
using System.IO;
using System.Net.Http;
using System.Text;
using SumUp.Http;
using Xunit;

namespace SumUp.Tests;

public class RequestBuilderTests
{
    [Fact]
    public void Build_ReplacesPathAndQuery()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/checkouts/{id}", new Uri("https://api.sumup.com"));
        builder.AddPath("id", "space id");
        builder.AddQuery("status", "open");
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/checkouts/space%20id?status=open", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void Build_IncludesRepeatedQueryParameters()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/items", new Uri("https://api.sumup.com"));
        builder.AddQuery("status", new[] { "open", "closed" });
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/items?status=open&status=closed", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void Build_SerializesEnumQueryUsingEnumMemberValue()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/items", new Uri("https://api.sumup.com"));
        builder.AddQuery("status", MembershipStatus.Accepted);
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/items?status=accepted", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void Build_SerializesRepeatedEnumQueryUsingEnumMemberValues()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/items", new Uri("https://api.sumup.com"));
        builder.AddQuery("status", new[] { MembershipStatus.Accepted, MembershipStatus.Pending });
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/items?status=accepted&status=pending", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void Build_OmitsUnsetOptionalQuery()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/items", new Uri("https://api.sumup.com"));
        builder.AddQuery("status", OptionalQuery<string>.Unset);
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/items", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void Build_EmitsExplicitNullOptionalQuery()
    {
        var builder = new RequestBuilder(HttpMethod.Get, "/v0.1/items", new Uri("https://api.sumup.com"));
        builder.AddQuery("status", OptionalQuery<string>.Null());
        var request = builder.Build();

        Assert.Equal("https://api.sumup.com/v0.1/items?status=null", request.RequestUri!.AbsoluteUri);
    }

    [Fact]
    public void CreateContent_SerializesEnumMemberValues()
    {
        using var httpClient = new HttpClient { BaseAddress = new Uri("https://api.sumup.com") };
        var apiClient = new ApiClient(httpClient, new SumUpClientOptions());
        var request = new CheckoutCreateRequest
        {
            CheckoutReference = "test-123",
            Amount = 10.0f,
            Currency = Currency.Eur,
            MerchantCode = "merchant-code",
            Description = "Test order",
        };

        using var content = apiClient.CreateContent(request, "application/json");
        using var stream = content.ReadAsStream();
        using var reader = new StreamReader(stream, Encoding.UTF8);
        var body = reader.ReadToEnd();

        Assert.Contains("\"currency\":\"EUR\"", body);
    }
}
