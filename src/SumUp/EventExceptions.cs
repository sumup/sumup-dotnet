namespace SumUp;

/// <summary>The event signature is missing, malformed, or does not match the request body.</summary>
public class EventSignatureException(string message, Exception? innerException = null) : Exception(message, innerException);

/// <summary>The signed delivery timestamp is outside the five-minute acceptance window.</summary>
public sealed class EventSignatureExpiredException(string message) : EventSignatureException(message);

/// <summary>The event body could not be deserialized.</summary>
public sealed class EventPayloadException(string message, Exception? innerException = null) : Exception(message, innerException);

/// <summary>An event callback failed. Inspect <see cref="Exception.InnerException"/> for the original exception.</summary>
public sealed class EventCallbackException(string message, Exception innerException) : Exception(message, innerException);

/// <summary>The event resource cannot be fetched through this client.</summary>
public sealed class EventObjectException(string message) : Exception(message);
