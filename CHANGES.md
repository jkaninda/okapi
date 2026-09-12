# Changes

## Unreleased

### Security

- **CORS no longer pairs the `"*"` wildcard with credentials.** `writeHeaders` echoed the
  request's `Origin` verbatim alongside `Access-Control-Allow-Credentials: true` whenever
  the allow-list contained `"*"`, which is exactly the header pair the same-origin policy
  exists to prevent — any site could read authenticated responses on a visitor's behalf.
  `Origin: null` was reflected too. A request matched by the bare wildcard now receives
  the literal `"*"` and no credentials header, and `null` never matches. Named and pattern
  origins are unaffected. `Vary: Origin` is now also set when the origin is refused, so a
  shared cache cannot serve a header-less response to an allowed origin.
- **JWKS key sets are cached, the fetch is bounded, and key material is validated.**
  The key set was refetched on every token validation, so unauthenticated traffic drove
  one outbound request to the identity provider per inbound request. The fetch used the
  default `http.Client` (no timeout), never checked the response status, and read the body
  with no size limit. Key sets are now cached per URL with a TTL (`JWTAuth.JwksCacheTTL`,
  15 minutes by default) with rate-limited refreshes for unknown `kid`s and failing
  endpoints, and fetches use a dedicated client with a 5s timeout, require HTTP 200, and
  read at most 1 MiB. RSA moduli under 2048 bits, degenerate exponents, EC points that are
  not on their named curve, and keys whose `use`/`alg` is inconsistent with signature
  verification are now rejected.
- **`Context.ServeFileFromFS` no longer serves a file it just rejected.** The traversal
  guard wrote a 404 but had no `return` after it, so the file body was appended to the 404
  response. The guard now returns and matches `".."` as a path element rather than as a
  substring. `ServeFile`, `ServeFileAttachment` and `ServeFileInline` have no guard at all
  by nature — `http.ServeFile`'s `".."` precaution inspects only `r.URL.Path` — so the new
  `Context.ServeFileFrom(root, name)` provides a confined alternative for names derived
  from a request.
- **The claims-expression cache no longer races.** `ClaimsExpression` was compiled into a
  field on the shared `*JWTAuth` with an unsynchronised check-then-write, a data race
  inside an authorization decision. Compiled expressions now live in a package-level cache.
- **The default `http.Server` has timeouts.** Every timeout was zero, meaning "no limit",
  so a Slowloris client held a connection and its goroutine indefinitely.
  `ReadHeaderTimeout` now defaults to 10s and `IdleTimeout` to 120s, and
  `WithReadHeaderTimeout` is exposed. `ReadTimeout` and `WriteTimeout` remain unset so
  uploads and streaming responses are not cut off.
- **SSE `id` and `event` values are stripped of newlines.** Both were written raw, so an
  application deriving them from user input — a reflected `Last-Event-ID`, say — let an
  attacker forge additional fields or whole events in the stream.
- **The request binders apply a default size cap.** `BindJSON`, `BindXML`, `BindYAML` and
  `BindProtoBuf` read the body with no ceiling, and `BodyLimit` is opt-in. They now cap at
  8 MB by default, adjustable with `WithMaxRequestBody`; `BodyLimit` still takes precedence
  where installed.
- **The built-in template renderer now escapes HTML.** `okapi.Template` — the renderer
  behind `NewTemplate`, `NewTemplateFromFiles`, `NewTemplateFromDirectory`,
  `NewTemplateWithConfig` and `c.Render()` — was built on `text/template`, which performs
  no escaping. Any application rendering user-controlled data through the documented
  templating path had a reflected XSS hole, while `Context.HTML` and `Context.HTMLView`
  escaped correctly via `html/template`. The renderer now uses `html/template` and its
  contextual auto-escaping.
- **JWT tokens must now carry an `exp` claim.** `golang-jwt` validates `exp` only when it
  is present, so a correctly signed token that omitted it passed validation permanently,
  with no revocation path. `jwt.WithExpirationRequired()` is now passed by default, for
  both the middleware and `ValidateToken`. Issuers that deliberately mint non-expiring
  tokens can set the new `JWTAuth.AllowMissingExpiry`.

### Features

- **`WithTrustedProxies(cidrs ...string)`** restricts the `X-Forwarded-For` and
  `X-Real-IP` headers that `RealIP()` reads to connections originating from the given CIDR
  blocks. Any client can set those headers, so `RealIP()` is spoofable and must not be used
  for rate limiting, allow-lists or audit trails until this is configured. The default is
  unchanged, so existing proxy deployments keep working.
- **`WithReadHeaderTimeout(seconds int)`** and **`WithMaxRequestBody(bytes int64)`** expose
  the two new defaults described above.
- **`Substring(field, val...)`** is a new claims-expression function, carrying the
  substring behaviour `Contains` used to have for a single value.

### Fixes

- **`WithCors` now installs the CORS handler, not just the preflight route.** Actual
  responses previously carried no `Access-Control-Allow-Origin`, so the browser blocked
  them after a preflight that had just said yes — headers appeared only if the user
  separately installed `Cors.CORSHandler` as middleware, which the documentation does not
  mention.
- **`Okapi.GetContext()` is documented and deprecated.** Its comment claimed it "returns
  the current context", but it hands back the single application-level `Context`: an empty
  request and a process-wide store shared concurrently by every goroutine. Since that store
  is where forwarded JWT claims land, mistaking it for the request `Context` leaks identity
  between users.
- **`DefaultErrorHandler`'s disclosure of internal error text is now documented.** It puts
  `err.Error()` into the response's `details` field, which for a JWKS-backed setup can
  include the JWKS URL and the underlying network error. Behaviour is unchanged; install a
  custom `ErrorHandler` to suppress it.
- **Leaving `JWTAuth.Audience` or `Issuer` unset no longer rejects every token.**
  `jwt.WithAudience` and `jwt.WithIssuer` are variadic, so passing an empty string
  registered `""` as the expected value and made the claim mandatory rather than
  unchecked. Both options are now only applied when the field is set — which is what
  their "Optional" documentation always claimed, and what the documented HS256 starter
  example needs in order to work at all.

### Breaking Changes

- Templates rendered through `okapi.Template` are now escaped. Templates that
  deliberately emit markup must pass `template.HTML` (or `template.JS` / `template.URL`)
  values, which is the explicit opt-in `html/template` expects.
- Tokens with no `exp` claim are now rejected unless `JWTAuth.AllowMissingExpiry` is set.
- Applications that relied on the unset-`Audience` behaviour to reject all traffic will
  now accept correctly signed tokens. Set `Audience` and `Issuer` explicitly if you
  require those claims.
- **`Contains` with a single value no longer performs substring matching.** It set its
  array-membership flag only when given more than one value, so `Contains(`role`, `admin`)`
  was satisfied by `role="not-admin-at-all"`, `"badmin"` or `"administrator-readonly"` —
  a privilege-escalation path wherever any part of a role or scope is user-influenced. It
  now matches by equality at every arity. Expressions that relied on the substring
  behaviour must switch to the new `Substring`.
- **`AllowedOrigins: ["*"]` with `AllowCredentials: true` no longer sends credentials.**
  Configurations that need credentials must name their origins, exactly or by pattern.
- **RSA keys under 2048 bits from a JWKS are rejected.** Deployments whose identity
  provider still publishes 1024-bit keys will fail to validate tokens until it is updated.
- **Request bodies over 8 MB are rejected by the binders** unless `WithMaxRequestBody` or
  `BodyLimit` raises the ceiling.
- **`http.Server` now ships with a 10s `ReadHeaderTimeout` and a 120s `IdleTimeout`.**
  Set them to zero explicitly to restore unlimited behaviour.

## v0.10.0

### Fixes

- **`r.PathValue` now works inside standard handlers.** Okapi's router captures path
  parameters into the request context, so `r.PathValue` — which reads what
  `http.ServeMux` registered — previously returned an empty string, and a handler moved
  over from `net/http` silently saw no parameters. `wrapHTTPHandler` now copies the
  captured parameters onto the request, for `HandleStd` and `HandleHTTP` on both the
  application and groups. `njia.Param` continues to work and remains the allocation-free
  accessor. The bridge costs one map allocation per request and is paid only by routes
  that use a standard handler and declare parameters; parameterless routes skip it and
  native handlers are unaffected.
- **`Group.HandleStd` now shares the single handler-conversion path.** It previously
  built its own adapter inline, which meant it would have missed the fix above and
  behaved differently from `Okapi.HandleStd`.

### Breaking Changes

- **The router now uses `github.com/jkaninda/njia/muxcompat` instead of the archived `github.com/gorilla/mux`.**
  `muxcompat` is a drop-in replacement, so routing, path variables, strict-slash redirects and
  `NotFound`/`MethodNotAllowed` handling behave as before and no application code needs to change.
  The one exception is the already-deprecated `WithMuxRouter` option: its parameter type is now
  `*muxcompat.Router`, so callers still passing a `*mux.Router` will no longer compile. The option is
  a no-op in spirit — Okapi manages its own router — and will be removed in a future release.

## v0.6.2

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