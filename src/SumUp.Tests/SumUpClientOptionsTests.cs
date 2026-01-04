using System;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace SumUp.Tests;

public class SumUpClientOptionsTests
{
    [Fact]
    public async Task AccessToken_ComesFromEnvironment()
    {
        const string variable = "SUMUP_DOTNET_TEST_TOKEN";
        Environment.SetEnvironmentVariable(variable, "env-token");

        try
        {
            var options = SumUpClientOptions.FromEnvironment();
            options.AccessTokenEnvironmentVariable = variable;
            options.AccessToken = null;

            var token = await options.GetAccessTokenAsync(CancellationToken.None);

            Assert.Equal("env-token", token);
        }
        finally
        {
            Environment.SetEnvironmentVariable(variable, null);
        }
    }

    [Fact]
    public async Task AccessTokenProvider_WinsOverEnvironment()
    {
        const string variable = "SUMUP_DOTNET_TEST_TOKEN";
        Environment.SetEnvironmentVariable(variable, "env-token");

        try
        {
            var options = SumUpClientOptions.FromEnvironment();
            options.AccessTokenEnvironmentVariable = variable;
            options.AccessTokenProvider = _ => Task.FromResult<string?>("provider-token");
            options.AccessToken = null;

            var token = await options.GetAccessTokenAsync(CancellationToken.None);

            Assert.Equal("provider-token", token);
        }
        finally
        {
            Environment.SetEnvironmentVariable(variable, null);
        }
    }
}
