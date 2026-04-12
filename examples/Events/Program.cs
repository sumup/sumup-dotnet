using SumUp;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();
using var client = new SumUpClient();
var secret = Environment.GetEnvironmentVariable("SUMUP_EVENT_SECRET")
    ?? throw new InvalidOperationException("Set SUMUP_EVENT_SECRET to your endpoint signing secret.");
var events = client.CreateEventsHandler(secret, (notification, _) =>
{
    app.Logger.LogInformation("Unhandled event {Id}: {Type}", notification.Id, notification.Type);
    return Task.CompletedTask;
});
events.OnMemberUpdated(async (notification, cancellationToken) =>
{
    var member = await notification.FetchObjectAsync(cancellationToken: cancellationToken);
    app.Logger.LogInformation("Member updated: {Id}", member?.Id);
});

app.MapPost("/events", async (HttpRequest request, CancellationToken cancellationToken) =>
{
    using var body = new MemoryStream();
    await request.Body.CopyToAsync(body, cancellationToken);
    try
    {
        await events.HandleAsync(body.ToArray(), request.Headers[EventSignature.HeaderName].ToString(), cancellationToken);
        return Results.NoContent();
    }
    catch (EventSignatureException)
    {
        return Results.BadRequest();
    }
    catch (EventPayloadException)
    {
        return Results.BadRequest();
    }
    // Callback failures produce HTTP 500 so delivery can be retried.
});

app.Run("http://localhost:5000");
