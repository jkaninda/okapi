---
title: Response Handling
sidebar_position: 3
---

# Response Handling

Okapi provides a rich set of response methods to send various types of responses back to the client with appropriate HTTP status codes and content types.

## Quick Start

The simplest way to send a response is using convenience methods:

```go
o.Get("/books/:id", func(c *okapi.Context) error {
    book := Book{
        ID:    1,
        Name:  "The Great Go Book",
        Price: 20,
    }
    return c.OK(book) // Sends 200 OK with JSON
})
```

## Response Methods

### JSON Responses

Send JSON data with a specific status code:

```go
o.Get("/books", func(c *okapi.Context) error {
    books := []Book{{ID: 1, Name: "Go Programming"}}
    return c.JSON(http.StatusOK, books)
})
```

### Text Responses

Send plain text:

```go
o.Get("/hello", func(c *okapi.Context) error {
    return c.String(http.StatusOK, "Hello, World!")
})
```

### HTML Responses

`c.HTML(code, file, data)` parses an HTML **template file** and executes it, and
`c.HTMLView(code, templateString, data)` does the same for a template held in a
string. Both escape interpolated values through `html/template`.

```go
o.Get("/page", func(c *okapi.Context) error {
    return c.HTML(http.StatusOK, "views/page.html", okapi.M{"title": "Welcome"})
})

o.Get("/inline", func(c *okapi.Context) error {
    return c.HTMLView(http.StatusOK, "<h1>{{.title}}</h1>", okapi.M{"title": "Welcome"})
})
```

To send HTML you already hold as bytes, without templating, use `c.Data`:

```go
o.Get("/raw", func(c *okapi.Context) error {
    return c.Data(http.StatusOK, "text/html", []byte("<h1>Welcome</h1>"))
})
```

### XML Responses

Send XML data:

```go
o.Get("/books", func(c *okapi.Context) error {
    books := []Book{{ID: 1, Name: "Go Programming"}}
    return c.XML(http.StatusOK, books)
})
```

### File Responses

Serve files for download. The file-serving helpers write to the response
directly and return nothing, so return `nil` yourself:

```go
o.Get("/download", func(c *okapi.Context) error {
    c.ServeFile("path/to/file.pdf")
    return nil
})
```

`ServeFile`, `ServeFileAttachment` and `ServeFileInline` use the path as given.
`http.ServeFile`'s `".."` precaution only inspects `r.URL.Path`, so it does not
protect a path your handler assembled. For a name that comes from the request,
use `c.ServeFileFrom(root, name)`, which refuses `..` and stays inside `root`:

```go
o.Get("/files/{name}", func(c *okapi.Context) error {
    c.ServeFileFrom("public", c.Param("name"))
    return nil
})
```

## Convenience Methods

Okapi provides shorthand methods for common HTTP status codes.

### Success Responses

```go
// 200 OK
return c.OK(data)

// 201 Created
return c.Created(data)

// 204 No Content
return c.NoContent()
```

### Client Error Responses

These write the value you pass as the response body, unchanged:

```go
// 400 Bad Request
return c.ErrorBadRequest(payload)

// 401 Unauthorized
return c.ErrorUnauthorized(payload)

// 403 Forbidden
return c.ErrorForbidden(payload)

// 404 Not Found
return c.ErrorNotFound(payload)
```

### Server Error Responses

```go
// 500 Internal Server Error
return c.ErrorInternalServerError(payload)
```

## Returning an Error From a Handler

An `error` returned by a handler is **not** sent to the client. It is logged, and
the configured `ErrorHandler` answers with a generic 500. When you want a
specific status or message, respond with one of the `Abort*` helpers instead of
returning the error:

```go
o.Get("/books/:id", func(c *okapi.Context) error {
    book, err := findBook(c.Param("id"))
    if err != nil {
        // Logged, and the client gets a generic 500
        return err
    }
    if book == nil {
        // Explicit status and message
        return c.AbortNotFound("Book not found")
    }
    return c.OK(book)
})
```

## Advanced Response Handling

## Response Struct Binding

When using `c.Respond()` / `c.Return()`, Okapi automatically serializes the response struct into the HTTP response.

It inspects the struct to determine:

* the **HTTP status code**
* **response headers**
* **cookies**
* and the **response body** (encoded according to the `Accept` header).

```go
type BookResponse struct {
    // HTTP status code — the field must be named Status (default: 200)
    Status int `json:"status"`

    // Response body — the field must be named Body
    Body struct {
        ID    int    `json:"id"`
        Name  string `json:"name"`
        Price int    `json:"price"`
    }

    // Custom headers
    XRequestID string `header:"X-Request-ID" json:"x_request_id"`

    // Cookies
    SessionID string `cookie:"session_id" json:"session_id"`
}

o.Get("/books/:id", func(c *okapi.Context) error {
    response := BookResponse{
        Status: http.StatusOK,
        Body: struct {
            ID    int    `json:"id"`
            Name  string `json:"name"`
            Price int    `json:"price"`
        }{
            ID:    1,
            Name:  "The Great Go Book",
            Price: 20,
        },
        XRequestID: "req-12345",
        SessionID:  "sess-67890",
    }
    return c.Respond(response)
})
```

**How the struct is read:**

- A field named `Status`, of type `int` and greater than zero, sets the HTTP
  status code. There is no `status` struct tag — the field name is what matters.
  Anything else falls back to `200`.
- A field named `Body` provides the response body. Again the field name matters,
  not its `json` tag.
- `header:"Header-Name"` sets a response header.
- `cookie:"cookie_name"` sets a cookie (path `/`).
- Any other exported field, apart from `Status` and `Body`, is written as a
  response header named after its `json` tag, or after the field name when there
  is none. Keep this in mind before adding an extra field to a response struct.
- Unexported fields are skipped.

The body is encoded from the `Accept` header: XML, YAML, plain text and HTML are
honoured, and anything else — including a missing `Accept` — is encoded as JSON.

### Setting Headers Manually

```go
o.Get("/", func(c *okapi.Context) error {
    c.SetHeader("X-Custom-Header", "value")
    c.SetHeader("Cache-Control", "max-age=3600")
    return c.OK(okapi.M{"message": "Success"})
})
```

### Setting Cookies

`c.SetCookie` takes the cookie's parts as arguments:

```go
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
```

```go
o.Get("/login", func(c *okapi.Context) error {
    c.SetCookie("session", "abc123", 3600, "/", "", true, true)
    return c.OK(okapi.M{"message": "Logged in"})
})
```

For full control — `SameSite`, `Expires`, partitioned cookies — write the cookie
yourself on the response:

```go
o.Get("/login", func(c *okapi.Context) error {
    http.SetCookie(c.Response(), &http.Cookie{
        Name:     "session",
        Value:    "abc123",
        Path:     "/",
        MaxAge:   3600,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })
    return c.OK(okapi.M{"message": "Logged in"})
})
```

## Abort Methods

Abort methods immediately stop request processing and send an error response through the configured `ErrorHandler`. They're useful in middleware or when you need to halt execution:

```go
o.Use(func(c *okapi.Context) error {
    token := c.Header("Authorization")
    if token == "" {
        return c.AbortUnauthorized("Missing authorization token")
    }
    return c.Next()
})
```

`c.Header(key)` reads a **request** header. Use `c.SetHeader(key, value)` to set
one on the response.

**Available abort methods:**

The error argument is optional and variadic; when omitted, the message is used as
the underlying error.

```go
// 400 Bad Request
return c.AbortBadRequest("Invalid input", err)

// 401 Unauthorized
return c.AbortUnauthorized("Not authenticated", err)

// 403 Forbidden
return c.AbortForbidden("Access denied", err)

// 404 Not Found
return c.AbortNotFound("Resource not found", err)

// 500 Internal Server Error
return c.AbortInternalServerError("Server error", err)
```

## Template Rendering

Render HTML templates with data:

```go
o.Get("/", func(c *okapi.Context) error {
    return c.Render(http.StatusOK, "welcome", okapi.M{
        "title":   "Welcome Page",
        "message": "Hello from Okapi!",
    })
})
```

`c.Render` requires a renderer to be attached; without one it returns
`okapi.ErrNoRenderer`. See the [Templating](../features/templating.md) section
for details on configuring template engines and on how templates are named.

## Examples

### Complete CRUD Endpoint

```go
type Book struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Price int    `json:"price"`
}

// List books
o.Get("/books", func(c *okapi.Context) error {
    books := []Book{{ID: 1, Name: "Go Programming"}}
    return c.OK(books)
})

// Get single book
o.Get("/books/:id", func(c *okapi.Context) error {
    book, ok := findBook(c.Param("id"))
    if !ok {
        return c.AbortNotFound("Book not found")
    }
    return c.OK(book)
})

// Create book
o.Post("/books", func(c *okapi.Context) error {
    var book Book
    if err := c.BindJSON(&book); err != nil {
        return c.AbortBadRequest("Invalid request body", err)
    }
    // Save book...
    return c.Created(book)
})

// Delete book
o.Delete("/books/:id", func(c *okapi.Context) error {
    if !deleteBook(c.Param("id")) {
        return c.AbortNotFound("Book not found")
    }
    return c.NoContent()
})
```
