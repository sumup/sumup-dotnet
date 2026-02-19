using System;
using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Xunit;

namespace SumUp.Tests;

public class ReadersClientTests
{
    private const string ReaderResponseBody = """
    {
      "id": "reader-456",
      "name": "Solo",
      "status": "paired",
      "device": {
        "identifier": "device-123",
        "model": "solo"
      },
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
    """;

    private const string ReadersListResponseBody = """
    {
      "items": [
        {
          "id": "rdr_123",
          "name": "READER01",
          "status": "paired",
          "device": {
            "identifier": "device-123",
            "model": "solo"
          },
          "created_at": "2026-02-18T16:16:38.097244Z",
          "updated_at": "2026-02-18T16:16:38.097244Z"
        }
      ]
    }
    """;

    [Fact]
    public async Task ListAsync_ParsesItemsResponse()
    {
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(ReadersListResponseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        using var client = new SumUpClient(new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "test-token"
        });

        var apiResponse = await client.Readers.ListAsync("merchant-123", cancellationToken: CancellationToken.None);

        Assert.Equal(HttpStatusCode.OK, apiResponse.StatusCode);
        var data = Assert.IsType<ReadersListResponse>(apiResponse.Data);
        var reader = Assert.Single(data.Items);
        Assert.Equal("rdr_123", reader.Id);
        Assert.Equal("READER01", reader.Name);
        Assert.Equal(ReaderStatus.Paired, reader.Status);
    }

    [Fact]
    public async Task GetStatusAsync_SendsProperRequestAndParsesResponse()
    {
        const string responseBody = """
        {
          "data": {
            "battery_level": 82.5,
            "battery_temperature": 29,
            "connection_type": "wifi",
            "firmware_version": "3.3.40",
            "last_activity": "2024-05-23T13:45:00Z",
            "state": "WAITING_FOR_CARD",
            "status": "ONLINE"
          }
        }
        """;

        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        using var client = new SumUpClient(new SumUpClientOptions
        {
            HttpClient = httpClient
        });

        var apiResponse = await client.Readers.GetStatusAsync(
            merchantCode: "merchant-123",
            readerId: "reader-456",
            accept: "application/vnd.sumup.status+json",
            contentType: "application/vnd.sumup.status+json",
            authorization: "Bearer test-token",
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        Assert.NotNull(handler.LastRequest);
        var request = handler.LastRequest!;
        Assert.Equal(HttpMethod.Get, request.Method);
        Assert.Equal("/v0.1/merchants/merchant-123/readers/reader-456/status", request.RequestUri!.AbsolutePath);
        Assert.Equal("application/json", Assert.Single(request.Headers.Accept).MediaType);
        Assert.Null(request.Content);
        Assert.True(request.Headers.TryGetValues("Authorization", out var authorizationHeaders));
        Assert.Contains("Bearer test-token", authorizationHeaders);

        Assert.Equal(HttpStatusCode.OK, apiResponse.StatusCode);
        Assert.NotNull(apiResponse.Data);
        var statusResponse = apiResponse.Data!;
        Assert.NotNull(statusResponse.Data);
        var statusData = statusResponse.Data!;
        Assert.Equal(82.5f, statusData.BatteryLevel);
        Assert.Equal(29, statusData.BatteryTemperature);
        Assert.Equal("wifi", statusData.ConnectionType);
        Assert.Equal("3.3.40", statusData.FirmwareVersion);
        var expectedLastActivity = DateTimeOffset.Parse("2024-05-23T13:45:00+00:00", CultureInfo.InvariantCulture);
        Assert.Equal(expectedLastActivity, statusData.LastActivity);
        Assert.Equal("WAITING_FOR_CARD", statusData.State);
        Assert.Equal("ONLINE", statusData.Status);
    }

    [Fact]
    public async Task Requests_IncludeDefaultUserAgentHeader()
    {
        const string responseBody = """{"data":{"status":"ONLINE"}}""";
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        var options = new SumUpClientOptions
        {
            HttpClient = httpClient
        };

        using var client = new SumUpClient(options);

        await client.Readers.GetStatusAsync(
            merchantCode: "merchant-123",
            readerId: "reader-456",
            accept: "application/json",
            contentType: "application/json",
            authorization: "Bearer test-token",
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        var request = Assert.IsType<HttpRequestMessage>(handler.LastRequest);
        Assert.Equal(options.UserAgent, request.Headers.UserAgent.ToString());
    }

    [Fact]
    public async Task Requests_IncludeCustomUserAgentHeader()
    {
        const string responseBody = """{"data":{"status":"ONLINE"}}""";
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        const string customAgent = "my-app/2.3.4";
        var options = new SumUpClientOptions
        {
            HttpClient = httpClient,
            UserAgent = customAgent
        };

        using var client = new SumUpClient(options);

        await client.Readers.GetStatusAsync(
            merchantCode: "merchant-123",
            readerId: "reader-456",
            accept: "application/json",
            contentType: "application/json",
            authorization: "Bearer test-token",
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        var request = Assert.IsType<HttpRequestMessage>(handler.LastRequest);
        Assert.Equal(customAgent, request.Headers.UserAgent.ToString());
    }

    [Fact]
    public async Task Requests_UseDefaultAccessToken()
    {
        const string token = "default-token";
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(ReaderResponseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        var options = new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = token
        };

        using var client = new SumUpClient(options);

        await client.Readers.GetAsync(
            merchantCode: "merchant-123",
            id: "reader-456",
            requestOptions: null,
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        var request = Assert.IsType<HttpRequestMessage>(handler.LastRequest);
        Assert.Equal($"Bearer {token}", request.Headers.Authorization?.ToString());
    }

    [Fact]
    public async Task Requests_CanOverrideAccessTokenPerRequest()
    {
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(ReaderResponseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        var options = new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "default-token"
        };

        using var client = new SumUpClient(options);

        var requestOptions = new RequestOptions
        {
            AccessToken = "request-token"
        };

        await client.Readers.GetAsync(
            merchantCode: "merchant-123",
            id: "reader-456",
            requestOptions: requestOptions,
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        var request = Assert.IsType<HttpRequestMessage>(handler.LastRequest);
        Assert.Equal("Bearer request-token", request.Headers.Authorization?.ToString());
    }

    [Fact]
    public async Task Requests_PreserveExistingAuthorizationHeaders()
    {
        const string responseBody = """{"data":{"status":"ONLINE"}}""";
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new RecordingHttpMessageHandler(_ =>
        {
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(responseBody, Encoding.UTF8, "application/json")
            };
        });

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        var options = new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "default-token"
        };

        using var client = new SumUpClient(options);

        await client.Readers.GetStatusAsync(
            merchantCode: "merchant-123",
            readerId: "reader-456",
            accept: "application/json",
            contentType: "application/json",
            authorization: "Bearer provided-token",
            cancellationToken: CancellationToken.None);

        Assert.Equal(1, handler.SendCount);
        var request = Assert.IsType<HttpRequestMessage>(handler.LastRequest);
        Assert.Equal("Bearer provided-token", request.Headers.Authorization?.ToString());
    }

    [Fact]
    public async Task Requests_RespectPerRequestTimeout()
    {
        using var accessTokenScope = new EnvironmentVariableScope("SUMUP_ACCESS_TOKEN", null);
        var handler = new DelayedHttpMessageHandler(TimeSpan.FromSeconds(5));

        using var httpClient = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://mocked.sumup.test/")
        };

        var options = new SumUpClientOptions
        {
            HttpClient = httpClient,
            AccessToken = "default-token"
        };

        using var client = new SumUpClient(options);

        var requestOptions = new RequestOptions
        {
            Timeout = TimeSpan.FromMilliseconds(50)
        };

        await Assert.ThrowsAsync<TaskCanceledException>(() => client.Readers.GetAsync(
            merchantCode: "merchant-123",
            id: "reader-456",
            requestOptions: requestOptions,
            cancellationToken: CancellationToken.None));
    }

    private sealed class EnvironmentVariableScope : IDisposable
    {
        private readonly string _name;
        private readonly string? _originalValue;

        internal EnvironmentVariableScope(string name, string? value)
        {
            _name = name;
            _originalValue = Environment.GetEnvironmentVariable(name);
            Environment.SetEnvironmentVariable(name, value);
        }

        public void Dispose()
        {
            Environment.SetEnvironmentVariable(_name, _originalValue);
        }
    }

    private sealed class RecordingHttpMessageHandler : HttpMessageHandler
    {
        private readonly Func<HttpRequestMessage, HttpResponseMessage> _responseFactory;

        internal RecordingHttpMessageHandler(Func<HttpRequestMessage, HttpResponseMessage> responseFactory)
        {
            _responseFactory = responseFactory;
        }

        internal HttpRequestMessage? LastRequest { get; private set; }

        internal int SendCount { get; private set; }

        protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            LastRequest = request;
            SendCount++;
            return Task.FromResult(_responseFactory(request));
        }
    }

    private sealed class DelayedHttpMessageHandler : HttpMessageHandler
    {
        private readonly TimeSpan _delay;

        internal DelayedHttpMessageHandler(TimeSpan delay)
        {
            _delay = delay;
        }

        protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
        {
            await Task.Delay(_delay, cancellationToken);
            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(ReaderResponseBody, Encoding.UTF8, "application/json")
            };
        }
    }
}
