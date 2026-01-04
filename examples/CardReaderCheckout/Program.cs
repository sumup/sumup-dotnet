using SumUp;
using SumUp.Http;

Console.WriteLine("Starting card reader checkout example...");
Console.WriteLine("Ensure SUMUP_ACCESS_TOKEN and SUMUP_MERCHANT_CODE are set. Optionally set SUMUP_READER_ID to target a specific device.");

var merchantCode = GetRequiredEnv("SUMUP_MERCHANT_CODE");
var readerId = Environment.GetEnvironmentVariable("SUMUP_READER_ID");

using var client = new SumUpClient();

if (string.IsNullOrWhiteSpace(readerId))
{
    Console.WriteLine("SUMUP_READER_ID not provided; fetching the first paired reader...");
    readerId = await ResolveReaderIdAsync(client, merchantCode, CancellationToken.None);
}

Console.WriteLine($"Using reader: {readerId}");
var checkoutReference = $"checkout-{Guid.NewGuid():N}";
Console.WriteLine($"Creating checkout with reference: {checkoutReference}");

var request = new CreateReaderCheckoutRequest
{
    Description = "sumup-dotnet card reader checkout example",
    TotalAmount = new CreateReaderCheckoutRequestTotalAmount
    {
        Currency = "EUR",
        MinorUnit = 2,
        Value = 1000,
    },
};

try
{
    await client.Readers.CreateCheckoutAsync(merchantCode, readerId!, request);
    Console.WriteLine("✓ Checkout created successfully!");
}
catch (ApiException ex)
{
    Console.WriteLine($"✗ Failed to create checkout ({(int)ex.StatusCode}): {ex.Message}");
    if (!string.IsNullOrWhiteSpace(ex.ResponseBody))
    {
        Console.WriteLine(ex.ResponseBody);
    }
    return;
}

static string GetRequiredEnv(string name)
{
    var value = Environment.GetEnvironmentVariable(name);
    if (string.IsNullOrWhiteSpace(value))
    {
        throw new InvalidOperationException($"Environment variable '{name}' must be set.");
    }
    return value;
}

static async Task<string> ResolveReaderIdAsync(SumUpClient client, string merchantCode, CancellationToken cancellationToken)
{
    var readersResponse = await client.Readers.ListAsync(merchantCode, cancellationToken);
    var readers = readersResponse.Data?.Items ?? throw new InvalidOperationException("Reader list response was empty.");
    foreach (var reader in readers)
    {
        if (!string.IsNullOrWhiteSpace(reader.Id))
        {
            return reader.Id;
        }
    }
    throw new InvalidOperationException("Merchant does not have any paired readers. Provide SUMUP_READER_ID to override.");
}
