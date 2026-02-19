using System;
using System.Net.Http;
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
}
