---
title: Core Concepts
sidebar_position: 5
---

# Core Concepts

The pages in this section cover the parts of Okapi you touch on every request:
how routes are matched, how request data reaches your structs, how responses are
written, and how route groups tie them together. Creating, configuring and
starting the server is covered [below](#creating-the-server).

- [Routing](routing.md) — HTTP methods, path syntax, enabling and disabling routes
- [Request Handling](request-handling.md) — path, query, form and struct binding
- [Response Handling](response-handling.md) — JSON, XML, files, cookies, aborts
- [Route Groups](group.md) — prefixes, shared middleware, tags and security

## Creating the Server

`okapi.New` creates an instance configured by `OptionFunc` values;
`okapi.Default()` is `New()` with the OpenAPI documentation enabled. Options can
also be applied later with `With`, or through the chaining method of the same
name on `*Okapi`:

```go
o := okapi.New(
    okapi.WithPort(9000),
    okapi.WithoutBanner(),
)

o.With(okapi.WithReadTimeout(30)) // apply more options later
o.WithDebug()                     // chaining-method equivalent
```

### Server Options

Each option exists as a package-level `OptionFunc` and, unless noted, as a
method on `*Okapi`. Timeouts are in seconds.

| Option                         | Effect                                                                                                    |
|--------------------------------|-----------------------------------------------------------------------------------------------------------|
| `WithAddr(addr)`               | Listen address (default `:8080`). An address without a port gets `8080`.                                  |
| `WithPort(port)`               | Listen port, keeping the host. Zero or negative means `8080`.                                             |
| `WithServer(*http.Server)`     | Use your own `http.Server`. Option only. Okapi fills only the timeouts it leaves at zero.                  |
| `WithTLS(cfg)`                 | Serve HTTPS on the main address. Option only. See [TLS/HTTPS](../features/tls-https.md).                  |
| `WithTLSServer(addr, cfg)`     | Add an HTTPS listener next to plain HTTP. Option only.                                                    |
| `WithContext(ctx)`             | Default context, used by `Stop` as the shutdown context.                                                  |
| `WithLogger(*slog.Logger)`     | Logger for access logs and Okapi's own messages.                                                          |
| `WithCors(cors)`               | Enable CORS. The method is spelled `WithCORS`. See [Middleware](../features/middleware.md#cors-middleware). |
| `WithReadTimeout(s)`           | `http.Server.ReadTimeout`. Unset by default.                                                              |
| `WithWriteTimeout(s)`          | `http.Server.WriteTimeout`. Unset by default.                                                             |
| `WithReadHeaderTimeout(s)`     | `http.Server.ReadHeaderTimeout`. Default 10.                                                              |
| `WithIdleTimeout(s)`           | `http.Server.IdleTimeout`. Default 120.                                                                   |
| `WithTrustedProxies(cidrs...)` | Peers allowed to set `X-Forwarded-For` / `X-Real-IP`. See [Request Handling](request-handling.md).        |
| `WithStrictSlash(bool)`        | When `true`, `/x/` is redirected (`301`) to `/x`, and `/x` to `/x/`, if only the other form is registered. |
| `WithDebug()`                  | Adds protocol, headers, query and path parameters to access logs and prints the route table on startup.   |
| `WithAccessLogDisabled()`      | Turn off access logging. The method is `DisableAccessLog()`.                                              |
| `WithoutBanner()`              | Do not print the server info banner on startup.                                                           |

Options for the OpenAPI document, request body limits, templates and error
handlers are covered on the [OpenAPI](../features/openapi.md),
[Request Handling](request-handling.md), [Templating](../features/templating.md)
and [Error Handling](../features/error-handling.md) pages.

### Starting and Stopping

```go
o.Start()          // listen on the configured address
o.StartOn(3000)    // set the port, then Start
o.StartServer(srv) // serve on your own *http.Server
```

All three block until the server stops. `StartServer` sets the server's
`Handler` and `BaseContext`, and serves TLS when `srv.TLSConfig` is set; the
other fields, timeouts included, are used as you set them.

Before listening, the address is checked with `okapi.ValidateAddr`, and an
invalid one **panics**. The host must be empty, an IP literal or a hostname —
`":8080"`, `"127.0.0.1:8080"`, `"[::1]:8080"` and `"localhost:8080"` are all
valid — and the port must be between 1 and 65535. Call `ValidateAddr(addr) bool` yourself to check a
configured address up front.

`Stop()` and `StopWithContext(ctx)` shut the server down gracefully; see
[TLS/HTTPS](../features/tls-https.md#graceful-shutdown).
