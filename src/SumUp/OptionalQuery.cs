namespace SumUp;

/// <summary>
/// Represents a query parameter that can be omitted, set to a value, or set explicitly to null.
/// </summary>
public readonly struct OptionalQuery<T>
{
    internal object? RawValue { get; }

    /// <summary>
    /// Indicates whether the query parameter should be emitted.
    /// </summary>
    public bool IsSet { get; }

    /// <summary>
    /// Indicates whether the query parameter should be emitted with a literal null value.
    /// </summary>
    public bool IsNull => IsSet && RawValue is null;

    private OptionalQuery(bool isSet, object? value)
    {
        IsSet = isSet;
        RawValue = value;
    }

    /// <summary>
    /// Omits the parameter from the query string.
    /// </summary>
    public static OptionalQuery<T> Unset => default;

    /// <summary>
    /// Emits the parameter as an explicit null literal.
    /// </summary>
    public static OptionalQuery<T> Null() => new(isSet: true, value: null);

    /// <summary>
    /// Emits the parameter with the provided value.
    /// </summary>
    public static OptionalQuery<T> From(T value) => new(isSet: true, value: value);

    public static implicit operator OptionalQuery<T>(T value) => From(value);
}
