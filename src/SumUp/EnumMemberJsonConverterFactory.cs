using System;
using System.Collections.Generic;
using System.Globalization;
using System.Linq;
using System.Reflection;
using System.Runtime.Serialization;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace SumUp;

/// <summary>
/// Serializes enums using <see cref="EnumMemberAttribute.Value"/> when present.
/// Replace this with built-in enum member name support once the SDK targets a runtime that supports it directly.
/// </summary>
public sealed class EnumMemberJsonConverterFactory : JsonConverterFactory
{
    /// <inheritdoc />
    public override bool CanConvert(Type typeToConvert)
    {
        var enumType = Nullable.GetUnderlyingType(typeToConvert) ?? typeToConvert;
        return enumType.IsEnum;
    }

    /// <inheritdoc />
    public override JsonConverter CreateConverter(Type typeToConvert, JsonSerializerOptions options)
    {
        var underlyingType = Nullable.GetUnderlyingType(typeToConvert);
        var enumType = underlyingType ?? typeToConvert;
        var converterType = underlyingType is null
            ? typeof(EnumMemberJsonConverter<>).MakeGenericType(enumType)
            : typeof(NullableEnumMemberJsonConverter<>).MakeGenericType(enumType);

        return (JsonConverter)Activator.CreateInstance(converterType)!;
    }

    private sealed class NullableEnumMemberJsonConverter<TEnum> : JsonConverter<TEnum?>
        where TEnum : struct, Enum
    {
        private static readonly EnumMemberJsonConverter<TEnum> InnerConverter = new();

        public override TEnum? Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
        {
            if (reader.TokenType == JsonTokenType.Null)
            {
                return null;
            }

            return InnerConverter.Read(ref reader, typeof(TEnum), options);
        }

        public override void Write(Utf8JsonWriter writer, TEnum? value, JsonSerializerOptions options)
        {
            if (value is null)
            {
                writer.WriteNullValue();
                return;
            }

            InnerConverter.Write(writer, value.Value, options);
        }
    }

    private sealed class EnumMemberJsonConverter<TEnum> : JsonConverter<TEnum>
        where TEnum : struct, Enum
    {
        private static readonly IReadOnlyDictionary<TEnum, string> WriteMappings = BuildWriteMappings();
        private static readonly IReadOnlyDictionary<string, TEnum> ReadMappings = BuildReadMappings(WriteMappings);

        public override TEnum Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
        {
            if (reader.TokenType == JsonTokenType.String)
            {
                var value = reader.GetString();
                if (value is not null)
                {
                    if (ReadMappings.TryGetValue(value, out var mapped))
                    {
                        return mapped;
                    }

                    if (Enum.TryParse<TEnum>(value, ignoreCase: true, out var parsed))
                    {
                        return parsed;
                    }
                }
            }
            else if (reader.TokenType == JsonTokenType.Number)
            {
                return ReadNumeric(ref reader);
            }

            throw new JsonException(
                string.Format(
                    CultureInfo.InvariantCulture,
                    "Unable to convert value to enum type '{0}'.",
                    typeof(TEnum).FullName));
        }

        public override void Write(Utf8JsonWriter writer, TEnum value, JsonSerializerOptions options)
        {
            if (WriteMappings.TryGetValue(value, out var mapped))
            {
                writer.WriteStringValue(mapped);
                return;
            }

            var name = Enum.GetName(typeof(TEnum), value);
            if (name is not null)
            {
                writer.WriteStringValue(name);
                return;
            }

            WriteNumeric(writer, value);
        }

        private static Dictionary<TEnum, string> BuildWriteMappings()
        {
            var values = new Dictionary<TEnum, string>();
            foreach (var field in typeof(TEnum).GetFields(BindingFlags.Public | BindingFlags.Static))
            {
                var enumValue = (TEnum)field.GetValue(null)!;
                var enumMember = field.GetCustomAttribute<EnumMemberAttribute>();
                values[enumValue] = enumMember?.Value ?? field.Name;
            }

            return values;
        }

        private static Dictionary<string, TEnum> BuildReadMappings(IReadOnlyDictionary<TEnum, string> writeMappings)
        {
            var values = new Dictionary<string, TEnum>(StringComparer.OrdinalIgnoreCase);
            foreach (var pair in writeMappings)
            {
                values[pair.Value] = pair.Key;
                values[pair.Key.ToString()] = pair.Key;
            }

            foreach (var value in Enum.GetValues(typeof(TEnum)).Cast<TEnum>())
            {
                values[value.ToString()] = value;
            }

            return values;
        }

        private static TEnum ReadNumeric(ref Utf8JsonReader reader)
        {
            object value = Type.GetTypeCode(Enum.GetUnderlyingType(typeof(TEnum))) switch
            {
                TypeCode.SByte => reader.GetSByte(),
                TypeCode.Byte => reader.GetByte(),
                TypeCode.Int16 => reader.GetInt16(),
                TypeCode.UInt16 => reader.GetUInt16(),
                TypeCode.Int32 => reader.GetInt32(),
                TypeCode.UInt32 => reader.GetUInt32(),
                TypeCode.Int64 => reader.GetInt64(),
                TypeCode.UInt64 => reader.GetUInt64(),
                _ => throw new JsonException(
                    string.Format(
                        CultureInfo.InvariantCulture,
                        "Enum type '{0}' has an unsupported underlying type.",
                        typeof(TEnum).FullName)),
            };

            return (TEnum)Enum.ToObject(typeof(TEnum), value);
        }

        private static void WriteNumeric(Utf8JsonWriter writer, TEnum value)
        {
            switch (Type.GetTypeCode(Enum.GetUnderlyingType(typeof(TEnum))))
            {
                case TypeCode.SByte:
                    writer.WriteNumberValue(Convert.ToSByte(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.Byte:
                    writer.WriteNumberValue(Convert.ToByte(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.Int16:
                    writer.WriteNumberValue(Convert.ToInt16(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.UInt16:
                    writer.WriteNumberValue(Convert.ToUInt16(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.Int32:
                    writer.WriteNumberValue(Convert.ToInt32(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.UInt32:
                    writer.WriteNumberValue(Convert.ToUInt32(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.Int64:
                    writer.WriteNumberValue(Convert.ToInt64(value, CultureInfo.InvariantCulture));
                    return;
                case TypeCode.UInt64:
                    writer.WriteNumberValue(Convert.ToUInt64(value, CultureInfo.InvariantCulture));
                    return;
                default:
                    throw new JsonException(
                        string.Format(
                            CultureInfo.InvariantCulture,
                            "Enum type '{0}' has an unsupported underlying type.",
                            typeof(TEnum).FullName));
            }
        }
    }
}
