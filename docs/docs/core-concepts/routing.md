---
title: Routing
sidebar_position: 1
---

# Routing

Okapi provides a simple and intuitive routing system that supports all standard HTTP methods and flexible path patterns.

## HTTP Methods

Okapi supports all standard HTTP methods:

```go
o.Get("/books", getBooks)
o.Post("/books", createBook)
o.Get("/books/:id", getBook)
o.Put("/books/:id", updateBook)
o.Delete("/books/:id", deleteBook)
o.Patch("/books/:id", patchBook)
o.Head("/books", headBooks)
o.Options("/books", optionsBooks)
```

`Any` registers one handler for every method:

```go
o.Any("/webhook", handleWebhook)
```

An `Any` route is documented under `get`, `post`, `put`, `patch` and `delete` in
the generated OpenAPI document. When CORS is enabled, a plain `OPTIONS` request
still reaches it — only requests carrying `Access-Control-Request-Method` are
treated as preflights.

## Path Syntax

Okapi supports flexible and expressive route path patterns, including named parameters and wildcards:

```go
o.Get("/books/{id}", getBook)        // Named path parameter using curly braces
o.Get("/books/{id:int}", getBook)    // Named path parameter, "id" documented as integer
o.Get("/books/:id", getBook)         // Named path parameter using colon prefix
o.Get("/*", getBook)                 // Catch-all wildcard (matches everything)
o.Get("/*any", getBook)              // Catch-all; the name is not used
o.Get("/*path", getBook)             // Catch-all; the name is not used either
```

Use whichever syntax feels most natural — Okapi normalizes both `{}` and `:` styles for named parameters and supports glob-style wildcards for flexible matching.

Named parameters are read with `c.Param("<name>")`:

```go
o.Get("/books/{id:int}", func(c *okapi.Context) error {
    return c.OK(okapi.M{"id": c.Param("id")})
})
```

A wildcard is different: whatever name you write after `*` is discarded when the
path is normalized — `/*`, `/*any` and `/*path` all become the same catch-all
pattern. The matched remainder is therefore always read with `c.Param("any")`:

```go
o.Get("/files/*path", func(c *okapi.Context) error {
    // GET /files/docs/readme.md -> "docs/readme.md"
    return c.OK(okapi.M{"path": c.Param("any")})
})
```

## Enabling and Disabling Routes

Okapi allows routes and route groups to be **enabled or disabled** without commenting out code.

### Features

| Type               | HTTP Response   | OpenAPI Document | Affects Child Routes |
|--------------------|-----------------|------------------|----------------------|
| **Disabled Route** | `404 Not Found` | Omitted          | N/A                  |
| **Disabled Group** | `404 Not Found` | Omitted          | Yes — all nested     |

The `404` is decided when the request is dispatched, so `.Disable()` and
`.Enable()` take effect immediately, and for a group they also cover routes that
were registered before the call.

The OpenAPI document is a different matter: it is built when the server starts,
and skips the routes that are disabled or hidden at that moment. Toggling a
route afterwards changes what the router answers, not what the published document
contains.

### Use Cases

* Temporarily removing endpoints during maintenance
* Controlling access based on feature flags
* Deprecating old API versions
* Creating toggleable test or staging routes

### Example

```go
func main() {
    app := okapi.Default()

    app.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{"version": "v1"})
    }).Disable() // returns 404 and is left out of the document

    // Deprecated route example
    app.Get("/deprecated", func(c *okapi.Context) error {
        return c.OK(okapi.M{"message": "This route is deprecated"})
    }).Deprecated() // mark route as deprecated in docs

    // Hidden route example
    app.Get("/hidden", func(c *okapi.Context) error {
        return c.OK(okapi.M{"message": "This route is hidden"})
    }).Hide() // hide route from docs, still reachable

    // Start the server
    if err := app.Start(); err != nil {
        panic(err)
    }
}
```

To re-enable any route or group, simply call the `.Enable()` method or remove the `.Disable()` call.

A route declared through `okapi.RouteDefinition` can start out disabled with the
`Disabled` field — see [Route Definition](../features/route-definition.md).
