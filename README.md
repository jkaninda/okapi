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

✔ Dynamic Route Management – Easily enable or disable individual routes or groups, with automatic Swagger sync and no code commenting.

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

###  Why Choose Okapi?

* **Easy to Learn** – Familiar Go syntax and intuitive APIs mean you’re productive in minutes.
* **Highly Flexible** – Designed to adapt to your architecture and workflow—not the other way around.
* **Built for Production** – Lightweight, fast, and reliable under real-world load.
* **Standard Library Compatibility** - Integrates seamlessly with Go’s net/http standard library, making it easy to combine Okapi with existing Go code and tools.
* **Automatic OpenAPI Documentation** - Generate comprehensive, first-class OpenAPI specs for every route—effortlessly keep your docs in sync with your code.
* **Dynamic Route Management** - Instantly enable or disable routes and route groups at runtime, offering a clean, efficient alternative to commenting out code when managing your API endpoints.

Ideal for:

*  **High-performance REST APIs**
*  **Composable microservices**
*  **Rapid prototyping**
*  **Learning & teaching Go web development**

Whether you're building your next startup, internal tools, or side projects—**Okapi scales with you.**


---

## Installation

```bash
mkdir myapi && cd myapi
go mod init myapi
```

```sh
go get github.com/jkaninda/okapi@latest
```

---

## Quick Start

Create a file named `main.go`:

### Example

#### Hello

```go
package main

import (
  "github.com/jkaninda/okapi"
)
func main() {

	o := okapi.Default()
	
	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Hello from Okapi Web Framework!","Licence":"MIT"})
	})
	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
```
####  Simple HTTP POST
```go
package main

import (
  "github.com/jkaninda/okapi"
  "net/http"
)

type Response struct {
  Success bool   `json:"success"`
  Message string `json:"message"`
  Data    Book   `json:"data"`
}
type Book struct {
  Name  string `json:"name" form:"name"  max:"50" required:"true" description:"Book name"`
  Price int    `json:"price" form:"price" query:"price" yaml:"price" required:"true" description:"Book price"`
}
type ErrorResponse struct {
  Success bool        `json:"success"`
  Status  int         `json:"status"`
  Details any `json:"details"`
}

func main() {
  // Create a new Okapi instance with default config
  o := okapi.Default()

  o.Post("/books", func(c okapi.Context) error {
    book := Book{}
    err := c.Bind(&book)
    if err != nil {
      return c.ErrorBadRequest(ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
    }
    response := Response{
      Success: true,
      Message: "This is a simple HTTP POST",
      Data:    book,
    }
    return c.OK(response)
  },
    // OpenAPI Documentation
    okapi.DocSummary("Create a new Book"),
    okapi.DocRequestBody(Book{}),                                   //  Request body
    okapi.DocResponse(Response{}),                                  // Success Response body
    okapi.DocErrorResponse(http.StatusBadRequest, ErrorResponse{}), // Error response body

  )
  // Start the server
  if err := o.Start(); err != nil {
    panic(err)
  }
}
```

Run your server:

```bash
go run main.go
```

Visit [`http://localhost:8080`](http://localhost:8080) to see the response:

```json
{
  "Licence": "MIT",
  "message": "Hello from Okapi Web Framework!"
}
```

Visit [`http://localhost:8080/docs/`](http://localhost:8080/docs/) to see the documentation

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

### Path Syntax Examples

Okapi supports flexible and expressive route path patterns, including named parameters and wildcards:

```go
o.Get("/books/{id}", getBook)       // Named path parameter using curly braces
o.Get("/books/:id", getBook)        // Named path parameter using colon prefix
o.Get("/*", getBook)                // Catch-all wildcard (matches everything)
o.Get("/*any", getBook)             // Catch-all with named parameter (name is ignored)
o.Get("/*path", getBook)            // Catch-all with named parameter
```

Use whichever syntax feels most natural — Okapi normalizes both `{}` and `:` styles for named parameters and supports glob-style wildcards for flexible matching.

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
* **Description**: `description` - OpenAPI description

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
func customMiddleware(next okapi.HandlerFunc) okapi.HandlerFunc {
	return func(c okapi.Context) error {
		start := time.Now()
		err := next(c)
		log.Printf("Request took %v", time.Since(start))
		return err
	}
}

o.Use(customMiddleware)
```

---

### OpenAPI/Swagger Integration

Okapi provides automatic OpenAPI (Swagger) documentation generation with built-in UI support.
The documentation is dynamically generated from your route definitions, keeping your API documentation always in sync with your implementation.

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
        PathPrefix: "/swagger",  // Base path for documentation
        Title:     "Example API",  // Displayed in UI
        Version:   "1.0.0",         // API version
        Contact: okapi.Contact{
        Name:  "API Support",
        Email: "support@example.com",
		},
		},
)
```

### Documenting Routes

Okapi provides two ways to attach OpenAPI documentation to your routes:

#### 1. Composable Functions (Direct Style)

This approach uses individual `okapi.Doc*` functions for each aspect of your route documentation. It’s concise and works well for simple routes.

```go
o.Get("/books", getBooksHandler,
  okapi.DocSummary("List all available books"),
  okapi.DocTags("Books"),
  okapi.DocQueryParam("author", "string", "Filter by author name", false),
  okapi.DocQueryParam("limit", "int", "Maximum results to return (default 20)", false),
  okapi.DocResponse([]Book{}), // Response for OpenAPI docs
  okapi.DocErrorResponse(400, ErrorResponse{}),// Response error for OpenAPI docs
  okapi.DocErrorResponse(401, ErrorResponse{}),// Response error for OpenAPI docs

)
```

#### 2. Fluent Builder Style `okapi.Doc()` + .`Build()`

For more complex or dynamic documentation setup, Okapi offers a fluent builder API.
Use `okapi.Doc()` to begin building, chain options, and call `.Build()` or `.AsOption()` to finalize.

```go
o.Post("/books", createBookHandler,
    okapi.Doc().
    Summary("Add a new book to inventory").
    Tags("Books").
    BearerAuth().
    RequestBody(BookRequest{}).
    Response(Book{}).
	ErrorResponse(400,ErrorResponse{}).
	ErrorResponse(401,ErrorResponse{}).
    Build(),
)
```

### Available Documentation Options

| Method                                       | Description                         |
|----------------------------------------------|-------------------------------------|
| `DocSummary()`/`Doc().Summary()`             | Short endpoint description          |
| `DocTag()/DocTags()`/`Doc().Tags()`          | Groups related endpoints            |
| `DocBearerAuth()`                            | Enables Bearer token authentication |
| `DocRequestBody()`/`Doc().RequestBody()`     | Documents request body structure    |
| `DocResponse()`/`Doc().Response()`           | Documents response structure        |
| `DocPathParam()`/`Doc().PathParam()`         | Documents path parameters           |
| `DocQueryParam()`/`Doc().QueryParam()`       | Documents query parameters          |
| `DocHeader()`/ `Doc().Header()`              | Documents header parameters         |
| `DocErrorResponse()`/`Doc().ErrorResponse()` | Documents response error            |
| `DocDeprecated()`/`Doc().Deprecated()`       | Mark route deprecated               |


### Swagger UI Preview

Okapi automatically generates Swagger UI for all routes:


![Okapi Swagger Interface](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png)

---

### Enabling and Disabling Routes & Groups

Okapi gives you flexible control over your API by allowing routes and route groups to be **dynamically enabled or disabled**. This is a clean and efficient alternative to commenting out code when you want to temporarily remove endpoints.

#### Overview

You can disable:

* **Individual routes** — blocks access to a specific endpoint
* **Route groups** — disables an entire section of your API, including all nested routes

This behavior is reflected both in runtime responses and API documentation.

| Type               | HTTP Response   | Swagger Docs | Affects Child Routes |
|--------------------|-----------------|--------------|----------------------|
| **Disabled Route** | `404 Not Found` | Hidden       | N/A                  |
| **Disabled Group** | `404 Not Found` | Hidden       | Yes — all nested     |

#### Key Features

* Disabled routes/groups return a `404 Not Found`
* Automatically excluded from Swagger/OpenAPI documentation
* Disabling a group recursively disables all nested routes and sub-groups
* No need to comment out code — just call `.Disable()` or `.Enable()`

#### Use Cases

* Temporarily removing endpoints during maintenance
* Controlling access based on feature flags
* Deprecating old API versions
* Creating toggleable test or staging routes

#### Usage Example

```go
app := okapi.Default()

// Create the root API group
api := app.Group("api")

// Define and disable v1 group
v1 := api.Group("v1").Disable() // All v1 routes return 404 and are hidden from docs
v1.Get("/", func(c okapi.Context) error {
    return c.OK(okapi.M{"version": "v1"})
})

// Define active v2 group
v2 := api.Group("v2")
v2.Get("/", func(c okapi.Context) error {
    return c.OK(okapi.M{"version": "v2"})
})

// Start the server
if err := app.Start(); err != nil {
    panic(err)
}
```

#### Behavior Details

* **Disabled Route:**

    * Responds with `404 Not Found`
    * Excluded from Swagger docs

* **Disabled Group:**

    * All nested routes and sub-groups are recursively disabled
    * All affected routes are hidden from Swagger

To re-enable any route or group, simply call the `.Enable()` method or remove the `.Disable()` call.

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
    tls, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
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
    }
```
---

##  Standard Library Compatibility

**Okapi** integrates seamlessly with Go’s `net/http` standard library, enabling you to:

1. Use existing `http.Handler` middleware
2. Register standard `http.HandlerFunc` handlers
3. Combine Okapi-style routes with standard library handlers

This makes Okapi ideal for gradual adoption or hybrid use in existing Go projects.


### Middleware Compatibility

Okapi’s `UseMiddleware` bridges standard `http.Handler` middleware into Okapi’s middleware system. This lets you reuse the wide ecosystem of community-built middleware—such as logging, metrics, tracing, compression, and more.

#### Signature

```go
func (o *Okapi) UseMiddleware(middleware func(http.Handler) http.Handler)
```

#### Example: Injecting a Custom Header

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

### Handler Compatibility

You can register any `http.HandlerFunc` using `HandleStd`, or use full `http.Handler` instances via `HandleHTTP`. These retain Okapi’s routing and middleware features while supporting familiar handler signatures.

#### HandleStd Signature

```go
func (o *Okapi) HandleStd(method, path string, handler http.HandlerFunc, opts ...RouteOption)
```

#### Example: Basic Standard Library Handler

```go
o := okapi.Default()

o.HandleStd("GET", "/greeting", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello from Okapi!"))
})
```

---

### Migration Tips

Migrating an existing `net/http` application? Okapi makes it painless.

#### Mixed Routing Support

You can mix Okapi and standard handlers in the same application:

```go
// Okapi-style route
o.Handle("GET", "/okapi", func(c okapi.Context) error {
    return c.OK(okapi.M{"status": "ok"})
})

// Standard library handler
o.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("standard response"))
})
```


#### Error Handling Differences
* `http.HandlerFunc`: must manually call `w.WriteHeader(...)`
* `okapi.Handle`: can return an error or use helpers like `c.JSON`, `c.Text`, `c.OK`, `c.ErrorNotFound()` or `c.AbortBadRequest()`


---

###  Explore Another Project: Goma Gateway

Are you building a microservices architecture?
Do you need a powerful yet lightweight API Gateway to secure and manage your services effortlessly?

Check out my other project — **[Goma Gateway](https://github.com/jkaninda/goma-gateway)**.

**Goma Gateway** is a high-performance, declarative API Gateway designed for modern microservices. It includes a rich set of built-in middleware for:

* Security: ForwardAuth, Basic Auth, JWT, OAuth
* Caching and rate limiting
* Simple configuration, minimal overhead

Whether you're managing internal APIs or exposing public endpoints, Goma Gateway helps you do it cleanly and securely.


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
