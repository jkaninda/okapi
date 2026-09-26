---
title: Error Handling
sidebar_position: 11
---

# Error Handling

Okapi provides a flexible error handling system with built-in support for standard JSON errors, custom error formats, and RFC 7807 Problem Details.

:::warning Note on error details in production

`DefaultErrorHandler` puts the underlying error's text into the `details`
field of the response, and middleware such as the JWT authenticator passes
its internal error through. Clients therefore learn exactly why a request
failed — with a JWKS-backed setup that can include the JWKS URL and the
network error behind it, which is useful reconnaissance against an internal
endpoint.

If that matters for your deployment, install a custom `ErrorHandler` that
logs the error server-side and omits `details` from the response. See
[Custom Error Handlers](#custom-error-handlers) below.

:::

## Return an Error, or Abort?

An `error` returned from a handler or middleware is **not** shown to the client.
It is logged, and the configured `ErrorHandler` answers with a generic `500` —
because an error's text routinely carries internals such as driver messages,
hostnames or file paths. If something has already been written to the response,
the error is logged and nothing further is sent.

So return an error only when a generic `500` is the right answer. Everywhere else,
respond with an `Abort*` helper, which lets you choose the status and the message.

## Quick Start

Use `c.Abort*` methods to immediately stop request processing and return an error response:

```go
o := okapi.Default()

o.Post("/books", func(c *okapi.Context) error {
    book := &Book{}
    if err := c.Bind(book); err != nil {
        return c.AbortBadRequest("Invalid request body", err)
    }
    // ... handle valid request
    return c.Created(book)
})
```

Response:

```json
{
  "code": 400,
  "message": "Invalid request body",
  "details": "field Name is required",
  "timestamp": "2026-02-09T21:34:17.290908+01:00"
}
```

The `err` argument is optional. When omitted, the message is used as the
underlying error, so `details` repeats the message.

## Available Error Methods

Okapi provides convenience methods for common HTTP error codes:

| Method                               | Status Code | Use Case                              |
|--------------------------------------|-------------|---------------------------------------|
| `AbortBadRequest(msg, err...)`       | 400         | Validation errors, malformed requests |
| `AbortUnauthorized(msg, err...)`     | 401         | Missing or invalid authentication     |
| `AbortForbidden(msg, err...)`        | 403         | Insufficient permissions              |
| `AbortNotFound(msg, err...)`         | 404         | Resource not found                    |
| `AbortConflict(msg, err...)`         | 409         | Resource conflicts                    |
| `AbortValidationError(msg, err...)`  | 422         | Unprocessable entity                  |
| `AbortTooManyRequests(msg, err...)`  | 429         | Rate limiting                          |
| `AbortInternalServerError(msg, err...)` | 500      | Unexpected server errors              |
| And more                             |             |                                       |

There is an `Abort*` helper for every standard 4xx and 5xx status, and an
`Error*` counterpart (`ErrorBadRequest`, `ErrorNotFound`, …) that writes the value
you pass as the body, bypassing the `ErrorHandler`.

For other status codes, use the generic method — the message is the status text:

```go
return c.AbortWithError(http.StatusTeapot, err)
```

### Field-Level Validation Errors

`AbortValidationErrors` answers `422` with a list of per-field failures. It always
uses Okapi's `ValidationErrorResponse` shape, regardless of the configured
`ErrorHandler`:

```go
return c.AbortValidationErrors([]okapi.ValidationError{
    {Field: "email", Message: "must be a valid email", Value: in.Email},
    {Field: "age", Message: "must be greater than 0"},
})
```

`AbortValidationErrorsWithProblemDetail` emits the same information as
`application/problem+json`.

## Custom Error Handlers

Override the default error format by providing a custom error handler:

```go
o := okapi.Default().With(
    okapi.WithErrorHandler(func(c *okapi.Context, code int, message string, err error) error {
        return c.JSON(code, map[string]any{
            "success": false,
            "error": map[string]any{
                "code":    code,
                "message": message,
                "details": err.Error(),
            },
        })
    }),
)
```

Response:

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "Invalid request body",
    "details": "field Name is required"
  }
}
```

To stop leaking internals, log `err` and leave it out of the response:

```go
okapi.WithErrorHandler(func(c *okapi.Context, code int, message string, err error) error {
    if err != nil {
        c.Logger().Error("request failed", "code", code, "error", err)
    }
    return c.JSON(code, map[string]any{"code": code, "message": message})
})
```

`err` can be `nil` — notably for the generic `500` produced when a handler returns
an error — so guard against it before calling `err.Error()`.

A handler can also be installed for a single request with
`c.SetErrorHandler(handler)`, which takes precedence over the application-wide one.

## RFC 7807 Problem Details

For APIs requiring standards-compliant error responses, Okapi supports [RFC 7807 Problem Details](https://datatracker.ietf.org/doc/html/rfc7807).

### Basic Setup

```go
o := okapi.Default()
o.WithSimpleProblemDetailErrorHandler()
```

This is `WithProblemDetailErrorHandler(nil)`, whose defaults are
`Format: ErrorFormatProblemJSON`, `TypePrefix: "about:blank"`,
`IncludeInstance: true` and `IncludeTimestamp: true`.

Response (`Content-Type: application/problem+json`) for
`c.AbortBadRequest("Invalid request body", err)`:

```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid request body",
  "instance": "/books",
  "timestamp": "2026-02-09T21:42:49+01:00"
}
```

`detail` is the **message** you passed to the `Abort*` helper. The underlying
error's text is used only when no message was given — that is, when the message
would just repeat the status text.

### Advanced Configuration

Customize the Problem Details output with additional fields and options:

```go
o := okapi.Default()
o.WithProblemDetailErrorHandler(&okapi.ErrorHandlerConfig{
    Format:           okapi.ErrorFormatProblemJSON,
    TypePrefix:       "https://api.example.com/errors/",
    IncludeInstance:  true,
    IncludeTimestamp: true,
    CustomFields: map[string]any{
        "api_version": "v1.0.0",
        "support_url": "https://support.example.com",
    },
})
```

Response:

```json
{
  "type": "https://api.example.com/errors/",
  "title": "Bad Request",
  "status": 400,
  "detail": "Invalid request body",
  "instance": "/books",
  "timestamp": "2026-02-09T21:42:49+01:00",
  "api_version": "v1.0.0",
  "support_url": "https://support.example.com"
}
```

`TypePrefix` is written to `type` **as given** — no per-status slug is appended.
Point it at a single document describing your error format, or install a custom
`ErrorHandler` if you need one URI per problem type.

### Configuration Options

| Option             | Description                                         |
|--------------------|-----------------------------------------------------|
| `Format`           | Response format (see below)                         |
| `TypePrefix`       | The value written to the `type` member              |
| `IncludeInstance`  | Include the request path in responses               |
| `IncludeTimestamp` | Add a `timestamp` member to each error              |
| `CustomFields`     | Additional members to include in all error responses |

### Supported Formats

| Format                   | Content-Type               | Description                    |
|--------------------------|----------------------------|--------------------------------|
| `ErrorFormatProblemJSON` | `application/problem+json` | RFC 7807 JSON format (default) |
| `ErrorFormatProblemXML`  | `application/problem+xml`  | RFC 7807 XML format            |
| `ErrorFormatDefault`     | `application/json`         | Okapi's own `ErrorResponse` shape |

Example with XML format:

```go
o.WithProblemDetailErrorHandler(&okapi.ErrorHandlerConfig{
    Format:           okapi.ErrorFormatProblemXML,
    TypePrefix:       "about:blank",
    IncludeInstance:  true,
    IncludeTimestamp: true,
})
```

Response (`Content-Type: application/problem+xml`):

```xml
<ProblemDetail><type>about:blank</type><title>Bad Request</title><status>400</status><detail>Invalid request body</detail><instance>/books</instance></ProblemDetail>
```

Two things to know about the XML form: the document element is `ProblemDetail`
with no namespace and no XML declaration, and `timestamp` and `CustomFields` are
**not** emitted — they are carried as JSON extension members and have no XML
representation. Use the JSON format when you need them.

## Writing a Problem Detail Directly

`AbortWithProblemDetail` sends a `ProblemDetail` you built yourself, without going
through the configured `ErrorHandler`:

```go
return c.AbortWithProblemDetail(&okapi.ProblemDetail{
    Type:     "https://api.example.com/errors/out-of-stock",
    Title:    "Out of stock",
    Status:   http.StatusConflict,
    Detail:   "The book is temporarily unavailable",
    Instance: c.Path(),
})
```
