---
title: OpenAPI & Swagger
layout: default
parent: Features
nav_order: 2
---

# OpenAPI & Swagger

Okapi provides **automatic OpenAPI (Swagger) documentation generation** with a built-in interactive UI. The documentation is dynamically generated from your route definitions, ensuring it always stays in sync with your API implementation.

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

## Accessing Documentation

* **Swagger UI**: `http://localhost:8080/docs`
* **Redoc**: `http://localhost:8080/redoc`
* **OpenAPI JSON**: `http://localhost:8080/openapi.json`

![Swagger UI](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png)

![Redoc](https://raw.githubusercontent.com/jkaninda/okapi/main/redoc.png)
