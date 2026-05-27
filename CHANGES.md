# Changes

## Unreleased

### Fixes

- **Response writers are now idempotent.** Once a response is committed (e.g., by an `Abort*` call),
  subsequent calls to `c.JSON`, `c.OK`, `c.XML`, `c.Text`, `c.Render`, `c.Data`, `c.Error`,
  `c.AbortNotModified`, etc. are silent no-ops instead of appending a second body to the wire.
  This fixes the double-response bug where a helper called `c.AbortBadRequest(...)` without
  propagating its return value and the caller then wrote a success body, producing two concatenated
  JSON objects in the response. Skipped writes emit a `Debug`-level log to aid troubleshooting.

## v0.6.0

### Breaking Changes

- **OpenAPI 3.1 is now the default.** The default endpoints `/openapi.json` and `/openapi.yaml` (and the
  `/docs`, `/swagger`, `/redoc`, `/scalar` UIs) now serve OpenAPI **3.1** instead of 3.0. The 3.0 document
  is preserved at the version-pinned routes `/openapi-3.0.json` and `/openapi-3.0.yaml`. Consumers or tooling
  that require 3.0 should point at those routes.

### Features

- **Scalar API Reference UI**: a third built-in documentation UI alongside Swagger UI and ReDoc, served at
  `/scalar`.
- **Selectable `/docs` UI**: choose which UI is rendered at `/docs` via the `UI` field on `okapi.OpenAPI`
  (`okapi.SwaggerUI` (default), `okapi.RedocUI`, `okapi.ScalarUI`) or the chainable `WithDocUI(...)` method.
  Each UI also stays reachable at its own route (`/swagger`, `/redoc`, `/scalar`) regardless of the selection.
- **OpenAPI 3.1 support**: Okapi serves both an OpenAPI 3.1 / JSON Schema 2020-12 document and the original
  OpenAPI 3.0 document. The 3.1 spec is the default (`/openapi.json`, `/openapi.yaml`); the 3.0 spec is
  available at `/openapi-3.0.json` / `/openapi-3.0.yaml`. The 3.1 document is derived from the 3.0 base and adds:
    - type-array nullability (`type: ["string", "null"]`) for pointer fields (rendered as `nullable: true` in 3.0),
    - `jsonSchemaDialect`,
    - SPDX `License.Identifier` (new field on `okapi.License`),
    - `const` via a new `const:"value"` struct tag,
    - webhooks via the new `(*Okapi).Webhook(name, method, ...Doc options)` API.

## v0.5.0

### Breaking Changes

- **Middleware signature changed**: `Middleware` type changed from `func(next HandlerFunc) HandlerFunc` to `func(*Context) error` (type alias for `HandlerFunc`). Middleware now calls `c.Next()` to pass control to the next handler instead of receiving `next` as a parameter.

  **Before:**

  ```go
  func RequestID() Middleware {
      return func(next HandlerFunc) HandlerFunc {
          return func(c *Context) error {
              id := c.Header(requestIDHeader)
              if id == "" {
                  id = uuid.New().String()
              }
              c.Set("request_id", id)
              c.Response().Header().Set(requestIDHeader, id)
              return next(c)
          }
      }
  }
  ```

  **After:**

  ```go
  func RequestID() Middleware {
      return func(c *Context) error {
          id := c.Header(requestIDHeader)
          if id == "" {
              id = uuid.New().String()
          }
          c.Set("request_id", id)
          c.Response().Header().Set(requestIDHeader, id)
          return c.Next()
      }
  }
  ```

- **`MiddlewareFunc`** is now also a type alias for `HandlerFunc`.

### New Features

- **`Context.Next()`**: New method on `Context` that executes the next handler in the middleware chain. This replaces the `next(c)` call pattern and simplifies middleware authoring.

### Migration Guide

To migrate existing middleware from v0.4.x to v0.5.0:

1. Remove the outer `func(next HandlerFunc) HandlerFunc` wrapper
2. Remove the inner `return func(c *Context) error` wrapper (flatten to a single function)
3. Replace all `next(c)` calls with `c.Next()`
4. Replace all `return next(c)` with `return c.Next()`

**Custom middleware before:**

```go
func myMiddleware(next okapi.HandlerFunc) okapi.HandlerFunc {
    return func(c *okapi.Context) error {
        // pre-processing
        err := next(c)
        // post-processing
        return err
    }
}
```

**Custom middleware after:**

```go
func myMiddleware(c *okapi.Context) error {
    // pre-processing
    err := c.Next()
    // post-processing
    return err
}
```

### Internal Changes

- Middleware chain execution replaced from function wrapping to slice-based handler chain (`buildHandlers()`)
- Updated all built-in middleware: `LoggerMiddleware`, `BasicAuth`, `BodyLimit`, `JWTAuth`, `RequestID`, `CORSHandler`, `handleAccessLog`
- Updated `UseMiddleware` adapters (standard `http.Handler` middleware compatibility) in both `Okapi` and `Group`
- Updated group middleware application in `add()`, `HandleStd()`, `HandleHTTP()`
