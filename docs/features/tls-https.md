---
title: TLS & HTTPS
layout: default
parent: Features
nav_order: 7
---

# TLS & HTTPS

Okapi provides built-in support for serving your API over HTTPS with TLS (Transport Layer Security).

## Basic TLS Setup

```go
package main

import (
    "fmt"
    "log"
    "github.com/jkaninda/okapi"
)

func main() {
    // Initialize TLS configuration for secure HTTPS connections
    tls, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
    if err != nil {
        panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
    }
    
    // Create a new Okapi instance with TLS
    o := okapi.New(okapi.WithTLS(tls))
    
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
func main() {
    // Initialize TLS configuration
    tls, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
    if err != nil {
        panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
    }
    
    // Create Okapi instance with default config (HTTP on :8080)
    o := okapi.Default()
    
    // Configure a secondary HTTPS server on port 8443
    o.With(okapi.WithTLSServer(":8443", tls))
    
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

## Generating Self-Signed Certificates

For development purposes, you can generate self-signed certificates:

```bash
# Generate private key
openssl genrsa -out server.key 2048

# Generate certificate
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 365
```

## Let's Encrypt (Production)

For production environments, use Let's Encrypt for free, automated SSL certificates:

```go
import (
    "golang.org/x/crypto/acme/autocert"
)

func main() {
    certManager := autocert.Manager{
        Prompt:     autocert.AcceptTOS,
        HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
        Cache:      autocert.DirCache("certs"),
    }
    
    o := okapi.New()
    
    // Configure TLS with autocert
    o.Server.TLSConfig = certManager.TLSConfig()
    
    o.Get("/", func(c *okapi.Context) error {
        return c.OK(okapi.M{"message": "Secure connection!"})
    })
    
    // Start HTTPS server
    log.Fatal(o.Server.ListenAndServeTLS("", ""))
}
```

## Security Best Practices

1. **Always use TLS 1.2 or higher** in production
2. **Use strong cipher suites** and disable weak ones
3. **Keep certificates up to date** and monitor expiration
4. **Use HSTS headers** to enforce HTTPS
5. **Redirect HTTP to HTTPS** for better security

### Example: HTTP to HTTPS Redirect

```go
func redirectToHTTPS(c *okapi.Context) error {
    if c.Request().TLS == nil {
        httpsURL := "https://" + c.Request().Host + c.Request().RequestURI
        return c.Redirect(http.StatusMovedPermanently, httpsURL)
    }
    return nil
}

// Apply to all routes
o.Use(redirectToHTTPS)
```

### Example: Adding HSTS Header

```go
func hstsMiddleware(next okapi.HandlerFunc) okapi.HandlerFunc {
    return func(c *okapi.Context) error {
        c.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        return next(c)
    }
}

o.Use(hstsMiddleware)
```
