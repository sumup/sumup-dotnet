using System.Text;
using SumUp;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddSingleton<SumUpClient>(_ => new SumUpClient());

var app = builder.Build();

app.MapPost("/webhooks/sumup", async (HttpRequest request, SumUpClient client) =>
{
    using var reader = new StreamReader(request.Body, Encoding.UTF8);
    var body = await reader.ReadToEndAsync();

    try
    {
        var webhookHandler = client.CreateWebhookHandler();
        var eventNotification = webhookHandler.VerifyAndParse(
            request.Headers[WebhookConstants.SignatureHeader].ToString(),
            request.Headers[WebhookConstants.TimestampHeader].ToString(),
            body);

        switch (eventNotification)
        {
            case CheckoutCreatedEvent checkoutCreated:
                {
                    var checkout = await checkoutCreated.FetchObjectAsync();
                    Console.WriteLine($"Received checkout.created for checkout {checkout?.Id}");
                    break;
                }
            case CheckoutProcessedEvent checkoutProcessed:
                {
                    var checkout = await checkoutProcessed.FetchObjectAsync();
                    Console.WriteLine($"Received checkout.processed for checkout {checkout?.Id}");
                    break;
                }
            case MemberCreatedEvent memberCreated:
                {
                    var member = await memberCreated.FetchObjectAsync();
                    Console.WriteLine($"Received member.created for member {member?.Id}");
                    break;
                }
            default:
                Console.WriteLine($"Received webhook {eventNotification.Type} for object {eventNotification.Object.Id}");
                break;
        }

        return Results.Ok();
    }
    catch (WebhookException exception)
    {
        Console.Error.WriteLine($"Webhook verification failed: {exception.Message}");
        return Results.BadRequest(new { error = exception.Message });
    }
});

app.MapGet("/", () => Results.Text("POST SumUp webhooks to /webhooks/sumup"));

Console.WriteLine("Listening on http://localhost:5000");
Console.WriteLine($"Set {WebhookConstants.SecretEnvironmentVariable} and SUMUP_ACCESS_TOKEN before sending live webhooks.");

await app.RunAsync("http://localhost:5000");
