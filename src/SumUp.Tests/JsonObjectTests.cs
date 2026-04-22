using System;
using System.Collections.Generic;
using System.IO;
using System.Net.Http;
using System.Text;
using SumUp.Http;
using Xunit;

namespace SumUp.Tests;

public class JsonObjectTests
{
    [Fact]
    public void CreateContent_SerializesJsonObjectPayloads()
    {
        using var httpClient = new HttpClient { BaseAddress = new Uri("https://api.sumup.com") };
        var apiClient = new ApiClient(httpClient, new SumUpClientOptions());
        var request = new ProcessCheckout
        {
            PaymentType = ProcessCheckoutPaymentType.Card,
            ApplePay = new JsonObject
            {
                ["token"] = new JsonObject
                {
                    ["version"] = "EC_v1"
                },
                ["sandbox"] = true,
            },
        };

        using var content = apiClient.CreateContent(request, "application/json");
        using var stream = content.ReadAsStream();
        using var reader = new StreamReader(stream, Encoding.UTF8);
        var body = reader.ReadToEnd();

        Assert.Contains("\"apple_pay\":{\"token\":{\"version\":\"EC_v1\"},\"sandbox\":true}", body);
    }

    [Fact]
    public void Parse_ReadsNestedObjectsAndArrays()
    {
        var json = """
            {
              "status": "ok",
              "count": 2,
              "items": [
                "one",
                {
                  "nested": true
                }
              ]
            }
            """;

        var result = JsonObject.Parse(json);

        Assert.Equal("ok", result.GetValue<string>("status"));
        Assert.Equal(2L, result.GetValue<long>("count"));

        Assert.True(result.TryGetValue("items", out var itemsValue));
        var items = Assert.IsType<List<object?>>(itemsValue);
        Assert.Equal("one", Assert.IsType<string>(items[0]));

        var nested = Assert.IsType<JsonObject>(items[1]);
        Assert.True(nested.GetValue<bool>("nested"));
    }
}
