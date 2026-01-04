using SumUp;

Console.WriteLine("Fetching recent checkouts...");
Console.WriteLine("Ensure SUMUP_ACCESS_TOKEN is set before running this example.");

using var client = new SumUpClient();

try
{
    var response = await client.Checkouts.ListAsync();
    if (response.Data is null)
    {
        Console.WriteLine("No checkouts returned.");
        return;
    }

    foreach (CheckoutSuccess checkout in response.Data)
    {
        Console.WriteLine($"Checkout {checkout.Id}: {checkout.Amount} {checkout.Currency}");
    }
}
catch (Exception ex)
{
    Console.WriteLine($"SumUp API call failed: {ex.Message}");
}
