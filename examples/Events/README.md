# Events with ASP.NET Core

```sh
export SUMUP_ACCESS_TOKEN="your_api_key"
export SUMUP_EVENT_SECRET="your_endpoint_signing_secret"
dotnet run --project examples/Events
```

Forward event deliveries to `POST http://localhost:5000/events`.
The example verifies the raw body, dispatches typed callbacks, and acknowledges successful processing with HTTP 204.

Return a success response only after processing succeeds. Deliveries may repeat;
use the event ID to make your application's processing idempotent.
Resource fetches return the current state; deleted resources may return an API error.
Configure request body limits in your web server as appropriate for your deployment.
