---
title: Standard Library Compatibility
layout: default
parent: Features
nav_order: 12
---

# Standard Library Compatibility

Okapi speaks `net/http`. Existing handlers and middleware drop in unchanged, which lets you:

1. Reuse `func(http.Handler) http.Handler` middleware from the wider ecosystem
2. Register standard `http.HandlerFunc` and `http.Handler` handlers
3. Mix standard and Okapi-style routes in the same application

This makes Okapi suitable for gradual adoption in an existing Go project — you can move
one route at a time, and keep the parts that already work.

## Middleware Compatibility

`UseMiddleware` bridges standard middleware into Okapi's chain, so you can reuse
community-built logging, metrics, tracing, and compression packages.

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
    "github.com/jkaninda/okapi"
    "github.com/rs/cors"
)

o := okapi.Default()

c := cors.New(cors.Options{
    AllowedOrigins: []string{"*"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
})

o.UseMiddleware(c.Handler)
```

Standard middleware registered this way applies to **every** route — both standard
handlers and native Okapi ones.

## Handler Compatibility

Register any `http.HandlerFunc` with `HandleStd`, or any `http.Handler` with `HandleHTTP`.
Both keep Okapi's routing, middleware, and CORS registration.

### Signatures

```go
func (o *Okapi) HandleStd(method, path string, handler http.HandlerFunc, opts ...RouteOption)
func (o *Okapi) HandleHTTP(method, path string, handler http.Handler, opts ...RouteOption)
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

### Example: Mounting a Standard File Server

A catch-all segment (`{any...}`) matches the prefix and everything beneath it, which
is what you want when handing a subtree to an `http.Handler`:

```go
o.HandleHTTP("GET", "/assets/{any...}",
    http.StripPrefix("/assets/", http.FileServer(http.Dir("./public"))))
```

## Reading Path Parameters in a Standard Handler

Path parameters work exactly as they do under `http.ServeMux` — read them with
`r.PathValue`:

```go
o.HandleStd("GET", "/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")   // "42" for /users/42
    w.Write([]byte("user " + id))
})
```

Okapi's router captures parameters into the request context rather than through
`http.ServeMux`, so Okapi copies them onto the request with `SetPathValue` before
invoking a standard handler. This applies to `HandleStd` and `HandleHTTP`, on the
application and on groups alike.

`njia.Param(r, "id")` reads the same value directly from the router's context and
avoids the copy. It is the cheaper accessor if you are writing new code, but
`r.PathValue` is what an existing `net/http` handler already calls, which is why
it is supported:

```go
import "github.com/jkaninda/njia"

o.HandleStd("GET", "/users/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := njia.Param(r, "id")   // same value, no map allocation
    w.Write([]byte("user " + id))
})
```

Native Okapi handlers use `c.Param("id")`.

{: .note }
The bridge costs one map allocation per request, and is paid only by routes that
use a standard handler **and** declare path parameters. A parameterless route
skips it, and native handlers never touch it — measured at roughly +85 ns and
+336 B on such a route, with no measurable effect on native routes.

## Standard Handlers in Groups

`Group` exposes the same two methods, so standard handlers inherit the group's path
prefix and middleware chain:

```go
api := o.Group("/api")
api.Use(authMiddleware)

// Served at /api/legacy, behind authMiddleware
api.HandleStd("GET", "/legacy", legacyHandler)
api.HandleHTTP("GET", "/custom", &MyHandler{})
```

## OpenAPI Documentation

Standard handlers are registered as ordinary routes, so they **do** appear in the
generated OpenAPI document, and accept the same documentation options:

```go
o.HandleStd("GET", "/std-documented", myHandler,
    okapi.DocSummary("A standard handler"),
    okapi.DocTag("std"))
```

Because a standard handler has no typed input or output, Okapi cannot infer request
or response schemas for it — the operation is documented with its summary, tags, and
operation ID, but the body schemas are absent. If you want a fully described
operation, either declare the schemas explicitly with the documentation options or
convert the route to a native handler. Use `okapi.DocHide()` to keep an
undocumented route out of the spec entirely.

## Error Handling Differences

| Aspect               | `http.HandlerFunc`                       | `okapi.HandlerFunc`                    |
|----------------------|------------------------------------------|----------------------------------------|
| **Signature**        | `func(http.ResponseWriter, *http.Request)` | `func(*okapi.Context) error`         |
| **Response Writing** | Write to `w` directly                    | Return a helper such as `c.OK(...)`    |
| **Error Handling**   | Handled inline; Okapi cannot capture it  | Return the error; Okapi handles it     |
| **Status Codes**     | Set explicitly via `w.WriteHeader(code)` | Use helpers like `c.OK()`, `c.JSON()`  |
| **Content Type**     | Set manually via `w.Header().Set(...)`   | Set automatically based on the helper  |
| **Path Parameters**  | `r.PathValue("id")` or `njia.Param(r, "id")` | `c.Param("id")`                    |
| **Binding**          | Parse the request yourself               | `c.Bind(&v)` with validation           |

The trade-off in one line: standard handlers get the full middleware chain and
routing, but not `*okapi.Context` — so request binding, struct-tag validation, and
the error-returning signature are unavailable inside them.

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

## Best Practices

1. **Start with the standard library**: use `HandleStd` for existing code
2. **Migrate gradually**: convert one route at a time to Okapi style
3. **Use middleware**: leverage Okapi's middleware for cross-cutting concerns
4. **Consistent error handling**: adopt Okapi's error handling patterns for new code
5. **Document both styles**: keep documentation clear when mixing handler types

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

From a native handler you can always reach the raw request and response:

```go
o.Get("/raw", func(c *okapi.Context) error {
    w := c.Response()   // okapi.ResponseWriter
    ua := c.Request().Header.Get("User-Agent")

    // Use standard library directly
    w.Header().Set("X-Custom", "value")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("direct access from " + ua))

    return nil
})
```

`c.Response()` returns `okapi.ResponseWriter`, which embeds `http.ResponseWriter` and
adds `StatusCode()`, `BytesWritten()`, `Close()`, `Hijack()`, `Flush()`, and `Push()`.
Use `c.ResponseWriter()` when you need the plain `http.ResponseWriter` — for example
to pass it to a library that expects exactly that type.

## Runnable Example

A complete server combining everything on this page — standard middleware, both
handler registrations, a catch-all file server, group-scoped standard handlers and
a native handler alongside them — lives at
[`examples/std`](https://github.com/jkaninda/okapi/tree/main/examples/std):

```sh
go run ./examples/std
curl localhost:8080/users/42
```
