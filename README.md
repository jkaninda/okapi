# OKAPI - Lightweight Go Web Framework

[![Tests](https://github.com/jkaninda/okapi/actions/workflows/tests.yml/badge.svg)](https://github.com/jkaninda/okapi/actions/workflows/tests.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jkaninda/okapi)](https://goreportcard.com/report/github.com/jkaninda/okapi)
[![Go Reference](https://pkg.go.dev/badge/github.com/jkaninda/okapi.svg)](https://pkg.go.dev/github.com/jkaninda/okapi)
[![GitHub Release](https://img.shields.io/github/v/release/jkaninda/okapi)](https://github.com/jkaninda/okapi/releases)


**Okapi** is a modern, minimalist HTTP web framework for Go, inspired by the simplicity of FastAPI. Designed to be intuitive, lightweight, and powerful, Okapi makes it easy to build fast and flexible web applications and REST APIs.

<p align="center">
  <img src="https://raw.githubusercontent.com/jkaninda/okapi/main/logo.png" width="150" alt="Okapi logo">
</p>

---

## Features

*  Clean and expressive API design
*  Powerful binding from JSON, forms, query, headers, and path parameters
*  Route grouping and middleware chaining
*  Built-in middleware: Basic Auth, JWT, OAuth
*  Easy custom middleware support
*  Templating engine integration
*  Static file serving
*  Built entirely on Go’s standard library
*  Simple and clear documentation

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
	o := okapi.New()

	o.Get("/", func(c okapi.Context) error {
		return c.JSON(http.StatusOK, okapi.M{"message": "Welcome to Okapi!"})
	})

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
v1.Get("/users", getUsers)

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

	// Supports both ?tags=a&tags=b and ?tags=a,b
	Tags []string `form:"tags" default:"a,b"`
}

o.Post("/books", func(c okapi.Context) error {
	book := &Book{}
	if err := c.Bind(book); err != nil {
		return err
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
	return c.ServeFile("public/favicon.ico")
})

// Serve an entire directory
o.Static("/static", "public/assets")
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

---
## Give a Star! ⭐

⭐ If you find Okapi useful, please consider giving it a star on [GitHub](https://github.com/jkaninda/okapi)!


## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Copyright

Copyright (c) 2025 Jonas Kaninda
