using System.Net;
using System.Net.Http;
using SumUp.Http;
using Xunit;

namespace SumUp.Tests;

public class ApiExceptionTests
{
    [Fact]
    public void BaseCatchExposesTheTypedProblemAndHeaders()
    {
        var problem = new Problem { Type = "about:blank", Detail = "Invalid amount" };
        using var response = new HttpResponseMessage(HttpStatusCode.BadRequest);
        response.Headers.Add("Retry-After", "10");
        ApiException error = new ApiException<Problem>(response.StatusCode, problem, "raw", null, response.Headers);
        Assert.Same(problem, error.Problem);
        Assert.Equal("Invalid amount", error.Problem!.Detail);
        Assert.Same(response.Headers, error.Headers);
        Assert.Equal("raw", error.ResponseBody);
    }

    [Theory]
    [InlineData("{invalid")]
    [InlineData("Bad gateway")]
    [InlineData("{\"message\":\"Unknown\"}")]
    public void NonProblemBodyRemainsAvailable(string body)
    {
        var error = new ApiException(HttpStatusCode.BadGateway, null, body, null);
        Assert.Null(error.Problem);
        Assert.Equal(body, error.ResponseBody);
    }
}
