# Changes

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
