---
title: Route Groups
sidebar_position: 4
---

# Route Groups

Route groups organize routes under a common path prefix, attach shared middleware, and expose group-level controls such as deprecation, disabling, OpenAPI tagging, and security requirements. Every group is bound to a single `*Okapi` instance and registers its routes through it.

## Features at a Glance

* **Path prefixing** — every route registered on a group is joined with its prefix
* **Nesting** — sub-groups inherit the parent's prefix, middlewares, and disabled state, including changes made to the parent after the sub-group was created
* **Middleware** — chainable middleware applied before any route in the group (including nested ones), resolved when a request is dispatched
* **Standard `net/http` interop** — register `http.Handler` / `http.HandlerFunc` and use `func(http.Handler) http.Handler` middleware
* **Disable / Enable** — flip a group on or off; disabled groups return `404` for every route in them and their sub-groups
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

Calling `Group` on an existing group creates a nested subgroup. The child resolves its parent's middleware chain and disabled state when a request arrives, so changes to the parent are picked up even after the child exists.

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
api.Any("/webhook", handleWebhook) // matches every method
```

Each method accepts the same `RouteOption` values as the top-level router (e.g. `DocSummary`, `DocResponse`, `UseMiddleware`).

Documentation-shaping settings — `Deprecated()`, `WithTags`, `WithTagInfo`,
`WithBearerAuth`, `WithBasicAuth` and `WithSecurity` — are copied onto each route
as it is registered, so set them on the group **before** registering its routes.
Middlewares and the disabled state are the exception: those are resolved per
request, so `Use` and `Disable` also affect routes registered earlier.

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

`Use` appends one or more middlewares to the group's chain. They run after the application's global middleware, before any route-level middleware, and are inherited by subgroups. Because the chain is built when a request is dispatched, `Use` also covers routes that were registered before the call.

```go
api := o.Group("/api")

api.Use(func(c *okapi.Context) error {
    slog.Info("api request", "path", c.Request().URL.Path)
    return c.Next()
})
```

### Wrapping Standard HTTP Middleware

`UseMiddleware` adapts middleware written as `func(http.Handler) http.Handler` — the common pattern used by `chi`, `rs/cors`, and similar libraries.

```go
api.UseMiddleware(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        slog.Info("standard middleware")
        next.ServeHTTP(w, r)
    })
})
```

The adapter forwards the request, so middleware that reads or replaces
`*http.Request` works. It cannot replace the `http.ResponseWriter`: the rest of
the chain keeps writing to Okapi's own writer, so a wrapper that the middleware
puts around `w` never sees the response. Compression, response recording and
metrics middleware written that way will not work here.

## Enabling and Disabling Groups

Groups (and individual routes) can be toggled on or off without commenting out code.

| Type               | HTTP Response   | OpenAPI Document | Affects Child Routes |
|--------------------|-----------------|------------------|----------------------|
| **Disabled Route** | `404 Not Found` | Omitted          | N/A                  |
| **Disabled Group** | `404 Not Found` | Omitted          | Yes — all nested     |

The `404` is decided per request, walking from the route through every parent
group, so `Disable()` and `Enable()` take effect immediately and cover routes
registered before the call. The OpenAPI document, by contrast, is built when the
server starts: it omits whatever is disabled at that point, and is not rebuilt
when you toggle a group later.

Typical uses:

* Temporarily removing endpoints during maintenance
* Gating routes behind feature flags
* Deprecating old API versions
* Building toggleable test or staging routes

```go
app := okapi.Default()
api := app.Group("/api")

v1 := api.Group("/v1").Disable() // Omitted from the document, returns 404 for all v1 routes
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

`Deprecated()` marks every route registered on the group afterwards as deprecated in the OpenAPI specification. Routes still work — clients are merely informed.

```go
v1 := api.Group("/v1").Deprecated()
v1.Get("/books", getBooks) // Documented as deprecated
```

## OpenAPI Tagging

Tags group operations in the documentation UI. Okapi falls back to the group prefix when no tag is set.

### Simple Tags

```go
api := o.Group("/api").WithTags([]string{"api"})
```

### Rich Tag Info

`WithTagInfo` registers tag descriptions (and optional external documentation links) at the **root** of the OpenAPI spec, so the UI renders them above the operations. Each name passed also becomes one of the group's tags.

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

`Register` goes through the same path as the group's verb methods, so routes
registered this way get the group's prefix, middlewares — **including
authentication** — tags, tag info, deprecation, security requirements and
disabled state, as well as the definition's own documentation fields
(`OperationId`, `Summary`, `Description`, `Tags`, `Request`, `Response`,
`Security`, `Disabled`).

A definition whose `Group` field is nil is attached to the group you call
`Register` on; one that names a `Group` is registered on that group instead. The
definitions you pass are copied, so your slice is not modified.

## Method Chaining

Group configuration methods return `*Group`, so they can be chained fluently:

```go
v1 := o.
    Group("/v1", LoggerMiddleware).
    WithTags([]string{"v1"}).
    WithBearerAuth().
    Deprecated()
```

`Use` and `UseMiddleware` are the exception: they return nothing, so call them on
their own line.
