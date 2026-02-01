---
title: Middlewares
layout: default
parent: Features
nav_order: 1
---

# Middleware

Middleware in Okapi allows you to intercept and process HTTP requests before they reach your route handlers. This is useful for authentication, logging, request validation, and more.

## Built-in Middleware

### Basic Authentication

```go
auth := okapi.BasicAuth{
    Username: "admin",
    Password: "password",
    Realm:    "Restricted",
    ContextKey: "user", // where to store the username (default: "username")
}

// Global middleware
o.Use(auth.Middleware)

// Route-specific middleware
o.Get("/admin", adminHandler).Use(auth.Middleware)
```

### CORS Middleware

```go
cors := okapi.Cors{
    AllowedOrigins: []string{"http://localhost:8080", "https://example.com"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
}

o := okapi.New(okapi.WithCors(cors))
```

## JWT Middleware

Okapi includes powerful JWT middleware to secure your routes with JSON Web Tokens.

### Features

* **HS256** symmetric signing via `SigningSecret`
* **RS256** and other asymmetric algorithms via `RSAKey`
* **Remote JWKS** discovery via `JwksUrl` (e.g., OIDC or Auth0)
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

### Remote JWKS (OIDC, Auth0)

```go
jwtAuth := okapi.JWTAuth{
    JwksUrl:     "https://example.com/.well-known/jwks.json",
    TokenLookup: "header:Authorization",
    ContextKey:  "user",
}
```

### Claims Expression

Use `ClaimsExpression` to validate claims using simple expressions:

#### Supported Functions

* `Equals(field, value)`
* `Prefix(field, prefix)`
* `Contains(field, val1, val2, ...)`
* `OneOf(field, val1, val2, ...)`

#### Logical Operators

* `!` — NOT
* `&&` — AND
* `||` — OR

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret:    []byte("supersecret"),
    ClaimsExpression: "Equals(`email_verified`, `true`) && Equals(`user.role`, `admin`)",
    TokenLookup:      "header:Authorization",
    ContextKey:       "user",
}
```

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
        return c.AbortUnauthorized("Unauthorized", fmt.Errorf("user not authenticated"))
    }
    
    return c.JSON(http.StatusOK, okapi.M{
        "email": email,
        "role":  c.GetString("role"),
        "name":  c.GetString("name"),
    })
}
```

### Custom Claim Validation

```go
jwtAuth.ValidateClaims = func(c *Context, claims jwt.Claims) error {
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
    SigningSecret: SigningSecret,
    OnUnauthorized: func(c *okapi.Context) error {
        return c.ErrorUnauthorized("Unauthorized")
    },
}
```

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

## Custom Middleware

Create your own middleware functions:

```go
func customMiddleware(next okapi.HandlerFunc) okapi.HandlerFunc {
    return func(c *okapi.Context) error {
        start := time.Now()
        err := next(c)
        log.Printf("Request took %v", time.Since(start))
        return err
    }
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

## Middleware Chaining

Apply multiple middleware to a route or group:

```go
o.Get("/admin", 
    adminHandler,
).Use(
    authMiddleware,
    loggingMiddleware,
    rateLimitMiddleware,
)
```
