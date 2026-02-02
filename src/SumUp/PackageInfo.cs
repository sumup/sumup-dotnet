using System;
using System.Reflection;

namespace SumUp;

internal static class PackageInfo
{
    private const string ProductName = "sumup-dotnet";

    internal static string Version => GetVersion();

    internal static string UserAgent => $"{ProductName}/v{Version}";

    private static string GetVersion()
    {
        var assemblyVersion = Assembly.GetExecutingAssembly().GetName().Version;
        if (assemblyVersion is not null)
        {
            return assemblyVersion.ToString();
        }

        return "0.0.0.0";
    }
}
