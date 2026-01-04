using System.Collections.Generic;
using System.Linq;
using SumUp;

Console.WriteLine("Fetching recent checkouts...");
Console.WriteLine("Ensure SUMUP_ACCESS_TOKEN is set before running this example.");

using var client = new SumUpClient();

try
{
    var checkoutsResponse = await client.Checkouts.ListAsync();
    var checkouts = checkoutsResponse.Data ?? Array.Empty<CheckoutSuccess>();

    if (checkouts.Any())
    {
        foreach (CheckoutSuccess checkout in checkouts)
        {
            Console.WriteLine($"Checkout {checkout.Id}: {checkout.Amount} {checkout.Currency}");
        }
    }
    else
    {
        Console.WriteLine("No checkouts returned.");
    }

    Console.WriteLine();
    Console.WriteLine("Fetching memberships filtered by status, resource type, and roles...");

    var membershipResponse = await client.Memberships.ListAsync(
        status: MembershipStatus.Accepted,
        resourceType: "merchant",
        limit: 5);

    var memberships = membershipResponse.Data?.Items ?? Array.Empty<Membership>();
    if (!memberships.Any())
    {
        Console.WriteLine("No memberships matched the provided filters.");
    }
    else
    {
        foreach (Membership membership in memberships)
        {
            var roles = string.Join(", ", membership.Roles ?? Array.Empty<string>());
            Console.WriteLine($"Membership {membership.Id} ({membership.Status}) in {membership.Resource.Name}: Roles [{roles}]");
        }
    }
}
catch (Exception ex)
{
    Console.WriteLine($"SumUp API call failed: {ex.Message}");
}
