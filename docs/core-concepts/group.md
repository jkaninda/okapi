---
title: Route Groups
layout: default
parent: Core Concepts
nav_order: 4
---

# Route Groups

Route groups organize routes under a common path prefix, attach shared middleware, and expose group-level controls such as deprecation, disabling, OpenAPI tagging, and security requirements. Every group is bound to a single `*Okapi` instance and registers its routes through it.

## Features at a Glance

* **Path prefixing** — every route registered on a group is joined with its prefix
* **Nesting** — sub-groups inherit the parent's prefix, middlewares, and disabled state
* **Middleware** — chainable middleware applied before any route in the group (including nested ones)
* **Standard `net/http` interop** — register `http.Handler` / `http.HandlerFunc` and use `func(http.Handler) http.Handler` middleware
* **Disable / Enable** — flip a group on or off at runtime; disabled groups return `404` and are hidden from the OpenAPI spec
* **Deprecation** — mark every route in the group as deprecated in the docs
* **Tagging** — apply OpenAPI tags, plus rich tag info with descriptions and external docs
* **Security** — declare Bearer, Basic, or fully custom security requirements at the group level
* **Bulk registration** — register controller-style `[]RouteDefinition` in one call

## Creating a Group

There are two ways to create a group:

```go
o := okapi.Default()

// Most common: create from the Okapi instance
api := o.Group("/api", LoggerMiddleware)

// Alternative: create explicitly with NewGroup (useful when wiring controllers)
v1 := okapi.NewGroup("/v1", o, AuthMiddleware)
```

Both forms accept zero or more middlewares applied to every route in the group. The prefix must be non-empty.

`g.Okapi()` returns the parent `*Okapi` instance, which is handy when a controller receives a `*Group` and needs access to the underlying app.

## Nesting Subgroups

Calling `Group` on an existing group creates a nested subgroup. The child inherits its parent's prefix, middleware chain, and disabled state.

```go
o := okapi.Default()

api := o.Group("/api", LoggerMiddleware)

v1 := api.Group("/v1").Deprecated()                 // Marked as deprecated in OpenAPI
v2 := api.Group("/v2")                              // Active version
v3 := api.Group("/v3", featureFlagMW).Disable()     // Disabled, returns 404

v1.Get("/books", getBooks)
v2.Get("/books", v2GetBooks)
v3.Get("/books", v3GetBooks) // Not reachable while v3 is disabled

admin := api.Group("/admin", adminAuthMiddleware)
admin.Get("/dashboard", getDashboard)
```

## Registering Routes

A group exposes the same HTTP verbs as the top-level Okapi instance:

```go
api := o.Group("/api")

api.Get("/books", listBooks)
api.Post("/books", createBook)
api.Put("/books/:id", updateBook)
api.Patch("/books/:id", patchBook)
api.Delete("/books/:id", deleteBook)
api.Options("/books", optionsBooks)
api.Head("/books", headBooks)
```

Each method accepts the same `RouteOption` values as the top-level router (e.g. `DocSummary`, `DocResponse`, `UseMiddleware`).

### Standard `net/http` Handlers

For interop with the standard library, groups expose `HandleStd` and `HandleHTTP`. Both wrap the handler with the group's middleware chain.

```go
api := o.Group("/api")

// Standard http.HandlerFunc
api.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte("hello"))
})

// Standard http.Handler (e.g. a file server)
api.HandleHTTP("GET", "/assets/*", http.FileServer(http.Dir("static")))
```

## Middleware

### Adding Okapi Middleware

`Use` appends one or more middlewares to the group's chain. They run before any route-level middleware and are inherited by subgroups.

```go
api := o.Group("/api")

api.Use(func(c *okapi.Context) error {
    slog.Info("api request", "path", c.Request().URL.Path)
    return c.Next()
})
```

### Wrapping Standard HTTP Middleware

`UseMiddleware` adapts middleware written as `func(http.Handler) http.Handler` — the common pattern used by `gorilla/handlers`, `chi`, and similar libraries.

```go
api.UseMiddleware(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        slog.Info("standard middleware")
        next.ServeHTTP(w, r)
    })
})
```

## Enabling and Disabling Groups

Groups (and individual routes) can be toggled on or off at runtime without commenting out code.

| Type               | HTTP Response   | Swagger Docs | Affects Child Routes |
|--------------------|-----------------|--------------|----------------------|
| **Disabled Route** | `404 Not Found` | Hidden       | N/A                  |
| **Disabled Group** | `404 Not Found` | Hidden       | Yes — all nested     |

Typical uses:

* Temporarily removing endpoints during maintenance
* Gating routes behind feature flags
* Deprecating old API versions
* Building toggleable test or staging routes

```go
app := okapi.Default()
api := app.Group("/api")

v1 := api.Group("/v1").Disable() // Hidden from docs, returns 404 for all v1 routes
v1.Get("/", func(c *okapi.Context) error {
    return c.OK(okapi.M{"version": "v1"})
})

v2 := api.Group("/v2")
v2.Get("/", func(c *okapi.Context) error {
    return c.OK(okapi.M{"version": "v2"})
})

if err := app.Start(); err != nil {
    panic(err)
}
```

Call `.Enable()` to turn a group back on, or remove the `.Disable()` call entirely.

## Deprecating a Group

`Deprecated()` marks every route in the group as deprecated in the OpenAPI specification. Routes still work — clients are merely informed.

```go
v1 := api.Group("/v1").Deprecated()
v1.Get("/books", getBooks) // Documented as deprecated
```

## OpenAPI Tagging

Tags group operations in Swagger / Redoc UI. Okapi falls back to the group prefix when no tag is set.

### Simple Tags

```go
api := o.Group("/api").WithTags([]string{"api"})
```

### Rich Tag Info

`WithTagInfo` registers tag descriptions (and optional external documentation links) at the **root** of the OpenAPI spec, so Swagger UI renders them above the operations.

```go
api := o.Group("/api").WithTagInfo(
    okapi.GroupTag{
        Name:        "books",
        Description: "Operations on the books catalog",
        ExternalDocs: &okapi.ExternalDocs{
            URL:         "https://example.com/docs/books",
            Description: "Full books API reference",
        },
    },
    okapi.GroupTag{
        Name:        "shared",
        Description: "Endpoints shared across catalogs",
    },
)

api.Get("/books", listBooks) // Tagged: "books", "shared"
```

Empty tag names are silently ignored, and duplicate tag names are deduplicated across routes.

## Group-Level Security

Okapi exposes three helpers for declaring security requirements on every route in a group. They register the requirement in the OpenAPI spec; pair them with your authentication middleware to actually enforce auth.

### Bearer Authentication

```go
secure := o.Group("/secure").WithBearerAuth()
secure.Use(authMiddleware) // Your enforcement logic
secure.Get("/me", profile)
```

### Basic Authentication

```go
internal := o.Group("/internal").WithBasicAuth()
internal.Use(basicAuthMiddleware)
```

### Custom Security Requirements

`WithSecurity` accepts a raw OpenAPI security requirement object for fine-grained schemes (OAuth2, API keys, scopes, multiple schemes, etc.).

```go
admin := o.Group("/admin").WithSecurity([]map[string][]string{
    {"oauth2": {"admin:read", "admin:write"}},
})
```

## Bulk Registration with `Register`

`Register` accepts one or more `RouteDefinition` values, making it easy to define routes inside a controller and attach them to a group later.

```go
type BookController struct{}

func (c *BookController) Routes() []okapi.RouteDefinition {
    return []okapi.RouteDefinition{
        {
            Method:      http.MethodGet,
            Path:        "/books",
            OperationId: "ListBooks",
            Handler:     c.list,
            Options:     []okapi.RouteOption{okapi.DocSummary("List books")},
        },
        {
            Method:      http.MethodPost,
            Path:        "/books",
            Handler:     c.create,
            Middlewares: []okapi.Middleware{rateLimitMW},
            Options:     []okapi.RouteOption{okapi.DocSummary("Create a book")},
        },
    }
}

func main() {
    app := okapi.Default()
    api := app.Group("/api").WithTags([]string{"books"})

    bc := &BookController{}
    api.Register(bc.Routes()...) // All routes inherit /api + middleware + tags
}
```

Routes registered through `Register` inherit the group's prefix, tags, tag info, and disabled state.

## Method Chaining

Group configuration methods return `*Group`, so they can be chained fluently:

```go
v1 := o.
    Group("/v1", LoggerMiddleware).
    WithTags([]string{"v1"}).
    WithBearerAuth().
    Deprecated()
```
