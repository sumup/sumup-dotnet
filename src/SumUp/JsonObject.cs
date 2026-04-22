using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace SumUp;

/// <summary>
/// Mutable JSON object helper used for free-form request and response payloads.
/// </summary>
[JsonConverter(typeof(JsonObjectConverter))]
public class JsonObject : Dictionary<string, object?>
{
    /// <summary>
    /// Parse a JSON object string into a <see cref="JsonObject"/>.
    /// </summary>
    public static JsonObject Parse(string json, JsonSerializerOptions? options = null)
    {
        if (json is null)
        {
            throw new ArgumentNullException(nameof(json));
        }

        return JsonSerializer.Deserialize<JsonObject>(json, options) ?? new JsonObject();
    }

    /// <summary>
    /// Read a stored value and convert it to the requested type.
    /// </summary>
    public T? GetValue<T>(string propertyName, JsonSerializerOptions? options = null)
    {
        if (!TryGetValue(propertyName, out var value))
        {
            return default;
        }

        if (value is T typedValue)
        {
            return typedValue;
        }

        var payload = JsonSerializer.SerializeToUtf8Bytes(value, options);
        return JsonSerializer.Deserialize<T>(payload, options);
    }
}

internal sealed class JsonObjectConverter : JsonConverter<JsonObject>
{
    public override JsonObject Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        if (reader.TokenType != JsonTokenType.StartObject)
        {
            throw new JsonException("Expected JSON object.");
        }

        var result = new JsonObject();

        while (reader.Read())
        {
            if (reader.TokenType == JsonTokenType.EndObject)
            {
                return result;
            }

            if (reader.TokenType != JsonTokenType.PropertyName)
            {
                throw new JsonException("Expected JSON property name.");
            }

            var propertyName = reader.GetString() ?? throw new JsonException("Property name cannot be null.");
            reader.Read();
            result[propertyName] = ReadValue(ref reader, options);
        }

        throw new JsonException("Incomplete JSON object.");
    }

    public override void Write(Utf8JsonWriter writer, JsonObject value, JsonSerializerOptions options)
    {
        writer.WriteStartObject();

        foreach (var pair in value)
        {
            writer.WritePropertyName(pair.Key);
            JsonSerializer.Serialize(writer, pair.Value, options);
        }

        writer.WriteEndObject();
    }

    private static object? ReadValue(ref Utf8JsonReader reader, JsonSerializerOptions options)
    {
        switch (reader.TokenType)
        {
            case JsonTokenType.StartObject:
                return JsonSerializer.Deserialize<JsonObject>(ref reader, options);
            case JsonTokenType.StartArray:
                return ReadArray(ref reader, options);
            case JsonTokenType.String:
                return reader.GetString();
            case JsonTokenType.Number:
                if (reader.TryGetInt64(out var longValue))
                {
                    return longValue;
                }
                if (reader.TryGetDecimal(out var decimalValue))
                {
                    return decimalValue;
                }
                return reader.GetDouble();
            case JsonTokenType.True:
                return true;
            case JsonTokenType.False:
                return false;
            case JsonTokenType.Null:
                return null;
            default:
                throw new JsonException($"Unsupported token type {reader.TokenType}.");
        }
    }

    private static List<object?> ReadArray(ref Utf8JsonReader reader, JsonSerializerOptions options)
    {
        var values = new List<object?>();

        while (reader.Read())
        {
            if (reader.TokenType == JsonTokenType.EndArray)
            {
                return values;
            }

            values.Add(ReadValue(ref reader, options));
        }

        throw new JsonException("Incomplete JSON array.");
    }
}
