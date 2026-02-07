---
title: Response Handling
layout: default
parent: Core Concepts
nav_order: 3
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
    return c.Ok(book)  // Sends 200 OK with JSON
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

Send HTML content:

```go
o.Get("/page", func(c *okapi.Context) error {
    html := "<h1>Welcome</h1>"
    return c.HTML(http.StatusOK, html)
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

Serve files for download:

```go
o.Get("/download", func(c *okapi.Context) error {
    return c.ServeFile("path/to/file.pdf")
})
```

## Convenience Methods

Okapi provides shorthand methods for common HTTP status codes.

### Success Responses

```go
// 200 OK
return c.Ok(data)

// 201 Created
return c.Created(data)

// 204 No Content
return c.NoContent()
```

### Client Error Responses

```go
// 400 Bad Request
return c.ErrorBadRequest(err)

// 401 Unauthorized
return c.ErrorUnauthorized(err)

// 403 Forbidden
return c.ErrorForbidden(err)

// 404 Not Found
return c.ErrorNotFound(err)
```

### Server Error Responses

```go
// 500 Internal Server Error
return c.ErrorInternalServerError(err)
```

## Advanced Response Handling

## Response Struct Binding

When using `c.Respond()/c.Return()`, Okapi automatically serializes the response struct into the HTTP response.

It inspects struct tags to determine:

* the **HTTP status code**
* **response headers**
* **cookies**
* and the **response body** (encoded according to the `Accept` header).

```go
type BookResponse struct {
    // HTTP status code (default: 200)
    Status int `status:"true" json:"status"`

    // Response body
    Body struct {
        ID    int    `json:"id"`
        Name  string `json:"name"`
        Price int    `json:"price"`
    } `json:"body"`

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

**Supported struct tags:**
- `status:"true"` - Sets the HTTP status code for the response
- `json:"body"` - Sets the response body 
- `header:"Header-Name"` - Sets a response header
- `cookie:"cookie_name"` - Sets a cookie value

### Setting Headers Manually

```go
o.Get("/", func(c *okapi.Context) error {
    c.SetHeader("X-Custom-Header", "value")
    c.SetHeader("Cache-Control", "max-age=3600")
    return c.Ok(okapi.M{"message": "Success"})
})
```

### Setting Cookies

```go
o.Get("/login", func(c *okapi.Context) error {
    c.SetCookie(&http.Cookie{
        Name:     "session",
        Value:    "abc123",
        Path:     "/",
        MaxAge:   3600,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
    })
    return c.Ok(okapi.M{"message": "Logged in"})
})
```

## Abort Methods

Abort methods immediately stop request processing and send an error response. They're useful in middleware or when you need to halt execution:

```go
o.Use(func(c *okapi.Context) error {
    token := c.GetHeader("Authorization")
    if token == "" {
        return c.AbortUnauthorized("Missing authorization token", nil)
    }
    return c.Next()
})
```

**Available abort methods:**

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

See the [Templating](/features/templating) section for details on configuring template engines.


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
    return c.Ok(books)
})

// Get single book
o.Get("/books/:id", func(c *okapi.Context) error {
    id := c.Param("id")
    book := Book{ID: 1, Name: "Go Programming", Price: 50}
    return c.Ok(book)
})

// Create book
o.Post("/books", func(c *okapi.Context) error {
    var book Book
    if err := c.BindJSON(&book); err != nil {
        return c.ErrorBadRequest(err)
    }
    // Save book...
    return c.Created(book)
})

// Delete book
o.Delete("/books/:id", func(c *okapi.Context) error {
    id := c.Param("id")
    // Delete book...
    return c.NoContent()
})
```
