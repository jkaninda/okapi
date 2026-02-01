---
title: Response Handling
layout: default
parent: Core Concepts
nav_order: 3
---

# Response Handling

Okapi provides a rich set of response methods to send various types of responses back to the client with appropriate HTTP status codes and content types.

## Response Struct Binding

When using `c.Respond()`, Okapi automatically serializes the response struct into the HTTP response. It inspects struct tags to determine:

* The **HTTP status code**
* **Response headers**
* **Cookies**
* And the **response body** (encoded according to the `Accept` header)

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
    XRequestID string `header:"X-Request-ID" json:"x-request-id"`

    // Cookies
    SessionID string `cookie:"session_id" json:"session_id"`
}

o.Get("/books/:id", func(c *okapi.Context) error {
    book := BookResponse{
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
    return c.Respond(book)
})
```

## Common Response Methods

### JSON Response

```go
o.Get("/books", func(c *okapi.Context) error {
    books := []Book{{ID: 1, Name: "Go Programming"}}
    return c.JSON(http.StatusOK, books)
})
```

### Text Response

```go
o.Get("/hello", func(c *okapi.Context) error {
    return c.String(http.StatusOK, "Hello, World!")
})
```

### HTML Response

```go
o.Get("/page", func(c *okapi.Context) error {
    html := "<h1>Welcome</h1>"
    return c.HTML(http.StatusOK, html)
})
```

### XML Response

```go
o.Get("/books", func(c *okapi.Context) error {
    books := []Book{{ID: 1, Name: "Go Programming"}}
    return c.XML(http.StatusOK, books)
})
```

### File Response

```go
o.Get("/download", func(c *okapi.Context) error {
    return c.ServeFile("path/to/file.pdf")
})
```

## Convenience Methods

### Success Responses

```go
// 200 OK
return c.OK(data)

// 201 Created
return c.Created(data)

// 204 No Content
return c.NoContent()
```

### Error Responses

```go
// 400 Bad Request
return c.ErrorBadRequest(err)

// 401 Unauthorized
return c.ErrorUnauthorized(err)

// 403 Forbidden
return c.ErrorForbidden(err)

// 404 Not Found
return c.ErrorNotFound(err)

// 500 Internal Server Error
return c.ErrorInternalServerError(err)
```

### Abort Methods

These methods immediately stop processing and send an error response:

```go
// Abort with 400
return c.AbortBadRequest("Invalid input", err)

// Abort with 401
return c.AbortUnauthorized("Not authenticated", err)

// Abort with 403
return c.AbortForbidden("Access denied", err)

// Abort with 404
return c.AbortNotFound("Resource not found", err)

// Abort with 500
return c.AbortInternalServerError("Server error", err)
```

## Setting Headers

```go
o.Get("/", func(c *okapi.Context) error {
    c.SetHeader("X-Custom-Header", "value")
    return c.OK(okapi.M{"message": "Success"})
})
```

## Setting Cookies

```go
o.Get("/login", func(c *okapi.Context) error {
    c.SetCookie(&http.Cookie{
        Name:     "session",
        Value:    "abc123",
        Path:     "/",
        MaxAge:   3600,
        HttpOnly: true,
        Secure:   true,
    })
    return c.OK(okapi.M{"message": "Logged in"})
})
```

## Template Rendering

```go
o.Get("/", func(c *okapi.Context) error {
    return c.Render(http.StatusOK, "welcome", okapi.M{
        "title":   "Welcome Page",
        "message": "Hello from Okapi!",
    })
})
```

See the [Templating](/docs/features/templating) section for more details on setting up templates.


