using System;
using System.Text.Json;
using Xunit;

namespace SumUp.Tests;

public class ModelDeserializationTests
{
    [Fact]
    public void Checkout_ReadOnlyId_IsDeserialized()
    {
        const string json = """
            {
              "id": "chk_123",
              "amount": 10.0,
              "checkout_reference": "order-123",
              "currency": "EUR"
            }
            """;

        var checkout = JsonSerializer.Deserialize<Checkout>(json);

        Assert.NotNull(checkout);
        Assert.Equal("chk_123", checkout!.Id);
    }

    [Fact]
    public void CheckoutSuccess_ReadOnlyProperties_AreDeserialized()
    {
        const string json = """
            {
              "id": "chk_456",
              "transaction_id": "tx_123",
              "transaction_code": "ABC123",
              "currency": "EUR"
            }
            """;

        var checkout = JsonSerializer.Deserialize<CheckoutSuccess>(json);

        Assert.NotNull(checkout);
        Assert.Equal("chk_456", checkout!.Id);
        Assert.Equal("tx_123", checkout.TransactionId);
        Assert.Equal("ABC123", checkout.TransactionCode);
    }

    [Fact]
    public void PersonalDetails_DateOnlyProperties_AreDeserialized()
    {
        const string json = """
            {
              "birth_date": "1980-01-12"
            }
            """;

        var personalDetails = JsonSerializer.Deserialize<PersonalDetails>(json);

        Assert.NotNull(personalDetails);
        Assert.Equal(new DateOnly(1980, 1, 12), personalDetails!.BirthDate);
    }
}
