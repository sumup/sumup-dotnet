using System;

namespace SumUp;

/// <summary>
/// Known SumUp API environments.
/// </summary>
public static class SumUpEnvironment
{
    /// <summary>
    /// Production API base address.
    /// </summary>
    public static Uri Production { get; } = new("https://api.sumup.com");

    /// <summary>
    /// Sandbox API base address.
    /// </summary>
    public static Uri Sandbox { get; } = new("https://sandbox.sumup.com");
}
