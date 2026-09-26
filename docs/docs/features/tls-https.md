---
title: TLS & HTTPS
sidebar_position: 8
---

# TLS & HTTPS

Okapi provides built-in support for serving your API over HTTPS with TLS (Transport Layer Security).

## Basic TLS Setup

`WithTLS` puts the TLS configuration on the server Okapi starts, so `Start()`
serves HTTPS on the instance's address (`:8080` unless you change it).

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/jkaninda/okapi"
)

func main() {
    // Initialize TLS configuration for secure HTTPS connections
    tlsConfig, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
    if err != nil {
        panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
    }

    // Create a new Okapi instance with TLS, listening on :8443
    o := okapi.New(
        okapi.WithTLS(tlsConfig),
        okapi.WithAddr(":8443"),
    )

    // Register routes
    o.Get("/", func(c *okapi.Context) error {
        return c.JSON(http.StatusOK, okapi.M{
            "message": "Welcome to Okapi!",
            "status":  "operational",
        })
    })

    // Start the HTTPS server
    log.Println("Starting HTTPS server on :8443")
    if err := o.Start(); err != nil {
        panic(fmt.Sprintf("Server failed to start: %v", err))
    }
}
```

## Dual HTTP and HTTPS Servers

You can run both HTTP and HTTPS servers simultaneously:

```go
package main

import (
    "fmt"
    "log"
    "net/http"

    "github.com/jkaninda/okapi"
)

func main() {
    // Initialize TLS configuration
    tlsConfig, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
    if err != nil {
        panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
    }

    // Create Okapi instance with default config (HTTP on :8080)
    o := okapi.Default()

    // Configure a secondary HTTPS server on port 8443
    o.With(okapi.WithTLSServer(":8443", tlsConfig))

    // Register routes (available on both HTTP and HTTPS)
    o.Get("/", func(c *okapi.Context) error {
        return c.JSON(http.StatusOK, okapi.M{
            "message": "Welcome to Okapi!",
            "status":  "operational",
        })
    })

    // Start both servers
    log.Println("Starting server on :8080 (HTTP) and :8443 (HTTPS)")
    if err := o.Start(); err != nil {
        panic(fmt.Sprintf("Server failed to start: %v", err))
    }
}
```

`WithTLSServer` panics on an invalid address, so pass it in the `":port"` or
`"host:port"` form. `WithTLS` and `WithTLSServer` are alternatives: use the first
to serve HTTPS on the main address, and the second to add a second listener
alongside plain HTTP.

## Server Timeouts

The server Okapi creates is not left with every timeout at zero: `ReadHeaderTimeout`
defaults to 10s and `IdleTimeout` to 120s, so a client dribbling headers cannot
hold a connection and its goroutine indefinitely. `ReadTimeout` and `WriteTimeout`
stay unset so uploads and streaming responses are not cut off.

Adjust them with `WithReadHeaderTimeout`, `WithIdleTimeout`, `WithReadTimeout` and
`WithWriteTimeout`. A server you supply yourself with `WithServer` keeps its own
timeouts — Okapi only fills the fields that are still zero.

## TLS Configuration Options

The `LoadTLSConfig` function accepts the following parameters:

```go
func LoadTLSConfig(certFile, keyFile, caFile string, clientAuth bool) (*tls.Config, error)
```

### Parameters

- **certFile**: Path to the TLS certificate file (PEM format)
- **keyFile**: Path to the TLS private key file (PEM format)
- **caFile**: Optional path to CA certificate for client authentication
- **clientAuth**: Whether to require client certificate authentication

The returned configuration sets `MinVersion` to TLS 1.2.

## Generating Self-Signed Certificates

For development purposes, you can generate self-signed certificates:

```bash
# Generate private key
openssl genrsa -out server.key 2048

# Generate certificate
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365
```

## Let's Encrypt (Production)

For production environments, use Let's Encrypt for free, automated SSL certificates.
`autocert` produces a `*tls.Config`, which is exactly what `WithTLS` takes — Okapi
has no exported `http.Server` field to reach into:

```go
package main

import (
    "log"

    "github.com/jkaninda/okapi"
    "golang.org/x/crypto/acme/autocert"
)

func main() {
    certManager := autocert.Manager{
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
        Cache:      autocert.DirCache("certs"),
    }

    o := okapi.New(
        okapi.WithTLS(certManager.TLSConfig()),
        okapi.WithAddr(":443"),
    )

    o.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{"message": "Secure connection!"})
    })

    // Serves HTTPS on :443
    log.Fatal(o.Start())
}
```

`golang.org/x/crypto` is not a dependency of Okapi; add it to your own module.

## Security Best Practices

1. **Always use TLS 1.2 or higher** in production
2. **Use strong cipher suites** and disable weak ones
3. **Keep certificates up to date** and monitor expiration
4. **Use HSTS headers** to enforce HTTPS
5. **Redirect HTTP to HTTPS** for better security

### Example: HTTP to HTTPS Redirect

`c.Redirect` writes the response itself and returns nothing, so a middleware
returns `nil` after redirecting and `c.Next()` otherwise:

```go
func redirectToHTTPS(c *okapi.Context) error {
    if c.Request().TLS == nil {
        httpsURL := "https://" + c.Request().Host + c.Request().RequestURI
        c.Redirect(http.StatusMovedPermanently, httpsURL)
        return nil
    }
    return c.Next()
}

// Apply to all routes
o.Use(redirectToHTTPS)
```

Returning `nil` without calling `c.Next()` is what stops the chain — forget it and
every request would be redirected *and* handled.

### Example: Adding HSTS Header

```go
func hstsMiddleware(c *okapi.Context) error {
    c.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
    return c.Next()
}

o.Use(hstsMiddleware)
```

## Graceful Shutdown

`Stop()` / `StopWithContext(ctx)` drain in-flight requests before cancelling their
contexts, so a handler using `c.Request().Context()` — a database query, an
outbound call — is not cancelled the moment a deploy starts. Contexts are
cancelled once draining finishes or the shutdown context expires. Active SSE
streams end as soon as shutdown begins. `Start` and `Stop` are safe to call from
different goroutines.
