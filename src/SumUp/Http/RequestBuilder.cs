using System;
using System.Collections;
using System.Collections.Generic;
using System.Collections.Concurrent;
using System.Globalization;
using System.Reflection;
using System.Runtime.Serialization;
using System.Net.Http;
using System.Text;

namespace SumUp.Http;

internal sealed class RequestBuilder
{
    private static readonly ConcurrentDictionary<Type, IReadOnlyDictionary<object, string>> EnumValueMappings = new();
    private readonly HttpMethod _method;
    private readonly string _pathTemplate;
    private readonly Uri _baseAddress;
    private readonly Dictionary<string, string> _pathParameters = new(StringComparer.OrdinalIgnoreCase);
    private readonly List<KeyValuePair<string, string>> _query = new();
    private readonly List<KeyValuePair<string, string>> _headers = new();

    internal RequestBuilder(HttpMethod method, string pathTemplate, Uri baseAddress)
    {
        _method = method;
        _pathTemplate = pathTemplate;
        _baseAddress = baseAddress;
    }

    internal void AddPath(string name, object? value)
    {
        if (value is null)
        {
            throw new ArgumentNullException(name);
        }

        _pathParameters[name] = ConvertToString(value);
    }

    internal void AddQuery(string name, object? value)
    {
        if (value is null)
        {
            return;
        }

        if (value is string stringValue)
        {
            _query.Add(new KeyValuePair<string, string>(name, stringValue));
            return;
        }

        if (value is IEnumerable enumerable and not string)
        {
            foreach (var entry in enumerable)
            {
                if (entry is null)
                {
                    continue;
                }

                _query.Add(new KeyValuePair<string, string>(name, ConvertToString(entry)));
            }

            return;
        }

        _query.Add(new KeyValuePair<string, string>(name, ConvertToString(value)));
    }

    internal void AddQuery<T>(string name, OptionalQuery<T> value)
    {
        if (!value.IsSet)
        {
            return;
        }

        if (value.IsNull)
        {
            _query.Add(new KeyValuePair<string, string>(name, "null"));
            return;
        }

        AddQuery(name, value.RawValue);
    }

    internal void AddHeader(string name, object? value)
    {
        if (value is null)
        {
            return;
        }

        if (value is IEnumerable enumerable and not string)
        {
            foreach (var entry in enumerable)
            {
                if (entry is null)
                {
                    continue;
                }

                _headers.Add(new KeyValuePair<string, string>(name, ConvertToString(entry)));
            }

            return;
        }

        _headers.Add(new KeyValuePair<string, string>(name, ConvertToString(value)));
    }

    internal HttpRequestMessage Build()
    {
        var path = _pathTemplate;
        foreach (var kvp in _pathParameters)
        {
            var key = kvp.Key;
            var value = kvp.Value;
            path = path.Replace($"{{{key}}}", Uri.EscapeDataString(value));
        }

        var uriBuilder = new UriBuilder(new Uri(_baseAddress, path));
        if (_query.Count > 0)
        {
            var queryBuilder = new StringBuilder();
            for (var i = 0; i < _query.Count; i++)
            {
                if (i > 0)
                {
                    queryBuilder.Append('&');
                }

                queryBuilder.Append(Uri.EscapeDataString(_query[i].Key));
                queryBuilder.Append('=');
                queryBuilder.Append(Uri.EscapeDataString(_query[i].Value));
            }

            uriBuilder.Query = queryBuilder.ToString();
        }

        var request = new HttpRequestMessage(_method, uriBuilder.Uri);
        foreach (var kvp in _headers)
        {
            request.Headers.TryAddWithoutValidation(kvp.Key, kvp.Value);
        }

        return request;
    }

    private static string ConvertToString(object value)
    {
        if (value is DateOnly dateOnly)
        {
            return dateOnly.ToString("yyyy-MM-dd", CultureInfo.InvariantCulture);
        }

        if (value is TimeOnly timeOnly)
        {
            return timeOnly.ToString(@"HH\:mm\:ss\.FFFFFFF", CultureInfo.InvariantCulture).TrimEnd('0').TrimEnd('.');
        }

        if (value is DateTimeOffset dateTimeOffset)
        {
            return dateTimeOffset.ToString("O", CultureInfo.InvariantCulture);
        }

        if (value is DateTime dateTime)
        {
            return dateTime.ToString("O", CultureInfo.InvariantCulture);
        }

        var type = value.GetType();
        if (type.IsEnum)
        {
            return ConvertEnumToString(type, value);
        }

        return Convert.ToString(value, CultureInfo.InvariantCulture) ?? string.Empty;
    }

    private static string ConvertEnumToString(Type enumType, object value)
    {
        var mappings = EnumValueMappings.GetOrAdd(enumType, static type =>
        {
            var values = new Dictionary<object, string>();
            foreach (var field in type.GetFields(BindingFlags.Public | BindingFlags.Static))
            {
                var enumValue = field.GetValue(null);
                if (enumValue is null)
                {
                    continue;
                }

                var enumMember = field.GetCustomAttribute<EnumMemberAttribute>();
                values[enumValue] = enumMember?.Value ?? field.Name;
            }

            return values;
        });

        if (mappings.TryGetValue(value, out var mapped))
        {
            return mapped;
        }

        return Convert.ToString(value, CultureInfo.InvariantCulture) ?? string.Empty;
    }
}
