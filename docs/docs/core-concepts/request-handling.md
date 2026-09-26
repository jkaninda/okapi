---
title: Request Handling
sidebar_position: 2
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
    return c.String(http.StatusOK, "Uploaded "+name+" ("+price+")")
})
```

## Client IP Address

`c.RealIP()` returns the client's IP address. By default it trusts the
`X-Forwarded-For` and `X-Real-IP` headers, which **any client can set**, so the
value it returns is forgeable. Do not use it for rate limiting, IP allow-lists or
audit trails until you have told Okapi which peers may set those headers:

```go
o := okapi.New(okapi.WithTrustedProxies("10.0.0.0/8", "192.0.2.7"))
```

Entries are CIDR blocks or bare addresses. Once configured:

- the headers are honoured only for connections coming from one of those peers,
  and ignored otherwise — `RealIP()` then falls back to the connection address;
- `X-Forwarded-For` is walked **from the right**, and the first address that is
  not itself a trusted proxy is returned. (The leftmost entry is the one a client
  writes, which is why it is not used.)

A configuration containing an invalid entry is rejected as a whole and logged as
an error: no peer is trusted until it is corrected, so a typo cannot silently
widen what is trusted. Calling `WithTrustedProxies()` with no arguments restores
the spoofable default.

`c.Request().RemoteAddr` is always the connection's own address, whatever the
headers say.

## Request Body Size

Every binder reads the body through a cap: 8 MB by default, adjustable with
`okapi.WithMaxRequestBody(bytes)`. The cap applies to JSON, XML, YAML and
protobuf bodies as well as to url-encoded and multipart forms, so a large upload
is rejected instead of being buffered or spilled to disk. A body over the limit
surfaces as an `*http.MaxBytesError` wrapped in the binding error.

If you install the `BodyLimit` middleware, its `MaxBytes` takes precedence — it
has already bounded the body by the time a binder runs.

```go
o := okapi.New(okapi.WithMaxRequestBody(32 << 20)) // 32 MB
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

In Body Field Binding, your struct declares a field **named `Body`** that
represents the main request payload. Other fields represent query params,
headers, cookies, or path parameters.

The field is recognised by its Go name, `Body`. A `json:"body"` tag does not
make a field the request body, and it is not needed on a field that is already
called `Body`.

```go
type BookRequest struct {
    // Request body — recognised by the field name, not by a tag
    Body struct {
        Name  string `json:"name" minLength:"4" maxLength:"50" required:"true"`
        Price int    `json:"price" required:"true"`
        Logo  *multipart.FileHeader `form:"logo" required:"true"`
    }

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
| YAML body        | `yaml`          | Decodes when the content type is a YAML type.                                                 |
| Protobuf body    | —               | Decoded with `BindProtoBuf` when the content type is protobuf and the target is a `proto.Message`. |

### Source Precedence

A field may carry several source tags. The **first non-empty value wins**, in a
fixed order:

1. `param`
2. `path`
3. `query`
4. `form` — flat structs only
5. `header`
6. `cookie`

The order is the same on every request, and an absent path parameter no longer
blanks a value that a query parameter had already supplied. `default` is applied
only when no source produced a value, and `required:"true"` is checked last.

For `multipart/form-data` the multipart binder uses the same order (param, path,
query, form, header).

### Binding Errors

Decode errors are returned rather than swallowed. Malformed JSON surfaces as
`invalid JSON body: …` (and likewise for XML, YAML and protobuf) instead of a
misleading `field X is required`, and a mistyped field is reported instead of
being bound as its zero value. An empty body is not an error: the other sources
may still satisfy the struct.

## OpenAPI & Documentation Tags

These struct tags control how fields appear in the generated **OpenAPI specification** (3.1 by default, also published as 3.0) and in the documentation UIs.

| Tag(s)               | Description                                                                 |
|----------------------|-----------------------------------------------------------------------------|
| `description`, `doc` | Adds descriptive documentation for the field in the OpenAPI schema.         |
| `deprecated:"true"`  | Marks the field as deprecated in the generated OpenAPI documentation.       |
| `hidden:"true"`      | Excludes the field from the generated OpenAPI specification and the UIs.    |
| `example:"..."`      | Adds an example value for the field in the OpenAPI schema, emitted with the field's JSON type (`example:"1"` on an int becomes `1`, not `"1"`). |
