using System;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace SumUp.Tests;

public class MembershipsClientTests
{
    [Theory]
    [InlineData(null, null, "")]
    [InlineData("", "", "?resource.parent.id=&resource.parent.type=")]
    [InlineData("parent /&", "organization", "?resource.parent.id=parent%20%2F%26&resource.parent.type=organization")]
    public async Task List_SerializesParentFilters(string? parentId, string? parentType, string expectedQuery)
    {
        using var handler = new RecordingHttpMessageHandler();
        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };
        using var client = new SumUpClient(new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "test-token"
        });
        var options = new MembershipsListOptions
        {
            ResourceParentId = parentId,
            ResourceParentType = parentType
        };
        var expectedUri = "https://mocked.sumup.test/v0.1/memberships" + expectedQuery;

        await client.Memberships.ListAsync(options);
        Assert.Equal(expectedUri, handler.RequestUri!.AbsoluteUri);

        client.Memberships.List(options);
        Assert.Equal(expectedUri, handler.RequestUri!.AbsoluteUri);

        await client.Memberships.ListAsync();
        Assert.Equal("https://mocked.sumup.test/v0.1/memberships", handler.RequestUri!.AbsoluteUri);
    }

    private sealed class RecordingHttpMessageHandler : HttpMessageHandler
    {
        public Uri? RequestUri { get; private set; }

        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            RequestUri = request.RequestUri;
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent("{\"items\":[]}", Encoding.UTF8, "application/json")
            });
        }
    }
}
