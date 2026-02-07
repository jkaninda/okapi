# OKAPI - Modern Go Web Framework

[![Tests](https://github.com/jkaninda/okapi/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/okapi/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/okapi)](https://goreportcard.com/report/github.com/jkaninda/okapi)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/okapi.svg)](https://pkg.go.dev/github.com/jkaninda/okapi)
[![codecov](https://codecov.io/gh/jkaninda/okapi/branch/main/graph/badge.svg?token=JHTW49M1LF)](https://codecov.io/gh/jkaninda/okapi)
[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/okapi)](https://github.com/jkaninda/okapi/releases)

**Okapi** is a modern, minimalist HTTP web framework for Go inspired by **FastAPI**'s elegant design philosophy. Build fast, scalable, and well-documented APIs with minimal boilerplate while maintaining full control over your application.

<p align="center">
  <img src="https://raw.githubusercontent.com/jkaninda/okapi/main/logo.png" width="150" alt="Okapi logo">
</p>

Named after the okapi (/oʊˈkɑːpiː/), a rare and graceful mammal native to the rainforests of northeastern Democratic Republic of the Congo—just like its namesake, Okapi blends simplicity and strength in a unique, powerful package.

---

## ✨ Key Features

-  **Intuitive API Design** – Clean, declarative syntax for routes and middleware
-  **Automatic Request Binding** – Seamlessly parse JSON, XML, forms, query params, headers, and path variables into structs
-  **Built-in Security** – Native JWT, Basic Auth, and extensible custom middleware support
-  **Standard Library Compatible** – Works seamlessly with Go's `net/http` and existing codebases
-  **High-Performance Routing** – Optimized HTTP router with minimal overhead
-  **Auto-Generated OpenAPI Docs** – OpenAPI 3.0 & Swagger UI automatically synced with your code
- ️ **Dynamic Route Management** – Enable/disable routes or groups at runtime without code changes
-  **Production Ready** – CORS, templating, static files, TLS, graceful shutdown, and comprehensive middleware

**Perfect for:** REST APIs, microservices, rapid prototyping, and learning modern Go web development.

---

##  Why Choose Okapi?

| Feature                      | Benefit                                                    |
|------------------------------|------------------------------------------------------------|
| **Easy to Learn**            | Familiar Go idioms, productive in minutes                  |
| **Lightweight**              | Full control with minimal abstraction overhead             |
| **Production Battle-Tested** | Fast, reliable, and efficient under real-world load        |
| **Standard Library First**   | Zero friction with existing Go code                        |
| **Self-Documenting**         | OpenAPI specs always in sync with implementation           |
| **Dynamic Control**          | Toggle routes and groups at runtime—no code changes needed |

---

## Installation
```bash
go get github.com/jkaninda/okapi@latest
```


##  Quick Start
```go
package main

import "github.com/jkaninda/okapi"

func main() {
    o := okapi.Default()
    
    o.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{
            "message": "Hello from Okapi!",
            "license": "MIT",
        })
    })
    
    if err := o.Start(); err != nil {
        panic(err)
    }
}
```

**Run your app:**
```bash
go run main.go
```

**Access your API:**
- Application: http://localhost:8080
- API Documentation: http://localhost:8080/docs

---

##  Core Concepts

### Request Handling & Validation
```go
// Path parameters with type constraints
o.Get("/books/{id:int}", func(c *okapi.Context) error {
    id := c.Param("id")
    return c.JSON(200, okapi.M{"book_id": id})
})

// Struct binding with automatic validation
type Book struct {
    Name  string `json:"name" minLength:"5" maxLength:"50" required:"true"`
    Price int    `json:"price" min:"1" max:"100" required:"true"`
}

// Method 1: Using WithIO for cleaner syntax
o.Put("/books", func(c *okapi.Context) error {
    book := &Book{}
    if err := c.Bind(book); err != nil {
        return c.ErrorBadRequest(err)
    }
    return c.OK(book)
}).WithIO(&Book{}, &Book{})

// Method 2: Using RouteOptions for more control
o.Post("/books", func(c *okapi.Context) error {
    book := &Book{}
    if err := c.Bind(book); err != nil {
        return c.ErrorBadRequest(err)
    }
    return c.Created(book)
}, 
    okapi.DocSummary("Create a new book"),
    okapi.DocRequestBody(Book{}),
    okapi.DocResponse(Book{}),
)
```

### Advanced Request/Response Patterns

Separate your payload from metadata using the `Body` field pattern for cleaner, more maintainable code:
```go
type Book struct {
    Name   string `json:"name" minLength:"4" maxLength:"50" required:"true" pattern:"^[A-Za-z]+$"`
    Price  int    `json:"price" required:"true" min:"5" max:"100"`
    Year   int    `json:"year" deprecated:"true"`
    Status string `json:"status" enum:"available,out_of_stock,discontinued" default:"available"`
}

type BookRequest struct {
    Body   Book   `json:"body"`
    ID     int    `param:"id" query:"id"`
    APIKey string `header:"X-API-Key" required:"true"`
}

type BookResponse struct {
    Status    int    // HTTP status code
    Body      Book   // Response payload
    RequestID string `header:"X-Request-ID"` // Custom response header
}

func main() {
    o := okapi.Default()
    
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
        return c.Respond(res) // Automatically sets status, headers, and body based on struct tags
		// Alternative: return c.Return(res) to use the Status field as the HTTP status code
    },
        okapi.DocSummary("Create a new book"),
        okapi.Request(&BookRequest{}),
        okapi.Response(BookResponse{}),
    )
	// or using WithIO for cleaner syntax
	// o.Post("/books", createBookHandler).WithIO(&BookRequest{}, &BookResponse{})
    
    if err := o.Start(); err != nil {
        panic(err)
    }
}
```

### Route Groups & Middleware
```go
api := o.Group("/api")

// Version management with deprecation markers
v1 := api.Group("/v1", authMiddleware).Deprecated()
v2 := api.Group("/v2")
v3 := api.Group("/v3")

v1.Get("/books", getBooks)
v2.Get("/books", v2GetBooks)

// Dynamically disable specific routes
v3.Get("/books", v2GetBooks).Disable()

// Apply middleware to individual routes
v2.Get("/books/:id", v2GetBookByID).Use(customMiddleware)

// Protected admin routes
admin := api.Group("/admin", adminMiddleware)
admin.Get("/dashboard", getDashboard)
```

### Declarative Route Definition

For better organization, define routes using the `RouteDefinition` struct—ideal for controller-based architectures:
```go
type BookService struct{}

func (s *BookService) List(c *okapi.Context) error {
    return c.OK(okapi.M{"success": true, "message": "Books retrieved"})
}

func (s *BookService) Create(c *okapi.Context) error {
    return c.Created(okapi.M{"success": true, "message": "Book created"})
}

func (s *BookService) bookRoutes() []okapi.RouteDefinition {
    apiGroup := &okapi.Group{Prefix: "/api"}
    return []okapi.RouteDefinition{
        {
            Method:      http.MethodPut,
            Path:        "/books",
            Handler:     s.Update,
            Group:       apiGroup,
			OperationId: "updateBook", // OpenAPI operationId
            Summary:     "Update Book", // OpenAPI summary
            Description: "Update an existing book in the inventory", // OpenAPI description
            Request:     &BookRequest{}, // OpenAPI request body (if applicable)
            Response:    &BooksResponse{}, // OpenAPI success response (if applicable)
        },
        {
            Method:      http.MethodPost,
            Path:        "/books",
            Handler:     s.Create,
            Group:       apiGroup,
            Middlewares: []okapi.Middleware{customMiddleware},
            Security:    bearerAuthSecurity,
			// Using RouteOptions for more control over OpenAPI metadata
            Options: []okapi.RouteOption{
                okapi.DocSummary("Create Book"),
                okapi.DocDescription("Add a new book to the inventory"),
                okapi.DocRequestBody(&Book{}),
                okapi.DocResponse(&Book{}),
                okapi.DocResponse(http.StatusUnauthorized, AuthError{}),
            },
        },
    }
}
```

**Register routes:**
```go
app := okapi.Default()
bookService := &BookService{}

// Method 1: Direct registration
app.Register(bookService.bookRoutes()...)

// Method 2: Using helper function
okapi.RegisterRoutes(app, bookService.Routes())
```

See the complete example in [examples/route-definition](https://github.com/jkaninda/okapi/tree/main/examples/route-definition).

---

###  Authentication & Security
```go
// JWT Authentication
jwtAuth := okapi.JWTAuth{
    SigningSecret: []byte("your-secret-key"),
    TokenLookup:   "header:Authorization",
}

protected := o.Group("/api", jwtAuth.Middleware).WithBearerAuth()
protected.Get("/profile", getProfile)
protected.Post("/logout", logout)

// Basic Authentication
basicAuth := okapi.BasicAuth{
    Username: "admin",
    Password: "secure-password",
}
admin := o.Group("/admin", basicAuth.Middleware)
admin.Get("/dashboard", getDashboard)
```

---

## Testing

Built-in testing utilities for comprehensive test coverage:
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

##  CLI Integration

Build production-ready command-line applications:
```go
import "github.com/jkaninda/okapi/okapicli"

func main() {
    o := okapi.Default()
    
    cli := okapicli.New(o, "Okapi CLI Application").
        String("config", "c", "config.yaml", "Configuration file path").
        Int("port", "p", 8000, "HTTP server port").
        Bool("debug", "d", false, "Enable debug mode")
    
    if err := cli.Parse(); err != nil {
        panic(err)
    }
    
    // Apply CLI options
    o.WithPort(cli.GetInt("port"))
    if cli.GetBool("debug") {
        o.WithDebug()
    }
    
    o.Get("/", func(ctx *okapi.Context) error {
        return ctx.OK(okapi.M{"message": "Hello, Okapi!"})
    })
    
    if err := cli.Run(); err != nil {
        panic(err)
    }
}
```

---

## Documentation

**Complete documentation:** [okapi.jkaninda.dev](https://okapi.jkaninda.dev)

### Topics Covered:
- **Routing** – Path patterns, groups, dynamic management
- **Request Binding** – JSON, XML, forms, validation
- **Responses** – JSON, XML, templates, file serving
- **Middleware** – Built-in and custom middleware
- **Authentication** – JWT, Basic Auth, OAuth2
- **OpenAPI/Swagger** – Auto-generated documentation
- **Testing** – Comprehensive testing utilities
- **Advanced Features** – TLS, CORS, graceful shutdown, CLI integration

---

## Related Projects

Building microservices? Check out **[Goma Gateway](https://github.com/jkaninda/goma-gateway)** – a high-performance API Gateway featuring:
- Authentication & authorization
- HTTP caching & rate limiting
- Load balancing
- Support for REST, GraphQL, gRPC, TCP, and UDP

---

## Contributing

We welcome contributions! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please read our [Contributing Guide](CONTRIBUTING.md) for detailed guidelines.

---

## ⭐ Star History

[![Star History Chart](https://api.star-history.com/svg?repos=jkaninda/okapi&type=Date)](https://star-history.com/#jkaninda/okapi&Date)

---

##  Support & Community

-  **Documentation:** [okapi.jkaninda.dev](https://okapi.jkaninda.dev)
-  **Bug Reports:** [GitHub Issues](https://github.com/jkaninda/okapi/issues)
-  **Discussions:** [GitHub Discussions](https://github.com/jkaninda/okapi/discussions)
-  **LinkedIn:** [Jonas Kaninda](https://www.linkedin.com/in/jkaninda/)

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**Made with ❤️ for the Go community**

**⭐ Star us on GitHub — it motivates us to keep improving!**

Copyright © 2025 Jonas Kaninda

</div>