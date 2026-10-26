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

`c.Param` is an alias of `c.PathParam`. Path parameters are always strings at
runtime; a type hint such as `{id:int}` only affects the OpenAPI document.
`c.Path()` returns the raw request path (e.g. `/books/42`). `c.Params()`, which
returns every path parameter as a `map[string]string`, is deprecated in favour of
`c.Param`.

## Query Parameters

Access query string parameters:

```go
o.Get("/books", func(c *okapi.Context) error {
    name := c.Query("name")
    return c.String(http.StatusOK, name)
})
```

| Method               | Returns               | Description                                                                                                                       |
|----------------------|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------|
| `c.Query(key)`       | `string`              | First value of the parameter, or `""` when absent.                                                                                |
| `c.QueryArray(key)`  | `[]string`            | Every value, accepting both `?tags=a&tags=b` and `?tags=a,b` (values are split on commas and trimmed). `nil` when absent.         |
| `c.QueryMap()`       | `map[string]string`   | All parameters, keeping only the **first** value of each (`?tags=a,b&tags=c` gives `tags: "a,b"`).                              |

## Headers and Request Metadata

| Method                   | Returns               | Description                                                                                           |
|--------------------------|-----------------------|-------------------------------------------------------------------------------------------------------|
| `c.Header(key)`          | `string`              | A request header value.                                                                               |
| `c.Headers()`            | `map[string][]string` | All request headers (the request's `http.Header`).                                                    |
| `c.ContentType()`        | `string`              | The raw `Content-Type` header, parameters included (e.g. `application/json; charset=utf-8`).         |
| `c.Accept()`             | `[]string`            | The `Accept` header split on commas (entries are not trimmed). `nil` when absent.                     |
| `c.AcceptLanguage()`     | `[]string`            | The `Accept-Language` header split on commas and trimmed, quality values kept (`["en-US", "fr;q=0.8"]`). `nil` when absent. |
| `c.Referer()`            | `string`              | The `Referer` header.                                                                                 |
| `c.Cookie(name)`         | `(string, error)`     | A cookie value; the error is `http.ErrNoCookie` when it is missing.                                   |
| `c.RealIP()`             | `string`              | The client IP address — see [Client IP Address](#client-ip-address) before trusting it.               |
| `c.IsSSE()`              | `bool`                | `true` for a `GET` whose `Accept` header is exactly `text/event-stream`.                              |
| `c.IsWebSocketUpgrade()` | `bool`                | `true` for a `GET` whose `Upgrade` header is exactly `websocket`.                                     |
| `c.Request()`            | `*http.Request`       | The underlying request, for anything not covered above.                                               |

```go
o.Get("/greeting", func(c *okapi.Context) error {
    langs := c.AcceptLanguage() // e.g. ["fr-FR", "en;q=0.8"]
    if len(langs) > 0 && strings.HasPrefix(langs[0], "fr") {
        return c.String(http.StatusOK, "Bonjour")
    }
    return c.String(http.StatusOK, "Hello")
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

### Multipart Memory

When a multipart form is parsed, up to `MaxMultipartMemory` bytes of it are kept
in memory (32 MB by default) and the rest of the file parts are stored in
temporary files. The whole body is still bounded by the
[request body cap](#request-body-size), which defaults to 8 MB, so raise that cap
too if you accept larger uploads.

```go
o := okapi.New(
    okapi.WithMaxRequestBody(100 << 20),     // accept bodies up to 100 MB
    okapi.WithMaxMultipartMemory(16 << 20),  // keep at most 16 MB in memory
)
// or, chained: o.WithMaxMultipartMemory(16 << 20)
```

Inside a handler, `c.MaxMultipartMemory()` returns the limit that applies to the
current request. `c.SetMaxMultipartMemory(n)` changes it for that request only,
before the form is parsed; other requests keep the application's limit. Passed
to the option, zero or less means the 32 MB default; the setter ignores such
values.

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

#### Body Payload Types

The `Body` field may be any type a decoder can fill, not only a struct:

- **A struct, or a pointer to a struct**, is decoded from JSON, XML, YAML,
  url-encoded or multipart bodies and then validated with its tags, including
  the tags of nested structs and of struct elements of slices.
- **A slice, map, string or other value** is decoded as a whole from a JSON, XML
  or YAML body. A form body is rejected (`form data needs a struct body, not
  []main.Book`). The elements of a slice of structs (or of pointers to structs)
  are validated with their tags; an error names the element, as in
  `Body[1].Name`.

An empty body is not an error in itself: a slice, map or string `Body` keeps its
zero value, while a struct payload is still validated, so its `required` fields
are reported.

A field is only treated as the request body on the **top-level** input. A payload
struct may have its own field named `Body`; it is ordinary data, decoded by its
tags like any other field.

```go
type BulkCreateInput struct {
    DryRun bool   `query:"dry_run"`
    Body   []Book // the body is a JSON array: [{"name": "..."}, ...]
}

type Message struct {
    Subject string `json:"subject" required:"true"`
    Body    string `json:"body"` // a payload field, not the request body
}

type SendInput struct {
    Body *Message
}
```

### Binding Methods

| Method                                   | Description                                                                                                                                                                                                |
|------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `c.Bind(&v)`                             | Binds from every source according to the content type and the struct tags, supports both binding styles, applies defaults and validates. This is what `okapi.Handle` and friends use.                      |
| `c.B(&v)`                                | Same as `Bind`, with the error prefixed by `binding error: `.                                                                                                                                              |
| `c.ShouldBind(&v)`                       | Same as `Bind`, returning `(bool, error)`: `true` when binding and validation succeeded.                                                                                                                    |
| `c.BindJSON(&v)`, `c.BindXML(&v)`, `c.BindYAML(&v)` | Decode the body only, ignoring the content type, source tags and validation tags. An empty body returns `io.EOF`.                                                                               |
| `c.BindProtoBuf(msg)`                    | Unmarshals the body into a `proto.Message`.                                                                                                                                                                |
| `c.BindMultipart(&v)`                    | Parses a `multipart/form-data` body and binds form fields, files and the param, path, query and header tags of a flat struct, then validates.                                                              |
| `c.BindForm(&v)`, `c.BindQuery(&v)`      | Both read the merged query string and url-encoded body into a flat struct, then validate. A field is matched by its `form` (for `BindForm`) or `query` (for `BindQuery`) tag, or else by its `json` name; slice fields take every value, also split on commas, and `default` tags apply. |

All of them read the body through the [request body cap](#request-body-size).

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

Decode errors are returned rather than swallowed, and each error names the field
it is about by its full path, such as `field Body.Items[2].Name is required`. An
empty body is not an error: the other sources may still satisfy the struct. See
[Binding Errors](../features/validation.md#binding-errors) for the message
formats and the `BindError` type.

## OpenAPI & Documentation Tags

These struct tags control how fields appear in the generated **OpenAPI specification** (3.1 by default, also published as 3.0) and in the documentation UIs.

| Tag(s)               | Description                                                                 |
|----------------------|-----------------------------------------------------------------------------|
| `description`, `doc` | Adds descriptive documentation for the field in the OpenAPI schema.         |
| `deprecated:"true"`  | Marks the field as deprecated in the generated OpenAPI documentation.       |
| `hidden:"true"`      | Excludes the field from the generated OpenAPI specification and the UIs.    |
| `example:"..."`      | Adds an example value for the field in the OpenAPI schema, emitted with the field's JSON type (`example:"1"` on an int becomes `1`, not `"1"`). |

## Context Store

Every request's `*okapi.Context` carries a key/value store that is safe for
concurrent use. Middleware typically puts values there for the handlers that run
after it:

```go
o.Use(func(c *okapi.Context) error {
    c.Set("requestStart", time.Now())
    c.Set("tenantID", c.Header("X-Tenant-ID"))
    return c.Next()
})

o.Get("/me", func(c *okapi.Context) error {
    tenant := c.GetString("tenantID")
    start, _ := c.GetTime("requestStart")
    return c.OK(map[string]any{"tenant": tenant, "since": time.Since(start).String()})
})
```

| Method            | Returns          | When the key is missing or has another type                                                     |
|-------------------|------------------|--------------------------------------------------------------------------------------------------|
| `c.Set(key, v)`   | —                | Stores any value, replacing an existing one.                                                    |
| `c.Get(key)`      | `(any, bool)`    | `nil, false` when missing.                                                                       |
| `c.GetString(key)`| `string`         | `""`.                                                                                            |
| `c.GetBool(key)`  | `bool`           | A `bool`, or the string `"true"`; otherwise `false`.                                             |
| `c.GetInt(key)`   | `int`            | An `int`, a numeric string or a `float64` (truncated); otherwise `0`.                           |
| `c.GetInt64(key)` | `int64`          | An `int64`, a numeric string or a `float64` (truncated); otherwise `0`. A value stored as `int` gives `0`. |
| `c.GetTime(key)`  | `(time.Time, bool)` | Only a stored `time.Time`; otherwise the zero time and `false`.                              |

The store lives as long as the request. `c.Copy()` returns a new `*Context` with
its **own** store, initialised with the current entries, and the same request and
response writer; changes made to one store afterwards are not seen by the other.
It keeps the application's settings (logger, error handler, multipart limit,
trusted proxies). The copy still writes to the original response, so do not use
it to answer the request once the handler has returned.

## Context Lifecycle

Okapi creates one `*okapi.Context` per request and passes it to the middleware
chain and the handler. `okapi.Ctx` and `okapi.C` are aliases of `*okapi.Context`,
so `func(c okapi.Ctx) error` is a valid handler. `c.Context()` returns the
request's `context.Context`, which is cancelled when the client goes away:

```go
o.Get("/slow", func(c okapi.Ctx) error {
    select {
    case <-time.After(5 * time.Second):
        return c.String(http.StatusOK, "done")
    case <-c.Context().Done():
        return c.Context().Err() // the client went away
    }
})
```

`okapi.NewContext(o, w, r)` builds a `*Context` outside the router, for example
to reuse Okapi helpers in a plain `http.Handler`. Pass your `*Okapi` so that the
application settings (body cap, trusted proxies, logger) apply; path parameters
are only available on requests routed by Okapi.

```go
o := okapi.New()
http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
    c := okapi.NewContext(o, w, r)
    _ = c.String(http.StatusOK, "pong")
})
```

`(*Okapi).SetContext` replaces the application-level context returned by the
deprecated `(*Okapi).GetContext`. That context is shared by every request, so
values stored on it are visible process-wide; use the `*Context` passed to the
handler instead.
