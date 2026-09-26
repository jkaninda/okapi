---
title: Middlewares
sidebar_position: 2
---

# Middleware

Middleware in Okapi allows you to intercept and process HTTP requests before they reach your route handlers. This is useful for authentication, logging, request validation, and more.

## Built-in Middleware

### Basic Authentication

```go
auth := okapi.BasicAuth{
    Username:   "admin",
    Password:   "password",
    Realm:      "Restricted",
    ContextKey: "user", // where to store the username (default: "username")
}

// Global middleware
o.Use(auth.Middleware)

// Route-specific middleware
o.Get("/admin", adminHandler).Use(auth.Middleware)
```

An empty `Username` or `Password` is treated as a misconfiguration, not as "no
credentials required": every request is rejected with `401` and the problem is
logged. Load the credentials before installing the middleware.

### Request Body Limit

`BodyLimit` rejects an oversized body with `413 Request Entity Too Large`:

```go
o.Use(okapi.BodyLimit{MaxBytes: 2 << 20}.Middleware) // 2 MB
```

Without it, the binders still apply a default ceiling of 8 MB, adjustable with
`okapi.WithMaxRequestBody`. Where `BodyLimit` is installed, its limit wins,
because it has already bounded the body by the time a binder runs.

### Request ID

```go
o.Use(okapi.RequestID())
```

It reuses an incoming `X-Request-Id`, or generates one, stores it under
`request_id` in the context, and sets the response header.

### CORS Middleware

```go
cors := okapi.Cors{
    AllowedOrigins: []string{"http://localhost:8080", "https://example.com"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
}

o := okapi.New(okapi.WithCors(cors))
```

`WithCors` installs both the per-path `OPTIONS` preflight handler **and** the CORS
handler in the middleware chain, so actual responses carry
`Access-Control-Allow-Origin` too. You do not need to add `Cors.CORSHandler` as
middleware yourself.

A bare `"*"` in `AllowedOrigins` and `AllowCredentials: true` cannot be combined
safely — that header pair is exactly what the same-origin policy exists to
prevent. Okapi degrades it instead of silently honouring it: a request matched by
the bare wildcard receives the literal `*` and no
`Access-Control-Allow-Credentials`. Name the origins, or use a pattern such as
`https://*.example.com`, when you need credentials. The `null` origin never
matches.

## JWT Middleware

Okapi includes powerful JWT middleware to secure your routes with JSON Web Tokens.

### Features

* **HS256** symmetric signing via `SigningSecret`
* **RS256** and other asymmetric algorithms via `RsaKey`
* **Remote JWKS** discovery via `JwksUrl` (e.g., OIDC or Auth0), cached per URL
* **Local JWKS** via `JwksFile`
* **Claims validation** with `ClaimsExpression` or `ValidateClaims`
* **OpenAPI integration** with `.WithBearerAuth()`
* **Selective claim forwarding** using `ForwardClaims`

### Basic HS256 Authentication

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret: []byte("supersecret"),      // Shared secret for HS256
    TokenLookup:   "header:Authorization",     // Token source
    ContextKey:    "user",                     // Key for storing claims in context
}
```

`TokenLookup` may list several sources separated by commas — for example
`"header:Authorization,query:token,cookie:jwt"`. They are tried in order and the
first non-empty token wins. The default is `"header:Authorization"`.

A **zero-length** `SigningSecret` counts as unset. `[]byte(os.Getenv("JWT_SECRET"))`
with the variable unset therefore does not configure HMAC with an empty key —
which anyone could forge — it leaves the middleware with no key at all, and every
request is rejected with `401`. `OnUnauthorized`, if you set one, runs for that
case too, exactly as for any other rejected token.

### Claim Validation Defaults

`Audience` and `Issuer` are optional: when either is left empty the
corresponding claim is not checked, and a token is accepted whether or not it
carries one. Set them to require a specific value.

An `exp` (expiry) claim is **required** by default. A signed token without one
never expires and the middleware has no revocation path, so any such token that
leaks stays valid until the signing key is rotated. If your issuer deliberately
mints non-expiring tokens, opt in explicitly:

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret:      []byte("supersecret"),
    AllowMissingExpiry: true, // accept tokens with no "exp" claim
}
```

`JWTAuth.ValidateToken` applies exactly the same checks as `Middleware` — key
resolution, the algorithm allow-list, expiry, `Audience`, `Issuer`,
`ClaimsExpression` and `ValidateClaims` — but writes no response and does not call
`OnUnauthorized`.

### Remote JWKS (OIDC, Auth0)

```go
jwtAuth := okapi.JWTAuth{
    JwksUrl:      "https://example.com/.well-known/jwks.json",
    JwksCacheTTL: 15 * time.Minute, // optional; 15 minutes is the default
    TokenLookup:  "header:Authorization",
    ContextKey:   "user",
}
```

The key set is cached per URL, because the key function runs on every token
validation: fetching per request would let unauthenticated traffic drive one
outbound request to your identity provider per inbound request. A token whose
`kid` is missing from the cached set triggers a rate-limited refresh, so key
rotation is picked up without waiting out the TTL, and refreshes never storm a
failing endpoint: they are rate-limited, run one at a time, and an expired set is
served for up to 15 minutes while the endpoint is unavailable.

Fetches use a dedicated client with a 5s timeout, require HTTP 200 and read at
most 1 MiB. Key material is validated: RSA moduli under 2048 bits, degenerate
exponents, EC points that are not on their named curve, and keys whose `use`/`alg`
is inconsistent with signature verification are rejected.

### Generating Tokens

`okapi.GenerateJwtToken(secret, claims, ttl)` signs an HS256 token, adding `exp`
and `iat`. It returns an error for an empty secret rather than signing a token
anyone could forge, and accepts `nil` claims.

```go
token, err := okapi.GenerateJwtToken([]byte("supersecret"), jwt.MapClaims{
    "sub":  "42",
    "role": "admin",
}, time.Hour)
```

### Claims Expression

Use `ClaimsExpression` to validate claims using simple expressions:

#### Supported Functions

* `Equals(field, value)`
* `Prefix(field, prefix)`
* `Contains(field, val1, val2, ...)` — the claim equals one of the values, or, for an array claim, has an element equal to one of them
* `OneOf(field, val1, val2, ...)`
* `Substring(field, val1, val2, ...)` — the claim contains one of the values as a substring

`Substring` carries the behaviour `Contains` used to have for a single value. It
is a deliberately weak test and is not suitable for authorization
decisions: wherever any part of a role, scope or tenant string is
user-influenced, an attacker can make it contain the value you are checking
for. Prefer `Equals`, `OneOf` or `Contains`.

#### Logical Operators

* `!` — NOT
* `&&` — AND (evaluated before OR)
* `||` — OR

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret:    []byte("supersecret"),
    ClaimsExpression: "Equals(`email_verified`, `true`) && Equals(`user.role`, `admin`)",
    TokenLookup:      "header:Authorization",
    ContextKey:       "user",
}
```

An expression that does not parse — including one with trailing input that
Okapi cannot make sense of — **denies** the request. Every part of the expression
is evaluated, so `Equals(...) && OneOf(...)` really checks both halves.

### Forwarding Claims to Context

```go
jwtAuth.ForwardClaims = map[string]string{
    "email": "user.email",
    "role":  "user.role",
    "name":  "user.name",
}
```

Access claims in your handler:

```go
func whoAmIHandler(c *okapi.Context) error {
    email := c.GetString("email")
    if email == "" {
        return c.AbortUnauthorized("Unauthorized")
    }

    return c.JSON(http.StatusOK, okapi.M{
        "email": email,
        "role":  c.GetString("role"),
        "name":  c.GetString("name"),
    })
}
```

Read them from the `*okapi.Context` the handler was given. `Okapi.GetContext()`
returns the single application-level `Context`, whose store is shared by every
goroutine — forwarded claims land there too, so using it in place of the request
context leaks identity between users. It is deprecated for that reason.

### Custom Claim Validation

```go
jwtAuth.ValidateClaims = func(c *okapi.Context, claims jwt.Claims) error {
    mapClaims, ok := claims.(jwt.MapClaims)
    if !ok {
        return errors.New("invalid claims type")
    }

    if emailVerified, _ := mapClaims["email_verified"].(bool); !emailVerified {
        return errors.New("email not verified")
    }

    if role, _ := mapClaims["role"].(string); role != "admin" {
        return errors.New("unauthorized role")
    }

    return nil
}
```

### Custom Error Handling

```go
auth := okapi.JWTAuth{
    Audience:      "okapi.example.com",
    SigningSecret: signingSecret,
    OnUnauthorized: func(c *okapi.Context) error {
        return c.ErrorUnauthorized("Unauthorized")
    },
}
```

The middleware no longer puts token-parsing, claims-expression or JWKS network
errors into the response: they are logged, and the client gets a generic message.
The token itself is never logged.

### Protecting Routes

```go
// Apply middleware globally
o.Use(jwtAuth.Middleware)

// Protect specific group
admin := o.Group("/admin", jwtAuth.Middleware).
    WithBearerAuth() // Adds Bearer auth to OpenAPI docs

admin.Get("/users", adminGetUsersHandler)

// Route-specific middleware
o.Get("/protected", protectedHandler).Use(jwtAuth.Middleware)
```

Group middlewares are resolved when a request is dispatched, walking through every
parent group, so a group's authentication covers every route in it and in its
sub-groups — including routes registered before `Use` was called, and routes
registered through `Group.Register`.

## Custom Middleware

Create your own middleware functions. Call `c.Next()` to pass control to the next middleware or handler:

```go
func customMiddleware(c *okapi.Context) error {
    start := time.Now()
    err := c.Next()
    log.Printf("Request took %v", time.Since(start))
    return err
}

o.Use(customMiddleware)
```

## Standard Library Middleware

You can also use standard `http.Handler` middleware:

```go
o.UseMiddleware(func(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        slog.Info("Hello from standard HTTP middleware")
        handler.ServeHTTP(w, r)
    })
})
```

The adapter forwards the `*http.Request`, so middleware that reads, decorates or
replaces the request works. It **cannot** replace the `http.ResponseWriter`: the
rest of the Okapi chain keeps writing to Okapi's own writer, so a writer your
middleware wraps around `w` never sees the response. Compression, response
recording and metrics middleware written that way will appear to install
correctly and then observe nothing.

## Middleware Chaining

Apply multiple middleware to a route or group:

```go
o.Get("/admin", adminHandler).Use(
    authMiddleware,
    loggingMiddleware,
    rateLimitMiddleware,
)
```

Middleware runs outermost-first: application-level (`Okapi.Use`), then group
middlewares from the outermost group inwards, then route middlewares, then the
handler.
