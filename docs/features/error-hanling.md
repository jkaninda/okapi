---
title: Error Handling
layout: default
parent: Features
nav_order: 10
---

# Error Handling

Okapi provides a flexible error handling system with built-in support for standard JSON errors, custom error formats, and RFC 7807 Problem Details.

## Quick Start

Use `c.Abort*` methods to immediately stop request processing and return an error response:

```go
o := okapi.Default()

o.Post("/books", func(c okapi.C) error {
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

## Available Error Methods

Okapi provides convenience methods for common HTTP error codes:

| Method                               | Status Code | Use Case                              |
|--------------------------------------|-------------|---------------------------------------|
| `AbortBadRequest(msg, err)`          | 400         | Validation errors, malformed requests |
| `AbortUnauthorized(msg, err)`        | 401         | Missing or invalid authentication     |
| `AbortForbidden(msg, err)`           | 403         | Insufficient permissions              |
| `AbortNotFound(msg, err)`            | 404         | Resource not found                    |
| `AbortConflict(msg, err)`            | 409         | Resource conflicts                    |
| `AbortInternalServerError(msg, err)` | 500         | Unexpected server errors              |
| And more                             |             |                                       |

For other status codes, use the generic method:

```go
return c.AbortWithError(http.StatusTeapot,  err)
```

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
    "message": "Invalid request body"
  }
}
```

## RFC 7807 Problem Details

For APIs requiring standards-compliant error responses, Okapi supports [RFC 7807 Problem Details](https://datatracker.ietf.org/doc/html/rfc7807).

### Basic Setup

```go
o := okapi.Default()
o.WithSimpleProblemDetailErrorHandler()
```

Response (`Content-Type: application/problem+json`):

```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "field Name is required",
  "instance": "/books"
}
```

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
  "type": "https://api.example.com/errors/bad-request",
  "title": "Bad Request",
  "status": 400,
  "detail": "field Name is required",
  "instance": "/books",
  "timestamp": "2026-02-09T21:42:49+01:00",
  "api_version": "v1.0.0",
  "support_url": "https://support.example.com"
}
```

### Configuration Options

| Option             | Description                                         |
|--------------------|-----------------------------------------------------|
| `Format`           | Response format (see below)                         |
| `TypePrefix`       | Base URL for error type URIs                        |
| `IncludeInstance`  | Include the request path in responses               |
| `IncludeTimestamp` | Add a timestamp to each error                       |
| `CustomFields`     | Additional fields to include in all error responses |

### Supported Formats

| Format                   | Content-Type               | Description                    |
|--------------------------|----------------------------|--------------------------------|
| `ErrorFormatProblemJSON` | `application/problem+json` | RFC 7807 JSON format (default) |
| `ErrorFormatProblemXML`  | `application/problem+xml`  | RFC 7807 XML format            |

Example with XML format:

```go
o.WithProblemDetailErrorHandler(&okapi.ErrorHandlerConfig{
    Format: okapi.ErrorFormatProblemXML,
})
```

Response (`Content-Type: application/problem+xml`):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<problem xmlns="urn:ietf:rfc:7807">
  <type>about:blank</type>
  <title>Bad Request</title>
  <status>400</status>
  <detail>field Name is required</detail>
  <instance>/books</instance>
</problem>
```
