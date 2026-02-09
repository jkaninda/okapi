---
title: Standard Library Compatibility
layout: default
parent: Features
nav_order: 10
---

# Standard Library Compatibility

Okapi integrates seamlessly with Go's `net/http` standard library, enabling you to:

1. Use existing `http.Handler` middleware
2. Register standard `http.HandlerFunc` handlers
3. Combine Okapi-style routes with standard library handlers

This makes Okapi ideal for gradual adoption or hybrid use in existing Go projects.

## Middleware Compatibility

Okapi's `UseMiddleware` bridges standard `http.Handler` middleware into Okapi's middleware system. This lets you reuse the wide ecosystem of community-built middleware—such as logging, metrics, tracing, compression, and more.

### Signature

```go
func (o *Okapi) UseMiddleware(middleware func(http.Handler) http.Handler)
```

### Example: Injecting a Custom Header

```go
o := okapi.Default()

// Add a custom version header to all responses
o.UseMiddleware(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Version", "v1.2.0")
        next.ServeHTTP(w, r)
    })
})
```

### Example: Using Third-Party Middleware

```go
import (
    "github.com/gorilla/handlers"
    "github.com/jkaninda/okapi"
)

o := okapi.Default()

// Use Gorilla's CORS middleware
o.UseMiddleware(handlers.CORS(
    handlers.AllowedOrigins([]string{"*"}),
    handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
))
```

## Handler Compatibility

You can register any `http.HandlerFunc` using `HandleStd`, or use full `http.Handler` instances via `HandleHTTP`. These retain Okapi's routing and middleware features while supporting familiar handler signatures.

### HandleStd Signature

```go
func (o *Okapi) HandleStd(method, path string, handler http.HandlerFunc, opts ...RouteOption)
```

### Example: Basic Standard Library Handler

```go
o := okapi.Default()

o.HandleStd("GET", "/greeting", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello from Okapi!"))
})
```

### Example: Using http.Handler

```go
type MyHandler struct{}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello from custom handler!"))
}

o.HandleHTTP("GET", "/custom", &MyHandler{})
```

## Migration Tips

Migrating an existing `net/http` application? Okapi makes it painless.

### Mixed Routing Support

You can mix Okapi and standard handlers in the same application:

```go
// Okapi-style route
o.Handle("GET", "/okapi", func(c *okapi.Context) error {
    return c.OK(okapi.M{"status": "ok"})
})

// Standard library handler
o.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("standard response"))
})
```

### Gradual Migration

Start by wrapping your existing routes:

```go
// Existing handler
func oldHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("legacy code"))
}

// Register with Okapi
o.HandleStd("GET", "/legacy", oldHandler)

// New Okapi-style handler
o.Get("/new", func(c *okapi.Context) error {
    return c.OK(okapi.M{"message": "new code"})
})
```

### Converting Handlers

Convert standard handlers to Okapi style when ready:

```go
// Before (standard library)
func handler(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(map[string]string{"message": "hello"})
}

// After (Okapi)
func handler(c *okapi.Context) error {
    return c.JSON(200, okapi.M{"message": "hello"})
}
```

## Error Handling Differences

| Aspect                 | http.HandlerFunc                          | okapi.HandlerFunc                      |
|------------------------|-------------------------------------------|----------------------------------------|
| **Response Writing**   | Must manually call `w.WriteHeader(...)`   | Return an error or use helper methods  |
| **Error Handling**     | Handle errors inline within the handler   | Return errors; Okapi handles them      |
| **Status Codes**       | Set explicitly via `w.WriteHeader(code)`  | Use helpers like `c.OK()`, `c.JSON()`  |
| **Content Type**       | Set manually via `w.Header().Set(...)`    | Automatically set based on method used |

## Best Practices

1. **Start with Standard Library**: Use `HandleStd` for existing code
2. **Migrate Gradually**: Convert one route at a time to Okapi style
3. **Use Middleware**: Leverage Okapi's middleware for cross-cutting concerns
4. **Consistent Error Handling**: Adopt Okapi's error handling patterns for new code
5. **Document Both Styles**: Keep documentation clear when mixing handler types

## Example: Complete Migration

### Before (Pure net/http)

```go
func main() {
    http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        
        users := []string{"Alice", "Bob"}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(users)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### After (Okapi)

```go
func main() {
    o := okapi.Default()
    
    o.Get("/users", func(c *okapi.Context) error {
        users := []string{"Alice", "Bob"}
        return c.JSON(http.StatusOK, users)
    })
    
    o.Start()
}
```

## Accessing Underlying Objects

When needed, you can access the underlying `http.Request` and `http.ResponseWriter`:

```go
o.Get("/raw", func(c *okapi.Context) error {
    req := c.Request()   // *http.Request
    w := c.Response()    // http.ResponseWriter
    
    // Use standard library directly
    w.Header().Set("X-Custom", "value")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("direct access"))
    
    return nil
})
```


