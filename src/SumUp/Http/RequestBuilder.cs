using System;
using System.Collections;
using System.Collections.Generic;
using System.Globalization;
using System.Net.Http;
using System.Text;

namespace SumUp.Http;

internal sealed class RequestBuilder
{
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

    private static string ConvertToString(object value) =>
        Convert.ToString(value, CultureInfo.InvariantCulture) ?? string.Empty;
}
