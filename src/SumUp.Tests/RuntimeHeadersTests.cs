using System;
using System.Net.Http;
using System.Runtime.InteropServices;
using SumUp.Http;
using Xunit;

namespace SumUp.Tests;

public class RuntimeHeadersTests
{
    [Fact]
    public void CreateRequest_AddsRuntimeHeaders()
    {
        using var httpClient = new HttpClient { BaseAddress = new Uri("https://api.sumup.com") };
        var options = new SumUpClientOptions();
        var apiClient = new ApiClient(httpClient, options);

        var request = apiClient.CreateRequest(HttpMethod.Get, "/ping");

        AssertHeader(request, "X-Sumup-Api-Version", ApiVersionInfo.Value);
        AssertHeader(request, "X-Sumup-Lang", "dotnet");
        AssertHeader(request, "X-Sumup-Runtime", "dotnet");
        AssertHeader(request, "X-Sumup-Os", GetExpectedOs());
        AssertHeader(request, "X-Sumup-Arch", GetExpectedArch());

        Assert.True(request.Headers.Contains("X-Sumup-Package-Version"));
        Assert.True(request.Headers.Contains("X-Sumup-Runtime-Version"));
    }

    private static void AssertHeader(HttpRequestMessage request, string name, string expected)
    {
        Assert.True(request.Headers.TryGetValues(name, out var values));
        Assert.Contains(expected, values);
    }

    private static string GetExpectedOs()
    {
        if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
        {
            return "windows";
        }

        if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux))
        {
            return "linux";
        }

        if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
        {
            return "darwin";
        }

        return RuntimeInformation.OSDescription;
    }

    private static string GetExpectedArch()
    {
        return RuntimeInformation.OSArchitecture switch
        {
            Architecture.X64 => "x86_64",
            Architecture.X86 => "x86",
            Architecture.Arm64 => "arm64",
            Architecture.Arm => "arm",
            _ => RuntimeInformation.OSArchitecture.ToString().ToLowerInvariant()
        };
    }
}
