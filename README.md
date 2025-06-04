# OKAPI - Lightweight Go Web Framework with OpenAPI 3.0 & Swagger UI

[![Tests](https://github.com/jkaninda/okapi/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/okapi/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/okapi)](https://goreportcard.com/report/github.com/jkaninda/okapi)
[![Go](https://img.shields.io/github/go-mod/go-version/jkaninda/okapi)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/okapi.svg)](https://pkg.go.dev/github.com/jkaninda/okapi)
[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/okapi)](https://github.com/jkaninda/okapi/releases)


**Okapi** is a modern, minimalist HTTP web framework for Go, inspired by **FastAPI**'s elegance. Designed for simplicity, performance, and developer happiness, it helps you build **fast, scalable, and well-documented APIs** with minimal boilerplate.

The framework is named after the okapi (/oʊˈkɑːpiː/), a rare and graceful mammal native to the rainforests of the northeastern Democratic Republic of the Congo. Just like its namesake — which resembles a blend of giraffe and zebra — Okapi blends simplicity and strength in a unique, powerful package.

<p align="center">
  <img src="https://raw.githubusercontent.com/jkaninda/okapi/main/logo.png" width="150" alt="Okapi logo">
</p>

---

### ✨ **Key Features**

✔ **Intuitive & Expressive API** – Clean, declarative syntax for effortless route and middleware definition.

✔ **Automatic Request Binding** – Seamlessly parse **JSON, XML, form data, query params, headers, and path variables** into structs.

✔ **Built-in Auth & Security** – Native support for **JWT, OAuth2, Basic Auth**, and custom middleware.

✔ **Blazing Fast Routing** – Optimized HTTP router with low overhead for high-performance applications.

✔ **First-Class Documentation** – **OpenAPI 3.0 & Swagger UI** integrated out of the box—auto-generate API docs with minimal effort.

✔ **Modern Tooling** –
- Route grouping & middleware chaining
- Static file serving
- Templating engine support
- CORS management
- Fine-grained timeout controls

✔ **Developer Experience** –
- Minimal boilerplate
- Clear error handling
- Structured logging
- Easy testing

Built for **speed, simplicity, and real-world use**—whether you're prototyping or running in production.

---

### Why Okapi?

✅ **Fast to learn** – If you know Go, you’re already halfway there.  
✅ **Flexible** – Adapts to your needs, not the other way around.  
✅ **Production-ready** – Robust enough for serious workloads.

Perfect for:
- **REST & JSON APIs**
- **Microservices**
- **Prototyping**
- **Educational projects**


---

## Installation

```bash
mkdir myapi && cd myapi
go mod init myapi
go get github.com/jkaninda/okapi
```

---

## Quick Start

Create a file named `main.go`:

```go
package main

import (
	"net/http"
	"github.com/jkaninda/okapi"
)

func main() {
	o := okapi.Default()

	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Welcome to Okapi!"})
	},
		okapi.DocSummary("Welcome page"),
	)

	if err := o.Start(); err != nil {
		panic(err)
	}
}
```

Run your server:

```bash
go run .
```

Visit [`http://localhost:8080`](http://localhost:8080) to see the response:

```json
{"message": "Welcome to Okapi!"}
```

Visit [`http://localhost:8080/docs/`](http://localhost:8080/docs/) to se the documentation

---

## Routing

Okapi supports all standard HTTP methods:

```go
o.Get("/books", getBooks)
o.Post("/books", createBook)
o.Get("/books/:id", getBook)
o.Put("/books/:id", updateBook)
o.Delete("/books/:id", deleteBook)
```

### Route Groups

Organize routes with nesting and middleware:

```go
api := o.Group("/api")

v1 := api.Group("/v1")
v2 := api.Group("/v2")

v1.Get("/users", getUsers)
v2.Get("/users", getUsers)


admin := api.Group("/admin", adminMiddleware)
admin.Get("/dashboard", getDashboard)
```

---

## Request Handling

### Path Parameters

```go
o.Get("/books/:id", func(c okapi.Context) error {
	id := c.Param("id")
	return c.String(http.StatusOK, id)
})
```

### Query Parameters

```go
o.Get("/books", func(c okapi.Context) error {
	name := c.Query("name")
	return c.String(http.StatusOK, name)
})
```

---

## Handling Form Data

### Multipart Form (`multipart/form-data`)

Handle standard form fields and file uploads:

```go
o.Post("/books", func(c okapi.Context) error {
	name := c.FormValue("name")
	price := c.FormValue("price")

	logo, err := c.FormFile("logo")
	if err != nil {
		return c.AbortWithError(http.StatusBadRequest, err)
	}
	file, err := logo.Open()
	if err != nil {
		return c.AbortWithError(http.StatusBadRequest, err)
	}
	defer file.Close()
	// You can now read or save the uploaded file
	return c.String(http.StatusOK, "File uploaded successfully")
})
```
---
## Struct Binding

Bind request data directly into a struct from multiple sources:

```go
type Book struct {
	ID    int    `json:"id" param:"id" query:"id" form:"id"`
	Name  string `json:"name" xml:"name" form:"name" min:"4" max:"50" required:"true"`
	Price int    `json:"price" form:"price" required:"true"`
	Logo *multipart.FileHeader `form:"logo" required:"true"`
    Content string `header:"Content-Type" json:"content-type" xml:"content-type" required:"true"`
	// Supports both ?tags=a&tags=b and ?tags=a,b
	Tags []string `form:"tags" query:"tags" default:"a,b"`
}

o.Post("/books", func(c okapi.Context) error {
	book := &Book{}
	if err := c.Bind(book); err != nil {
		return c.ErrorBadRequest(err)
	}
	return c.JSON(http.StatusOK, book)
})
```

### Supported Sources

* **Path parameters**: `param`
* **Query parameters**: `query`
* **Form fields**: `form`
* **JSON body**: `json`
* **XML body**: `xml`
* **Headers**: `header`

---

## Validation and Defaults

Okapi supports simple, declarative validation using struct tags.

### Semantics

| Field Type | Tag               | Meaning                |
|------------|-------------------|------------------------|
| `string`   | `min:"10"`        | Minimum length = 10    |
| `string`   | `max:"50"`        | Maximum length = 50    |
| `number`   | `min:"5"`         | Minimum value = 5      |
| `number`   | `max:"100"`       | Maximum value = 100    |
| `any`      | `default:"..."`   | Default value if empty |
| `any`      | `required:"true"` | Field must be provided |

---

## Middleware

### Built-in Example (Basic Auth)

```go
auth := okapi.BasicAuthMiddleware{
	Username: "admin",
	Password: "password",
	Realm:    "Restricted",
}

o.Use(auth.Middleware)
o.Get("/admin", adminHandler)
```

### CORS middleware

```go
cors := okapi.Cors{AllowedOrigins: []string{"http://localhost:8080", "https://example.com"}, AllowedHeaders: []string{}}
o := okapi.New(okapi.WithCors(cors))
	o.Get("/", func(c okapi.Context) error {
		return c.String(http.StatusOK, "Hello World!")
	})
```

### Custom Middleware

```go
func logger(next okapi.HandlerFunc) okapi.HandlerFunc {
	return func(c okapi.Context) error {
		start := time.Now()
		err := next(c)
		log.Printf("Request took %v", time.Since(start))
		return err
	}
}

o.Use(logger)
```

---

### OpenAPI/Swagger Integration

Okapi provides automatic OpenAPI (Swagger) documentation generation with built-in UI support. The documentation is dynamically generated from your route definitions.

#### Quick Start

To enable OpenAPI docs with default settings:

```go
o := okapi.Default()  // Docs available at /docs
```

#### Custom Configuration

Configure OpenAPI settings during initialization:

```go
o := okapi.New().WithOpenAPIDocs(
    okapi.OpenAPI{
        PathPrefix: "/swagger",  // Documentation path
        Title:     "My API",    // API title
        Version:   "1.0",       // API version
    }
)
```

### Documenting Routes

#### Example: Create Book Endpoint

```go
o.Post("/books", createBook,
    okapi.DocSummary("Create a new book"),
    okapi.DocTag("bookController"),
    okapi.DocBearerAuth(),  // Enable Bearer token authentication
    
    // RequestBody documentation
    okapi.RequestBody(BookRequest{}),
    
    // Response documentation
    okapi.DocResponse(BookResponse{}),
    
    // Header parameter
    okapi.DocHeader("Key", "1234", "API Key", true),
)
```

#### Example: Get Book Endpoint

```go
o.Get("/books/{id}", getBook,
    okapi.DocSummary("Get book by ID"),
    okapi.DocTag("bookController"),
    okapi.DocBearerAuth(),
    
    // Path parameter
    okapi.DocPathParam("id", "int", "Book ID"),
    
    // Query parameter
    okapi.DocQueryParam("country", "string", "Country filter", true),
    
    // Response documentation
    okapi.DocResponse(BookResponse{}),
)
```

### Available Documentation Options

| Method             | Description                         |
|--------------------|-------------------------------------|
| `DocSummary()`     | Short endpoint description          |
| `DocTag()`         | Groups related endpoints            |
| `DocTags()`        | Groups related endpoints            |
| `DocBearerAuth()`  | Enables Bearer token authentication |
| `DocRequestBody()` | Documents request body structure    |
| `DocResponse()`    | Documents response structure        |
| `DocPathParam()`   | Documents path parameters           |
| `DocQueryParam()`  | Documents query parameters          |
| `DocHeader()`      | Documents header parameters         |

### Swagger UI Preview

The automatically generated Swagger UI provides interactive documentation:

![Okapi Swagger Interface](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png)

---

## Templating

### Using a Custom Renderer

```go
o.Renderer = okapi.RendererFunc(func(w io.Writer, name string, data interface{}, c okapi.Context) error {
	tmpl, err := template.ParseFiles("templates/" + name + ".html")
	if err != nil {
		return err
	}
	return tmpl.ExecuteTemplate(w, name, data)
})
```

### Or Using a Struct-Based Renderer

```go
type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c okapi.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

tmpl := &Template{
	templates: template.Must(template.ParseGlob("templates/*.html")),
}

o.Renderer = tmpl
```

### Rendering a View

```go
o.Get("/", func(c okapi.Context) error {
	return c.Render(http.StatusOK, "welcome", okapi.M{
		"title":   "Welcome Page",
		"message": "Hello from Okapi!",
	})
})
```

---

## Static File Serving

Serve static assets and individual files:

```go
// Serve a single file
o.Get("/favicon.ico", func(c okapi.Context) error {
	c.ServeFile("public/favicon.ico")
	return nil
})

// Serve an entire directory
o.Static("/static", "public/assets")
```

## TLS Server

```go
// Initialize TLS configuration for secure HTTPS connections
    tls, err := okapi.LoadTLSConfig("public/cert.pem", "public/key.pem", "", false)
    if err != nil {
    panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
    }
    // Create a new Okapi instance with default config
    // With OpenAPI enabled, /docs
    o := okapi.Default()
    // Use HTTPS
    // o := okapi.New(okapi.WithTLS(tls))
    
    // Configure a secondary HTTPS server listening on port 8443
    // This creates both HTTP (8080) and HTTPS (8443) endpoints
    o.With(okapi.WithTLSServer(":8443", tls))
    
    // Register application routes and handlers
    o.Get("/", func(c okapi.Context) error {
    return c.JSON(http.StatusOK, okapi.M{
    "message": "Welcome to Okapi!",
    "status":  "operational",
    })
    })
    // Start the servers
    // This will launch both HTTP and HTTPS listeners in separate goroutines
    log.Println("Starting server on :8080 (HTTP) and :8443 (HTTPS)")
    if err := o.Start(); err != nil {
    panic(fmt.Sprintf("Server failed to start: %v", err))
    }
    // Start the server
    err = o.Start()
    if err != nil {
    panic(err)
    
    }
```

---

## Contributing

Contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to your fork
5. Open a Pull Request



---
## Give a Star! ⭐

⭐ If you find Okapi useful, please consider giving it a star on [GitHub](https://github.com/jkaninda/okapi)!


## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Copyright

Copyright (c) 2025 Jonas Kaninda
