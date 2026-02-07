---
title: Request Handling
layout: default
parent: Core Concepts
nav_order: 2
---

# Request Handling

Okapi provides powerful and flexible request handling capabilities, allowing you to easily access and process various parts of incoming HTTP requests.

## Path Parameters

Extract path parameters from the URL:

```go
o.Get("/books/:id", func(c *okapi.Context) error {
    id := c.Param("id")
    return c.String(http.StatusOK, id)
})
```

## Query Parameters

Access query string parameters:

```go
o.Get("/books", func(c *okapi.Context) error {
    name := c.Query("name")
    return c.String(http.StatusOK, name)
})
```

## Form Data

### Multipart Form (`multipart/form-data`)

Handle standard form fields and file uploads:

```go
o.Post("/books", func(c *okapi.Context) error {
    name := c.FormValue("name")
    price := c.FormValue("price")

    logo, err := c.FormFile("logo")
    if err != nil {
        return c.AbortBadRequest("Bad request", err)
    }
    
    file, err := logo.Open()
    if err != nil {
        return c.AbortBadRequest("Bad request", err)
    }
    defer file.Close()
    
    // You can now read or save the uploaded file
    return c.String(http.StatusOK, "File uploaded successfully")
})
```

## Struct Binding

Okapi provides powerful request binding that automatically maps incoming request data into Go structs. It supports two complementary binding styles:

### 1. Flat Binding

In Flat Binding, you define a single struct where each field can be sourced from any part of the request. This allows you to mix request body fields (JSON, XML, YAML, Protobuf, Form) with query parameters, headers, cookies, and path parameters.

```go
type Book struct {
    ID      int    `json:"id" path:"id" query:"id" form:"id"`
    Name    string `json:"name" xml:"name" form:"name" minLength:"4" maxLength:"50" required:"true"`
    Price   int    `json:"price" form:"price" required:"true"`
    Logo    *multipart.FileHeader `form:"logo" required:"true"`
    Content string `header:"Content-Type" json:"content-type" xml:"content-type" required:"true"`
    // Supports both ?tags=a&tags=b and ?tags=a,b
    Tags    []string `form:"tags" query:"tags" default:"a,b"`
    Year    int      `json:"year" yaml:"year" description:"Book year" deprecated:"true"`
	SessionID string   `cookie:"session_id"`
}

o.Post("/books", func(c *okapi.Context) error {
    book := &Book{}
    if err := c.Bind(book); err != nil {
        return c.ErrorBadRequest(err)
    }
    return c.JSON(http.StatusOK, book)
})
```

### 2. Body Field Binding (Recommended)

In Body Field Binding, your struct defines a dedicated `Body` field that represents the main request payload. Other fields represent query params, headers, cookies, or path parameters.

```go
type BookRequest struct {
    Body struct {
        Name  string `json:"name" minLength:"4" maxLength:"50" required:"true"`
        Price int    `json:"price" required:"true"`
        Logo  *multipart.FileHeader `form:"logo" required:"true"`
    } `json:"body"` // Request body

    ID        int      `json:"id" param:"id" query:"id"`        // from path or query
    Tags      []string `query:"tags" default:"a,b"`             // supports arrays
    APIKey    string   `header:"X-API-Key" required:"true"`     // from header
    SessionID string   `cookie:"session_id" json:"session_id"`  // from cookie
}

o.Post("/books", func(c *okapi.Context) error {
    bookReq := &BookRequest{}
    if err := c.Bind(bookReq); err != nil {
        return c.ErrorBadRequest(err)
    }
    return c.Respond(bookReq)
})
```

## Supported Sources

| Source           | Tag(s)          | Description                                                                                   |
|------------------|-----------------|-----------------------------------------------------------------------------------------------|
| Path parameters  | `path`, `param` | Extracted from path variables (e.g. `/books/:id` or `/books/{id:int}`).                       |
| Query parameters | `query`         | Parses query strings; supports repeated arrays (`?tags=a&tags=b`) and comma-separated values. |
| Headers          | `header`        | Reads values from HTTP request headers.                                                       |
| Cookies          | `cookie`        | Reads values from cookies.                                                                    |
| Form fields      | `form`          | Supports both `application/x-www-form-urlencoded` and `multipart/form-data` (file uploads).   |
| JSON body        | `json`          | Decodes when `Content-Type: application/json`.                                                |
| XML body         | `xml`           | Decodes when `Content-Type: application/xml`.                                                 |

## OpenAPI & Documentation Tags

These struct tags control how fields appear in the generated **OpenAPI 3 specification** and Swagger UI.

| Tag(s)               | Description                                                                 |
|----------------------|-----------------------------------------------------------------------------|
| `description`, `doc` | Adds descriptive documentation for the field in the OpenAPI schema.         |
| `deprecated:"true"`  | Marks the field as deprecated in the generated OpenAPI documentation.       |
| `hidden:"true"`      | Excludes the field from the generated OpenAPI specification and Swagger UI. |
| `example:"..."`      | Adds an example value for the field in the OpenAPI schema.                  |

