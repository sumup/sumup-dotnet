using System;
using System.Net.Http.Headers;
using System.Runtime.InteropServices;

namespace SumUp.Http;

internal static class RuntimeHeaders
{
    private const string Language = "dotnet";
    private const string Runtime = "dotnet";

    private static readonly string PackageVersion = PackageInfo.Version;

    private static readonly string OsName = GetOsName();
    private static readonly string RuntimeArch = GetArchitecture();
    private static readonly string RuntimeVersion = RuntimeInformation.FrameworkDescription;

    internal static void Apply(HttpRequestHeaders headers)
    {
        SetHeader(headers, "X-Sumup-Api-Version", ApiVersionInfo.Value);
        SetHeader(headers, "X-Sumup-Lang", Language);
        SetHeader(headers, "X-Sumup-Package-Version", PackageVersion);
        SetHeader(headers, "X-Sumup-Os", OsName);
        SetHeader(headers, "X-Sumup-Arch", RuntimeArch);
        SetHeader(headers, "X-Sumup-Runtime", Runtime);
        SetHeader(headers, "X-Sumup-Runtime-Version", RuntimeVersion);
    }

    private static void SetHeader(HttpRequestHeaders headers, string name, string value)
    {
        headers.Remove(name);
        headers.TryAddWithoutValidation(name, value);
    }

    private static string GetOsName()
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

    private static string GetArchitecture()
    {
        return RuntimeInformation.OSArchitecture switch
        {
            System.Runtime.InteropServices.Architecture.X64 => "x86_64",
            System.Runtime.InteropServices.Architecture.X86 => "x86",
            System.Runtime.InteropServices.Architecture.Arm64 => "arm64",
            System.Runtime.InteropServices.Architecture.Arm => "arm",
            _ => RuntimeInformation.OSArchitecture.ToString().ToLowerInvariant()
        };
    }
}
