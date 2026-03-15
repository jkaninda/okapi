## Okapi SSE (Server-Sent Events) Skills

Okapi provides built-in SSE support for real-time server-to-client streaming over HTTP. SSE connections are long-lived GET requests where the server pushes events using the `text/event-stream` content type. The framework handles headers, flushing, keep-alive pings, and serialization automatically.

### Core Types

| Type | Description |
|------|-------------|
| `Message` | Single SSE message with `ID`, `Event`, `Data`, `Retry`, `Serializer` fields |
| `StreamOptions` | Advanced stream config: `Serializer`, `PingInterval`, `OnError` callback |
| `Serializer` | Interface: `Serialize(data any) (string, error)` — custom data encoding |
| `SendFunc` | `func(data any, eventType string) (string, error)` |

### Built-in Serializers

| Serializer | Description |
|------------|-------------|
| `okapi.JSONSerializer` | Default — marshals data as JSON |
| `okapi.TextSerializer` | Plain text via `fmt.Sprintf("%v", data)` |
| `okapi.Base64Serializer` | Base64-encodes `[]byte` data |

### Sending Single Events

```go
// Fire-and-forget event (no ID, auto-generated internally)
c.SSEvent("message", data)

// Event with explicit ID
c.SSESendEvent("event-1", "message", data)

// Data-only (no event type)
c.SSESendData(data)

// Explicit serializer variants
c.SSESendJSON(data)            // JSON serializer
c.SSESendText("hello")         // Text serializer
c.SSESendBinary([]byte{...})   // Base64 serializer
c.SendSSECustom(data, mySerializer)  // Custom serializer
```

### Simple Event Loop (Recommended for Most Cases)

The simplest pattern — use a ticker and send events directly in a loop:

```go
app.Get("/events", func(c *okapi.Context) error {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    ctx := c.Request().Context()
    for {
        select {
        case <-ctx.Done():
            return nil
        case t := <-ticker.C:
            data := okapi.M{
                "time":      t.Format("15:04:05"),
                "timestamp": t.Unix(),
            }
            if err := c.SSEvent("message", data); err != nil {
                return err
            }
        }
    }
})
```

### Channel-Based Streaming

For producer/consumer patterns, use `SSEStream` with a message channel:

```go
app.Get("/events", func(c *okapi.Context) error {
    ctx, cancel := context.WithCancel(c.Request().Context())
    defer cancel()

    messageChan := make(chan okapi.Message, 10)

    // Producer goroutine
    go func() {
        defer close(messageChan)
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return
            case t := <-ticker.C:
                messageChan <- okapi.Message{
                    Event: "update",
                    Data:  okapi.M{"time": t.Format("15:04:05"), "count": i},
                }
            }
        }
    }()

    return c.SSEStream(ctx, messageChan)
})
```

### Advanced Streaming with Options

Use `SSEStreamWithOptions` for keep-alive pings, custom serializers, and error handling:

```go
app.Get("/events", func(c *okapi.Context) error {
    ctx, cancel := context.WithCancel(c.Request().Context())
    defer cancel()

    messageChan := make(chan okapi.Message, 10)

    go func() {
        defer close(messageChan)
        // ... produce messages ...
    }()

    return c.SSEStreamWithOptions(ctx, messageChan, &okapi.StreamOptions{
        Serializer:   &okapi.JSONSerializer{},
        PingInterval: 30 * time.Second,
        OnError: func(err error) {
            slog.Error("SSE stream error", "error", err)
        },
    })
})
```

### Detecting SSE Requests

```go
if c.IsSSE() {
    // Request has Accept: text/event-stream and method is GET
}
```

### SSE Message Fields

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Message identifier (auto-generated UUID if empty) |
| `Event` | `string` | Event type (client listens via `addEventListener`) |
| `Data` | `any` | Payload — auto-serialized based on type or `Serializer` |
| `Retry` | `uint` | Client reconnection interval in milliseconds |
| `Serializer` | `Serializer` | Per-message serializer (overrides stream-level) |

### StreamOptions Fields

| Field | Type | Description |
|-------|------|-------------|
| `Serializer` | `Serializer` | Default serializer for all messages in stream |
| `PingInterval` | `time.Duration` | Keep-alive ping interval (sends `: ping` comments) |
| `OnError` | `func(error)` | Error callback for send/ping failures |

### Custom Serializer

Implement the `Serializer` interface for custom data formats:

```go
type XMLSerializer struct{}

func (x XMLSerializer) Serialize(data any) (string, error) {
    b, err := xml.Marshal(data)
    if err != nil {
        return "", err
    }
    return string(b), nil
}

// Use per-message
c.SendSSECustom(data, XMLSerializer{})

// Or stream-level
c.SSEStreamWithOptions(ctx, ch, &okapi.StreamOptions{
    Serializer: XMLSerializer{},
})
```

### Client-Side (JavaScript)

```javascript
const eventSource = new EventSource('/events');

// Listen for named events
eventSource.addEventListener('message', (event) => {
    const data = JSON.parse(event.data);
    console.log(data);
});

// Handle errors and reconnect
eventSource.onerror = () => {
    eventSource.close();
    setTimeout(() => { /* reconnect */ }, 3000);
};
```