---
title: OpenAPI & Swagger
layout: default
parent: Features
nav_order: 3
---

# OpenAPI & Swagger

Okapi provides **automatic OpenAPI documentation generation** with built-in interactive UIs. The documentation is dynamically generated from your route definitions, ensuring it always stays in sync with your API implementation.

Okapi serves both **OpenAPI 3.1** (the default) and **OpenAPI 3.0**, and ships with three interactive UIs out of the box — **Swagger UI** (default), **ReDoc**, and **Scalar** — with the UI rendered at `/docs` fully selectable.

## Quick Start

### Using `okapi.Default()`

Documentation is enabled by default and served at `/docs`:

```go
o := okapi.Default() // Docs available at /docs
```

### Using `okapi.New()` with `WithOpenAPIDocs()`

If you initialize Okapi with `okapi.New()`, documentation is disabled by default. Enable it with `WithOpenAPIDocs()`:

```go
o := okapi.New()

if os.Getenv("ENABLE_DOCS") == "true" {
    o.WithOpenAPIDocs() 
}
```

## Custom Configuration

Customize the OpenAPI documentation:

```go
o := okapi.New().WithOpenAPIDocs(
    okapi.OpenAPI{
        Title:      "Example API",
        Version:    "1.0.0",
        Contact: okapi.Contact{
            Name:  "API Support",
            Email: "support@example.com",
        },
    },
)
```

## Security Schemes

Define authentication mechanisms for your API:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:   "Okapi Web Framework Example",
    Version: "1.0.0",
    License: okapi.License{Name: "MIT"},
    SecuritySchemes: okapi.SecuritySchemes{
        {
            Name:   "basicAuth",
            Type:   "http",
            Scheme: "basic",
        },
        {
            Name:         "bearerAuth",
            Type:         "http",
            Scheme:       "bearer",
            BearerFormat: "JWT",
        },
        {
            Name: "OAuth2",
            Type: "oauth2",
            Flows: &okapi.OAuthFlows{
                AuthorizationCode: &okapi.OAuthFlow{
                    AuthorizationURL: "https://auth.example.com/authorize",
                    TokenURL:         "https://auth.example.com/token",
                    Scopes: map[string]string{
                        "read":  "Read access",
                        "write": "Write access",
                    },
                },
            },
        },
    },
})
```

## Applying Security to Routes

### Single Route

```go
var bearerAuthSecurity = []map[string][]string{
    {"bearerAuth": {}},
}

o.Get("/books", getBooksHandler).WithSecurity(bearerAuthSecurity...)
```

### Route Group

```go
api := o.Group("/api", jwtMiddleware).WithSecurity(bearerAuthSecurity)
api.Get("/", apiHandler)
```

## Documenting Routes

Okapi offers multiple ways to document your routes.

### 1. Composable Functions (Direct Style)

Simple and readable approach for small to medium routes:

```go
o.Get("/books", getBooksHandler,
    okapi.DocSummary("List all available books"),
    okapi.DocTags("Books"),
    okapi.DocQueryParam("author", "string", "Filter by author name", false),
    okapi.DocQueryParam("limit", "int", "Maximum results to return", false),
    okapi.DocResponseHeader("X-Client-Id", "string", "Client ID"),
    okapi.DocResponse([]Book{}),
    okapi.DocResponse(400, ErrorResponse{}),
    okapi.DocResponse(401, ErrorResponse{}),
)
```

### 2. Fluent Builder Style

For complex or dynamic documentation:

```go
o.Post("/books", createBookHandler,
    okapi.Doc().
        Summary("Add a new book to the inventory").
        Tags("Books").
        BearerAuth().
        ResponseHeader("X-Client-Id", "string", "Client ID").
        RequestBody(BookRequest{}).
        Response(201, Book{}).
        Response(400, ErrorResponse{}).
        Response(401, ErrorResponse{}).
        Build(),
)
```

### 3. Body Field Style

Using structs with dedicated body fields:

```go
type BookRequest struct {
    Body struct {
        Name  string `json:"name" minLength:"4" maxLength:"50" required:"true"`
        Price int    `json:"price" required:"true"`
    } `json:"body"`

    ID     int    `param:"id" query:"id"`
    APIKey string `header:"X-API-Key" required:"true"`
}

o.Post("/books", createBookHandler,
    okapi.Request(&BookRequest{}),
    okapi.Response(&BookResponse{}),
)
```

### Using `.WithIO()`, `.WithInput()`, `.WithOutput()`

```go
// Both request & response
o.Post("/books", handler).WithIO(&BookRequest{}, &BookResponse{})

// Request only
o.Post("/books", handler).WithInput(&BookRequest{})

// Response only
o.Get("/books", handler).WithOutput(&BooksResponse{})
```

### 4. Declarative Route Definition

With `okapi.RouteDefinition`, documentation lives right next to the route. Common
fields — `Summary`, `Description`, `Tags`, `Request`, `Response`, and `Security` —
are set directly on the struct:

```go
routes := []okapi.RouteDefinition{
    {
        Method:      http.MethodPost,
        Path:        "/books",
        Handler:     createBookHandler,
        Summary:     "Add a new book",
        Description: "Create a new book in the inventory",
        Tags:        []string{"Books"},
        Request:     &BookRequest{},      // request body + params, validated and documented
        Response:    &Book{},             // 200 response schema
        Security: []map[string][]string{  // requires bearer auth
            {"bearerAuth": {}},
        },
    },
}

app := okapi.New()
okapi.RegisterRoutes(app, routes)
```

For anything the struct fields don't cover (extra status codes, headers, query
params, …), drop down to the `Options` field with the same `Doc*` helpers used
elsewhere. Struct fields and `Options` can be mixed freely:

```go
routes := []okapi.RouteDefinition{
    {
        Method:  http.MethodGet,
        Path:    "/books/{id:int}",
        Handler: getBookHandler,
        Tags:    []string{"Books"},
        Options: []okapi.RouteOption{
            okapi.DocSummary("Get a book by ID"),
            okapi.DocPathParam("id", "int", "The ID of the book"),
            okapi.DocResponse(Book{}),                       // 200
            okapi.DocResponse(404, ErrorResponse{}),         // 404
            okapi.DocResponseHeader("X-Request-Id", "string", "Request ID"),
        },
    },
}
```

Attach routes to a group to share a prefix, tags, middleware, and security across
several definitions:

```go
books := &okapi.Group{Prefix: "/api/v1", Tags: []string{"Books"}}

routes := []okapi.RouteDefinition{
    {
        Method:   http.MethodGet,
        Path:     "/books",
        Handler:  listBooksHandler,
        Group:    books,
        Summary:  "List all books",
        Response: &BooksResponse{},
    },
    {
        Method:   http.MethodPost,
        Path:     "/books",
        Handler:  createBookHandler,
        Group:    books,
        Summary:  "Add a new book",
        Request:  &BookRequest{},
        Response: &Book{},
    },
}

app.Register(routes...) // or okapi.RegisterRoutes(app, routes)
```

## Available Documentation Options

| Method                                           | Description                              |
|--------------------------------------------------|------------------------------------------|
| `DocSummary()` / `Doc().Summary()`               | Short endpoint summary                   |
| `DocTags()` / `Doc().Tags()`                     | Group endpoints under tags               |
| `DocBearerAuth()` / `Doc().BearerAuth()`         | Enable Bearer token authentication       |
| `DocRequestBody()` / `Doc().RequestBody()`       | Document request body schema             |
| `DocResponse()` / `Doc().Response()`             | Document response schema or status codes |
| `DocPathParam()` / `Doc().PathParam()`           | Document path parameters                 |
| `DocQueryParam()` / `Doc().QueryParam()`         | Document query parameters                |
| `DocHeader()` / `Doc().Header()`                 | Document request headers                 |
| `DocResponseHeader()` / `Doc().ResponseHeader()` | Document response headers                |
| `DocDeprecated()` / `Doc().Deprecated()`         | Mark route as deprecated                 |

## Choosing the Documentation UI

Okapi ships with three interactive UIs: **Swagger UI** (default), **ReDoc**, and **Scalar**.
The UI rendered at `/docs` is selectable. With `okapi.Default()`, each UI also stays reachable at
its own dedicated route (`/swagger`, `/redoc`, `/scalar`); with `okapi.New()`, only `/docs` is served
(see [Restricting to a single UI](#restricting-to-a-single-ui)).

Select it with the `UI` field on `okapi.OpenAPI`:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title: "My API",
    UI:    okapi.ScalarUI, // okapi.SwaggerUI (default) | okapi.RedocUI | okapi.ScalarUI
})
```

…or with the chainable `WithDocUI` method:

```go
o := okapi.New().WithOpenAPIDocs().WithDocUI(okapi.ScalarUI)
```

### Restricting to a single UI

Whether the non-selected UIs stay reachable depends on how you create the instance:

* `okapi.Default()` — all UI routes (`/swagger`, `/redoc`, `/scalar`) stay reachable
  alongside `/docs` (`StrictDocUI` is **disabled**).
* `okapi.New()` — only the selected UI is served at `/docs`; the other routes return
  `404` (`StrictDocUI` is **enabled**).

Override this explicitly via the `StrictDocUI` field. Set it to `true` to register only the
selected UI — the other UI routes then return `404`:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:       "My API",
    UI:          okapi.ScalarUI,
    StrictDocUI: true, // only /docs and /scalar are served; /swagger and /redoc return 404
})
```

…or set it to `false` to keep every UI reachable regardless of which one `/docs` renders:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:       "My API",
    UI:          okapi.ScalarUI,
    StrictDocUI: false, // /swagger, /redoc and /scalar all stay reachable
})
```

## OpenAPI 3.1 and 3.0

Okapi serves the same API description as both **OpenAPI 3.1 / JSON Schema 2020-12** and **OpenAPI 3.0**.
The default endpoints (`/openapi.json`, `/openapi.yaml`) serve **3.1**, and the documentation UIs render it.
The 3.0 document remains available at `/openapi-3.0.{json,yaml}`, so 3.0-only consumers stay supported.

The 3.1 document is derived from the 3.0 base and adds these 3.1 features:

- **Type-array nullability** — pointer fields render as `nullable: true` in 3.0 and as
  `type: ["string", "null"]` in 3.1.
- **`jsonSchemaDialect`** — set to the JSON Schema 2020-12 base dialect on the 3.1 document.
- **SPDX license identifier** — set `License.Identifier` (e.g. `"Apache-2.0"`); it appears only on
  the 3.1 document and is mutually exclusive with `License.URL`.
- **`const`** — the `const:"value"` struct tag becomes a JSON Schema `const` on the 3.1 document.
- **Webhooks** — declare outbound callbacks with `o.Webhook(...)`; they appear under the `webhooks`
  field of the 3.1 document only.

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:   "Example API",
    Version: "1.0.0",
    License: okapi.License{Name: "Apache 2.0", Identifier: "Apache-2.0"},
})

// A webhook is documentation-only: it is not added to the router.
o.Webhook("newBook", http.MethodPost,
    okapi.DocSummary("Notifies subscribers about a newly added book"),
    okapi.DocRequestBody(Book{}),
    okapi.DocResponse(200, okapi.M{"received": true}),
)
```

## Accessing Documentation

| Route               | Content                                          |
|---------------------|--------------------------------------------------|
| `/docs`             | The selected UI (Swagger UI by default)          |
| `/swagger`          | Swagger UI                                        |
| `/redoc`            | ReDoc                                             |
| `/scalar`           | Scalar API Reference                             |
| `/openapi.json`     | OpenAPI spec (JSON) — **3.1 by default**          |
| `/openapi.yaml`     | OpenAPI spec (YAML) — **3.1 by default**          |
| `/openapi-3.0.json` | OpenAPI **3.0** spec (JSON)                       |
| `/openapi-3.0.yaml` | OpenAPI **3.0** spec (YAML)                       |

![Swagger UI](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png)

![Redoc](https://raw.githubusercontent.com/jkaninda/okapi/main/redoc.png)
![Scalar](https://raw.githubusercontent.com/jkaninda/okapi/main/scalar.png)
