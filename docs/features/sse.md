---
title: Server Sent Events (SSE)
layout: default
parent: Features
nav_order: 9
---

# Server Sent Events (SSE)

Okapi provides built-in support for Server Sent Events (SSE), allowing you to easily implement real-time data streaming from the server to clients. SSE enables one-way communication where the server can push updates to connected clients without polling.

## Table of Contents
- [Quick Start](#quick-start)
- [Basic Examples](#basic-examples)
- [Advanced Features](#advanced-features)
- [Custom Serializers](#custom-serializers)
- [Stream Options](#stream-options)
- [Best Practices](#best-practices)

## Quick Start

### Simple Time Stream

```go
package main

import (
    "fmt"
    "github.com/jkaninda/okapi"
    "time"
)

func main() {
    o := okapi.Default()

    o.Get("/time", func(c *okapi.Context) error {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case t := <-ticker.C:
                err := c.SSESendText(t.Format(time.RFC3339))
                if err != nil {
                    return err
                }
            case <-c.Request().Context().Done():
                return nil
            }
        }
    })

	err := o.Start()
	if err != nil {
		panic(err) 
	}
}
```

## Basic Examples

### 1. Simple Data Stream

```go
o.Get("/counter", func(c *okapi.Context) error {
    for i := 0; i <= 10; i++ {
        err := c.SSESendData(i)
        if err != nil {
            return err
        }
        time.Sleep(1 * time.Second)
    }
    return nil
})
```

### 2. JSON Data Stream

```go
type StockUpdate struct {
    Name string  `json:"name"`
    Price  float64 `json:"price"`
    Time   string  `json:"time"`
}

o.Get("/stocks", func(c *okapi.Context) error {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            update := StockUpdate{
                Name: "STOCK-XYZ",
                Price:  150.25 + float64(time.Now().Second()),
                Time:   time.Now().Format(time.RFC3339),
            }
            
            err := c.SSESendJSON(update)
            if err != nil {
                return err
            }
            
        case <-c.Request().Context().Done():
            return nil
        }
    }
})
```

### 3. Custom Event Types

```go
o.Get("/notifications", func(c *okapi.Context) error {
    notifications := []struct {
        Type string
        Data okapi.M
    }{
        {"info", okapi.M{"message": "System started"}},
        {"warning", okapi.M{"message": "High CPU usage"}},
        {"error", okapi.M{"message": "Database connection failed"}},
    }

    for _, notif := range notifications {
        err := c.SSEvent(notif.Type, notif.Data)
        if err != nil {
            return err
        }
        time.Sleep(2 * time.Second)
    }
    return nil
})
```

### 4. Events with IDs

```go
o.Get("/messages", func(c *okapi.Context) error {
    for i := 0; i < 5; i++ {
        id := fmt.Sprintf("msg-%d", i)
        data := okapi.M{
            "text": fmt.Sprintf("Message %d", i),
            "timestamp": time.Now().Unix(),
        }
        
        err := c.SSESendEvent(id, "message", data)
        if err != nil {
            return err
        }
        time.Sleep(1 * time.Second)
    }
    return nil
})
```

## Advanced Features

### 1. Channel-Based Streaming

```go
o.Get("/events", func(c *okapi.Context) error {
    messageChan := make(chan okapi.Message, 10)
    ctx := c.Request().Context()

    // Producer goroutine
    go func() {
        defer close(messageChan)
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for i := 0; i < 10; i++ {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                messageChan <- okapi.Message{
                    Event: "update",
                    Data: okapi.M{
                        "count": i,
                        "time": time.Now().Format(time.RFC3339),
                    },
                }
            }
        }
    }()

    return c.SSEStream(ctx, messageChan)
})
```

### 2. Stream with Options (Ping & Error Handling)

```go
o.Get("/live-data", func(c *okapi.Context) error {
    messageChan := make(chan okapi.Message, 10)
    ctx := c.Request().Context()

    opts := &okapi.StreamOptions{
        PingInterval: 30 * time.Second, // Keep connection alive
        Serializer:   okapi.JSONSerializer{},
        OnError: func(err error) {
            fmt.Printf("Stream error: %v\n", err)
        },
    }

    // Producer goroutine
    go func() {
        defer close(messageChan)
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                messageChan <- okapi.Message{
                    Event: "data",
                    Data: okapi.M{
                        "temperature": 20 + time.Now().Second()%10,
                        "humidity": 60 + time.Now().Second()%20,
                    },
                }
            }
        }
    }()

    return c.SSEStreamWithOptions(ctx, messageChan, opts)
})
```

### 3. Binary Data Streaming

```go
o.Get("/images", func(c *okapi.Context) error {
    // Simulate sending binary data
    imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header example
    
    err := c.SSESendBinary(imageData)
    if err != nil {
        return err
    }
    
    return nil
})
```

## Custom Serializers

### Create a Custom Serializer

```go
// XMLSerializer for XML data
type XMLSerializer struct{}

func (x XMLSerializer) Serialize(data any) (string, error) {
    xmlBytes, err := xml.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("failed to encode data as XML: %w", err)
    }
    return string(xmlBytes), nil
}

// Usage
o.Get("/xml-events", func(c *okapi.Context) error {
    data := struct {
        XMLName xml.Name `xml:"update"`
        Message string   `xml:"message"`
        Time    string   `xml:"time"`
    }{
        Message: "Hello World",
        Time:    time.Now().Format(time.RFC3339),
    }

    return c.SendSSECustom(data, XMLSerializer{})
})
```

## Complete Example with Client

```go
package main

import (
    "fmt"
    "github.com/jkaninda/okapi"
    "net/http"
    "time"
)

const template = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.title}}</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; }
        #events { border: 1px solid #ccc; padding: 20px; height: 400px; overflow-y: auto; }
        .event { margin: 10px 0; padding: 10px; background: #f5f5f5; border-radius: 4px; }
        .event-type { font-weight: bold; color: #007bff; }
    </style>
</head>
<body>
    <h1>{{.title}}</h1>
    <div id="status">Connecting...</div>
    <div id="events"></div>
    
    <script>
        const eventsDiv = document.getElementById('events');
        const statusDiv = document.getElementById('status');
        const eventSource = new EventSource('{{.eventURL}}');
        
        eventSource.onopen = function() {
            statusDiv.textContent = 'Connected ✓';
            statusDiv.style.color = 'green';
        };
        
        eventSource.onmessage = function(event) {
            addEvent('message', event.data);
        };
        
        // Listen for custom event types
        eventSource.addEventListener('update', function(event) {
            addEvent('update', event.data);
        });
        
        eventSource.addEventListener('notification', function(event) {
            addEvent('notification', event.data);
        });
        
        eventSource.onerror = function(error) {
            statusDiv.textContent = 'Error - Reconnecting...';
            statusDiv.style.color = 'red';
        };
        
        function addEvent(type, data) {
            const eventElement = document.createElement('div');
            eventElement.className = 'event';
            eventElement.innerHTML = '<span class="event-type">' + type + ':</span> ' + data;
            eventsDiv.insertBefore(eventElement, eventsDiv.firstChild);
        }
    </script>
</body>
</html>`

func main() {
    o := okapi.Default()

    // Serve HTML page
    o.Get("/", func(c *okapi.Context) error {
        return c.HTMLView(http.StatusOK, template, okapi.M{
            "title":    "Okapi SSE Demo",
            "eventURL": "/events",
        })
    })

    // SSE endpoint
    o.Get("/events", func(c *okapi.Context) error {
        messageChan := make(chan okapi.Message, 10)
        ctx := c.Request().Context()

        go func() {
            defer close(messageChan)
            ticker := time.NewTicker(2 * time.Second)
            defer ticker.Stop()

            counter := 0
            for {
                select {
                case <-ctx.Done():
                    return
                case <-ticker.C:
                    counter++
                    
                    // Send different event types
                    eventType := "update"
                    if counter%5 == 0 {
                        eventType = "notification"
                    }
                    
                    messageChan <- okapi.Message{
                        Event: eventType,
                        Data: okapi.M{
                            "count":     counter,
                            "timestamp": time.Now().Format(time.RFC3339),
                            "message":   fmt.Sprintf("Event #%d", counter),
                        },
                    }
                }
            }
        }()

        opts := &okapi.StreamOptions{
            PingInterval: 30 * time.Second,
            OnError: func(err error) {
                fmt.Printf("Stream error: %v\n", err)
            },
        }

        return c.SSEStreamWithOptions(ctx, messageChan, opts)
    })

    if err := o.Start(); err != nil {
        panic(err)
    }
}
```

## Best Practices

### 1. Always Handle Context Cancellation

```go
o.Get("/events", func(c *okapi.Context) error {
    ctx := c.Request().Context()
    
    for {
        select {
        case <-ctx.Done():
            // Client disconnected
            return nil
        default:
            // Send event
            err := c.SSESendData("data")
            if err != nil {
                return err
            }
        }
    }
})
```

### 2. Use Buffered Channels

```go
messageChan := make(chan okapi.Message, 100) // Buffered channel
```

### 3. Set Appropriate Ping Intervals

```go
opts := &okapi.StreamOptions{
    PingInterval: 30 * time.Second, // Recommended: 15-60 seconds
}
```

### 4. Clean Up Resources

```go
go func() {
    defer close(messageChan)
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop() // Always clean up tickers
    
    // ... your code
}()
```

### 5. Handle Errors Gracefully

```go
opts := &okapi.StreamOptions{
    OnError: func(err error) {
        log.Printf("SSE stream error: %v", err)
        // Implement your error handling logic
    },
}
```

## API Reference

### Methods

- `SSESendData(data any)` - Send data with automatic serialization
- `SSESendJSON(data any)` - Send JSON data explicitly
- `SSESendText(text string)` - Send plain text
- `SSESendBinary(data []byte)` - Send binary data as base64
- `SSEvent(eventType string, data any)` - Send event with custom type
- `SSESendEvent(id, eventType string, data any)` - Send event with ID and type
- `SSEStream(ctx, messageChan)` - Stream from channel
- `SSEStreamWithOptions(ctx, messageChan, opts)` - Stream with advanced options
- `SendSSECustom(data, serializer)` - Send with custom serializer

### StreamOptions

```go
type StreamOptions struct {
    Serializer   Serializer    // Custom serializer
    PingInterval time.Duration // Keep-alive interval
    OnError      func(error)   // Error handler
}
```

## More Examples

See the complete examples repository: [okapi-example](https://github.com/jkaninda/okapi-example)
```
