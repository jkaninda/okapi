# Okapi

A modern, minimalist HTTP web framework for Go inspired by FastAPI's elegant design philosophy.

[![Tests](https://github.com/jkaninda/okapi/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/okapi/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/okapi)](https://goreportcard.com/report/github.com/jkaninda/okapi)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/okapi.svg)](https://pkg.go.dev/github.com/jkaninda/okapi)
[![codecov](https://codecov.io/gh/jkaninda/okapi/branch/main/graph/badge.svg?token=JHTW49M1LF)](https://codecov.io/gh/jkaninda/okapi)
[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/okapi)](https://github.com/jkaninda/okapi/releases)

<p align="center">
  <img src="https://raw.githubusercontent.com/jkaninda/okapi/main/logo.png" width="150" alt="Okapi logo">
</p>

Named after the okapi, a rare and graceful mammal native to the rainforests of northeastern Democratic Republic of the Congo—Okapi blends simplicity and strength in a unique, powerful package.

## Features

- **Intuitive API Design** – Clean, declarative syntax for routes and middleware
- **Automatic Request Binding** – Parse JSON, XML, forms, query params, headers, and path variables into structs
- **Built-in Validation** – Struct tag-based validation with comprehensive error messages
- **Auto-Generated OpenAPI Docs** – Swagger UI and ReDoc automatically synced with your code
- **Runtime Documentation Control** – Enable/disable OpenAPI docs at runtime without redeployment
- **Authentication Ready** – Native JWT, Basic Auth, and extensible middleware support
- **Standard Library Compatible** – Works seamlessly with Go's `net/http`
- **Dynamic Route Management** – Enable/disable routes at runtime without code changes
- **Production Ready** – CORS, TLS, graceful shutdown, and comprehensive middleware

## Installation

```bash
mkdir myapi && cd myapi
go mod init myapi
go get github.com/jkaninda/okapi@latest
```

## Quick Start

```go
package main

import "github.com/jkaninda/okapi"

func main() {
    o := okapi.Default()
    
    o.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{
            "message": "Hello from Okapi!",
        })
    })

	err := o.Start()
	if err != nil {
		panic(err) 
	}
}
```

Run with `go run main.go` and visit:
- **API**: http://localhost:8080
- **Docs**: http://localhost:8080/docs

---

## Request Binding & Validation

Okapi provides multiple approaches to bind and validate incoming requests, from simple binding to fully typed handler signatures.

### Validation Tags

Define validation rules directly on your structs:

```go
type Book struct {
    Name   string `json:"name" minLength:"4" maxLength:"50" required:"true" pattern:"^[A-Za-z]+$"`
    Price  int    `json:"price" required:"true" min:"5" max:"100"`
    Year   int    `json:"year" deprecated:"true"`
    Status string `json:"status" enum:"available,out_of_stock,discontinued" default:"available"`
}
```

### Method 1: Binding with `c.Bind()`

The simplest approach to bind and validate within your handler:

```go
o.Post("/books", func(c *okapi.Context) error {
    var book Book
    if err := c.Bind(&book); err != nil {
        return c.ErrorBadRequest(err)
    }
    return c.Created(book)
})
```

### Method 2: Typed Input with `okapi.Handle()`

Automatic input binding with a typed handler signature:

```go
o.Post("/books", okapi.Handle(func(c *okapi.Context, book *Book) error {
    book.ID = generateID()
    return c.Created(book)
}),
    okapi.DocRequestBody(&Book{}),
    okapi.DocResponse(&Book{}),
)
```

### Method 3: Shorthand with `okapi.H()`

A concise version for simple input validation:

```go
type BookDetailInput struct {
    ID int `path:"id"`
}

o.Get("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
    book := findBookByID(input.ID)
    if book == nil {
        return c.AbortNotFound("Book not found")
    }
    return c.OK(book)
}))
```

### Method 4: Input & Output with `okapi.HandleIO()`

Define both input and output structs separately for complex operations:

```go
type BookEditInput struct {
    ID   int  `path:"id" required:"true"`
    Body Book `json:"body"`
}

type BookOutput struct {
    Status int
    Body   Book
}

o.Put("/books/{id:int}", okapi.HandleIO(func(c *okapi.Context, input *BookEditInput) (*BookOutput, error) {
    book := updateBook(input.ID, input.Body)
    if book == nil {
        return nil, c.AbortNotFound("Book not found")
    }
    return &BookOutput{Body: *book}, nil
})).WithIO(&BookEditInput{}, &BookOutput{})
```

### Method 5: Output Only with `okapi.HandleO()`

When you only need a structured output without specific input validation:

```go
type BooksResponse struct {
    Body []Book `json:"books"`
}

o.Get("/books", okapi.HandleO(func(c *okapi.Context) (*BooksResponse, error) {
    return &BooksResponse{Body: getAllBooks()}, nil
})).WithOutput(&BooksResponse{})
```

---

## Advanced Request/Response Patterns

Separate payload from metadata using the `Body` field pattern:

```go
type BookRequest struct {
    Body   Book   `json:"body"`              // Request payload
    ID     int    `param:"id" query:"id"`    // Path or query parameter
    APIKey string `header:"X-API-Key" required:"true"` // Header
}

type BookResponse struct {
    Status    int    // HTTP status code
    Body      Book   // Response payload
    RequestID string `header:"X-Request-ID"` // Response header
}


o.Post("/books", func(c *okapi.Context) error {
    var req BookRequest
    if err := c.Bind(&req); err != nil {
        return c.ErrorBadRequest(err)
    }
    
    res := &BookResponse{
        Status:    201,
        RequestID: uuid.New().String(),
        Body:      req.Body,
    }
    return c.Respond(res) // Automatically sets status, headers, and body
},
    okapi.Request(&BookRequest{}),
    okapi.Response(BookResponse{}),
)
```

---

## Route Groups & Middleware

```go
api := o.Group("/api")

// Versioned API groups
v1 := api.Group("/v1", authMiddleware).Deprecated()
v2 := api.Group("/v2")

v1.Get("/books", getBooks)
v2.Get("/books", v2GetBooks)

// Disable routes at runtime
v2.Get("/experimental", experimentalHandler).Disable()

// Apply middleware to individual routes
v2.Get("/books/{id}", getBookByID).Use(cacheMiddleware)

// Protected admin routes
admin := api.Group("/admin", adminMiddleware)
admin.Get("/dashboard", getDashboard)
```

---

## Declarative Route Definition

Ideal for controller or service-based architectures:

```go
type BookService struct{}

func (s *BookService) Routes() []okapi.RouteDefinition {
    apiGroup := &okapi.Group{Prefix: "/api"}
    
    return []okapi.RouteDefinition{
        {
            Method:      http.MethodGet,
            Path:        "/books",
            Handler:     s.List,
            Group:       apiGroup,
            Summary:     "List all books",
            Response:    &BooksResponse{},
        },
        {
            Method:      http.MethodPost,
            Path:        "/books",
            Handler:     s.Create,
            Group:       apiGroup,
            Middlewares: []okapi.Middleware{authMiddleware},
            Security:    bearerAuthSecurity,
            Options: []okapi.RouteOption{
                okapi.DocSummary("Create a book"),
                okapi.DocRequestBody(&Book{}),
                okapi.DocResponse(&Book{}),
            },
        },
    }
}

// Register routes
app := okapi.Default()
bookService := &BookService{}
app.Register(bookService.Routes()...)
```

---

## Authentication

### JWT Authentication

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret:    []byte("your-secret-key"),
    ClaimsExpression: "Equals(`email_verified`, `true`)",
    TokenLookup:      "header:Authorization",
    ContextKey:       "user",
}

protected := o.Group("/api", jwtAuth.Middleware).WithBearerAuth()
protected.Get("/profile", getProfile)
```

### Basic Authentication

```go
basicAuth := okapi.BasicAuth{
    Username: "admin",
    Password: "secure-password",
}

admin := o.Group("/admin", basicAuth.Middleware)
admin.Get("/dashboard", getDashboard)
```

---

## Template Rendering

```go
func main() {
    tmpl, _ := okapi.NewTemplateFromDirectory("views", ".html")
    o := okapi.Default().WithRenderer(tmpl)
    
    o.Get("/", func(c *okapi.Context) error {
        return c.Render(http.StatusOK, "home", okapi.M{
            "title":   "Welcome",
            "message": "Hello, World!",
        })
    })
    
    o.Start()
}
```

### Embedded Templates

```go
//go:embed views/*
var Views embed.FS

func main() {
    app := okapi.New()
    app.WithRendererFromFS(Views, "views/*.html")
    app.StaticFS("/assets", http.FS(must(fs.Sub(Views, "views/assets"))))
    app.Start()
}
```

---

## Testing

```go
import "github.com/jkaninda/okapi/okapitest"

func TestGetBooks(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Get("/books", GetBooksHandler)
    
    okapitest.GET(t, server.BaseURL+"/books").
        ExpectStatusOK().
        ExpectBodyContains("Go Programming").
        ExpectHeader("Content-Type", "application/json")
}
```

---

## CLI Integration

```go
import "github.com/jkaninda/okapi/okapicli"

func main() {
    o := okapi.Default()
    
    cli := okapicli.New(o, "My API").
        String("config", "c", "config.yaml", "Config file").
        Int("port", "p", 8000, "Server port").
        Bool("debug", "d", false, "Debug mode")
    
    cli.Parse()
    o.WithPort(cli.GetInt("port"))
    
    // ... register routes ...
    
    cli.Run()
}
```

---

---

## OpenAPI Documentation

Okapi automatically generates interactive API documentation with multiple approaches to document your routes.

### Enabling Documentation

**With `okapi.Default()`** – Documentation is enabled by default at `/docs` and `/redoc`.

**With `okapi.New()`** – Documentation is disabled by default. Enable it conditionally:

```go
o := okapi.New()

if os.Getenv("ENABLE_DOCS") == "true" {
    o.WithOpenAPIDocs()
}
```

### Documenting Routes

#### Composable Functions

Simple and readable for most routes:

```go
o.Get("/books", getBooksHandler,
    okapi.DocSummary("List all available books"),
    okapi.DocTags("Books"),
    okapi.DocQueryParam("author", "string", "Filter by author name", false),
    okapi.DocQueryParam("limit", "int", "Maximum results to return", false),
    okapi.DocResponseHeader("X-Client-Id", "string", "Client ID"),
    okapi.DocResponse([]Book{}),
    okapi.DocResponse(400, ErrorResponse{}),
)
```

#### Fluent Builder

For complex or dynamic documentation needs:

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
        Build(),
)
```

#### Struct-Based with Body Field

Define request/response metadata directly in structs:

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

#### Fluent Route Methods

Chain documentation directly on route definitions:

```go
o.Post("/books", handler).WithIO(&BookRequest{}, &BookResponse{})  // Both request & response
o.Post("/books", handler).WithInput(&BookRequest{})                 // Request only
o.Get("/books", handler).WithOutput(&BooksResponse{})               // Response only
```

See the full guide at **[okapi.jkaninda.dev/features/openapi](https://okapi.jkaninda.dev/features/openapi.html)**

### Generated Documentation

|                               Swagger UI (`/docs`)                               |                             ReDoc (`/redoc`)                              |
|:--------------------------------------------------------------------------------:|:-------------------------------------------------------------------------:|
| ![Swagger UI](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png) | ![ReDoc](https://raw.githubusercontent.com/jkaninda/okapi/main/redoc.png) |

---

## Documentation

Full documentation available at **[okapi.jkaninda.dev](https://okapi.jkaninda.dev)**

Topics covered: Routing, Request Binding, Validation, Responses, Middleware, Authentication, OpenAPI, Testing, TLS, CORS, Graceful Shutdown, and more.

---

## Related Projects

Building microservices? 
Check out **[Goma Gateway](https://github.com/jkaninda/goma-gateway)** a high-performance API Gateway with authentication, rate limiting, load balancing, and support for REST, GraphQL, gRPC, TCP, and UDP.

## Okapi vs Huma

Both **[Okapi](https://github.com/jkaninda/okapi)** and **[Huma](https://github.com/danielgtaylor/huma)** aim to improve developer experience in Go APIs with strong typing and OpenAPI integration. The key difference is **philosophy**: Okapi is a *batteries-included web framework*, while Huma is an *API layer designed to sit on top of existing routers*.

| Feature / Aspect             | **Okapi**                                                              | **Huma**                                                       |
|------------------------------|------------------------------------------------------------------------|----------------------------------------------------------------|
| **Positioning**              | Full web framework                                                     | API framework built on top of existing routers                 |
| **Router**                   | Built-in high-performance router                                       | Uses external routers (Chi, httprouter, Fiber, etc.)           |
| **OpenAPI Generation**       | Native, framework-level (Swagger UI & Redoc included)                  | Native, schema-first API design                                |
| **Request Binding**          | Unified binder for JSON, XML, forms, query, headers, path params       | Struct tags + resolver pattern for headers, query, path params |
| **Validation**               | Tag-based (min, max, enum, required, default, pattern, etc.)           | Included                                                       |
| **Response Modeling**        | Output structs with `Body` pattern; headers & status via struct fields | Strongly typed response models with similar patterns           |
| **Middleware**               | Built-in + custom middleware, groups, per-route middleware             | Router middleware + Huma-specific middleware and transformers  |
| **Authentication**           | Built-in JWT, Basic Auth, security schemes for OpenAPI                 | Security schemes via OpenAPI; middleware via router            |
| **Dynamic Route Management** | Enable/disable routes & groups at runtime                              | Not a core feature                                             |
| **Templating / HTML**        | Built-in rendering (HTML templates, static files)                      | API-focused; not intended for HTML apps                        |
| **CLI Integration**          | Built-in CLI support (flags, env config)                               | Included                                                       |
| **Testing Utilities**        | Built-in test server and fluent HTTP assertions                        | Relies on standard Go testing tools                            |
| **Learning Curve**           | Very approachable for Go web developers                                | Slightly steeper (requires OpenAPI-first mental model)         |
| **Use Case Fit**             | Full web apps, APIs, gateways, microservices                           | Pure API services, schema-first API design                     |
| **Philosophy**               | "FastAPI-like DX for Go, batteries included"                           | "OpenAPI-first typed APIs on top of your router of choice"     |


### Quick Comparison

**Okapi** — define a route with built-in validation and OpenAPI metadata:

```go
app:=okapi.Default()
app.Register(okapi.RouteDefinition{
     Method:      http.MethodPost,
     Path:        "/users",
     Handler:     createUser,
     OperationId: "create-user",
     Summary:     "Create a new user", 
     Tags: []string{"users"},
     Request: &UserRequest{},
     Response:    &User{},
})
```

**Huma** — similar concept, different style:

```go
huma.Register(api, huma.Operation{
    OperationID: "create-user",
    Method:      http.MethodPost,
    Path:        "/users",
    Summary:     "Create a new user",
    Tags:        []string{"Users"},
}, createUser)
```

Both approaches generate OpenAPI documentation automatically.

---

### When to Choose Which?

#### Choose Okapi if you want:

- A **batteries-included web framework** with routing, middleware, auth, OpenAPI, templates, and CLI in one cohesive package
- **FastAPI-like developer experience** that feels idiomatic in Go
- **Dynamic route control** — enable or disable routes and groups at runtime
- To build APIs **and** serve HTML pages or static assets from the same application

#### Choose Huma if you want:

- A **schema-first, OpenAPI-driven API layer** where the spec drives your implementation
- To **keep using your existing router** (Chi, Fiber, Echo, etc.) without adopting a new framework
- **Strict typed request/response contracts** as your primary design model
- A **minimal, API-only stack** without broader web framework concerns

---

### Community & Maturity

- **Huma**: More established with a larger community and extensive production usage
- **Okapi**: Newer and rapidly evolving, with a smaller but growing community

Both are actively maintained. Choose based on your architectural preferences and project needs rather than stability concerns alone.

> **Note**: If you're already using Huma with Chi or another router and it's working well for you, there's no urgent reason to switch. Okapi is ideal for new projects or when you want a more integrated, batteries-included framework experience.


---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## Support

- **Documentation:** [okapi.jkaninda.dev](https://okapi.jkaninda.dev)
- **Issues:** [GitHub Issues](https://github.com/jkaninda/okapi/issues)
- **Discussions:** [GitHub Discussions](https://github.com/jkaninda/okapi/discussions)
- **LinkedIn:** [Jonas Kaninda](https://www.linkedin.com/in/jkaninda/)
---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**Made with ❤️ for the Go community**

⭐ **Star us on GitHub** — it motivates us to keep improving!

Copyright © 2025 Jonas Kaninda

</div>