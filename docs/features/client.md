---
title: HTTP Client
layout: default
parent: Features
nav_order: 14
---

# HTTP Client

The `okapi/client` package is a small fluent HTTP client:

- Accepts `context.Context` at the terminal call
- Supports middleware / interceptors
- Built-in retry policy with exponential backoff
- Zero dependency on the okapi server package — usable against any REST API


## Quick Start

```go
import (
    "context"
    "time"

    "github.com/jkaninda/okapi/client"
)

c := client.New("https://api.example.com",
    client.WithBearerToken(token),
    client.WithUserAgent("my-app/1.0"),
    client.WithTimeout(10*time.Second),
)

var user User
resp, err := c.Get("/users/42").
    WithContext(ctx).
    QueryParam("expand", "profile").
    Do()
if err != nil {
    return err
}
if err := resp.Error(); err != nil {
    return err // *client.HTTPError for non-2xx
}
if err := resp.JSON(&user); err != nil {
    return err
}
```

`Do` issues the request. `Send` is an alias for `Do` — use whichever verb reads better:

```go
resp, err := c.Get("/users/42").Send() // identical to Do()
```

For the common "do request, decode the response, fail on non-2xx" path, use `Decode`:

```go
var user User
err := c.Get("/users/42").Decode(&user)
```

`Decode` also works on a response you already hold — the format is chosen from the `Content-Type`:

```go
resp, _ := c.Get("/users/42").Do()
err := resp.Decode(&user)
```

## Client Options

Options passed to `client.New` apply to every request issued by the client.

| Option                     | Purpose                                              |
|----------------------------|------------------------------------------------------|
| `WithHTTPClient(*http.Client)` | Provide a pre-configured `http.Client` (TLS, transport) |
| `WithTimeout(d)`           | Default per-request timeout                          |
| `WithHeader(k, v)`         | Add one default header                               |
| `WithHeaders(map)`         | Merge multiple default headers                       |
| `WithBearerToken(token)`   | Set `Authorization: Bearer <token>`                  |
| `WithBasicAuth(u, p)`      | Set `Authorization: Basic ...`                       |
| `WithUserAgent(ua)`        | Set the default `User-Agent`                         |
| `WithMiddleware(mw...)`    | Append middleware to the chain                       |
| `WithRetry(policy)`        | Default retry policy                                 |

## Request Builder

Every verb method returns a `*RequestBuilder` that supports fluent configuration:

```go
resp, err := c.Post("/items").
    WithContext(ctx).
    Header("X-Trace-Id", traceID).
    QueryParam("dry_run", "true").
    JSONBody(Item{Title: "hello"}).
    Timeout(5*time.Second).
    Do()
```

### Terminal Methods

| Method            | Behavior                                                        |
|-------------------|-----------------------------------------------------------------|
| `Do()`            | Issues the request, returns `(*Response, error)`                |
| `Send()`          | Alias for `Do()`                                                |
| `Decode(target)`  | `Do()` + `Response.Decode` into `target`; returns `*HTTPError` on non-2xx |

### Body Encoders

| Method                      | Sets Content-Type                       |
|-----------------------------|-----------------------------------------|
| `JSONBody(v any)`           | `application/json`                      |
| `XMLBody(v any)`            | `application/xml`                       |
| `YAMLBody(v any)`           | `application/yaml`                      |
| `FormBody(map[string]string)` | `application/x-www-form-urlencoded`   |
| `Multipart(func(*multipart.Writer) error)` | `multipart/form-data; boundary=…` |
| `RawBody([]byte)`           | none (set with `Header`)                |
| `Body(io.Reader)`           | none (set with `Header`)                |

### Auth Helpers

```go
c.Get("/me").BearerToken(jwt).Send()
c.Get("/admin").BasicAuth("user", "pass").Send()
```

### Per-Request Overrides

Builders can override client defaults for a single call:

```go
c.Get("/big").
    Timeout(30 * time.Second).
    Retry(client.RetryPolicy{MaxAttempts: 5, BaseDelay: 100 * time.Millisecond}).
    Middleware(client.LoggingMiddleware(os.Stdout)).
    Do()
```

## Response

`Do` returns a `*Response` that wraps the underlying `*http.Response` with the body already read into memory:

```go
resp.IsSuccess()             // 2xx?
resp.Error()                 // *HTTPError for non-2xx, nil otherwise
resp.String()                // body as string
resp.Body                    // []byte
resp.Decode(&target)         // format chosen from Content-Type (xml/yaml/json)
resp.JSON(&target)
resp.XML(&target)
resp.YAML(&target)
resp.JSONPath("user.profile.name") // dot-path lookup in a JSON object
resp.Cookie("sid")           // *http.Cookie or nil
resp.Header                  // *http.Header
resp.StatusCode              // int
```

`Decode` inspects the response `Content-Type`: an `xml` type decodes as XML, a `yaml` type as YAML, and anything else as JSON. Use `JSON` / `XML` / `YAML` directly when you need explicit control.

## Middleware

Middleware composes around the underlying transport:

```go
type RoundTripFunc func(*http.Request) (*http.Response, error)
type Middleware    func(next RoundTripFunc) RoundTripFunc
```

Client middlewares are outermost; per-request middlewares run next; the retry middleware sits innermost. Built-in middlewares:

| Middleware                  | Behavior                                  |
|-----------------------------|-------------------------------------------|
| `LoggingMiddleware(io.Writer)` | One line per request (method, URL, status, duration) |
| `UserAgentMiddleware(ua)`   | Forces `User-Agent` on every request      |
| `RequestIDMiddleware()`     | Sets `X-Request-Id` (random hex) if absent |

Custom middleware example:

```go
auth := func(next client.RoundTripFunc) client.RoundTripFunc {
    return func(req *http.Request) (*http.Response, error) {
        req.Header.Set("X-Service-Token", currentServiceToken())
        return next(req)
    }
}
c := client.New(baseURL, client.WithMiddleware(auth))
```

## Retries

```go
c := client.New(baseURL, client.WithRetry(client.RetryPolicy{
    MaxAttempts: 4,
    BaseDelay:   100 * time.Millisecond,
    MaxDelay:    2 * time.Second,
}))
```

Defaults when fields are zero:

- `MaxAttempts <= 1` → no retries
- `RetryOnStatus` nil → retries on `408`, `429`, `500`, `502`, `503`, `504`
- `MaxDelay` zero → backoff doubles indefinitely
- Transport errors (network failures) always retry while attempts remain

Custom retry predicate:

```go
client.RetryPolicy{
    MaxAttempts: 3,
    BaseDelay:   50 * time.Millisecond,
    ShouldRetry: func(resp *http.Response, err error) bool {
        return err != nil || (resp != nil && resp.StatusCode == http.StatusBadGateway)
    },
}
```

Request bodies are buffered once and rewound between attempts, so retries work for POST/PUT/PATCH out of the box. Backoff is interrupted when the request context is cancelled.

## Errors

```go
resp, err := c.Get("/missing").Do()
if err != nil {
    return err // transport/build error
}
if err := resp.Error(); err != nil {
    var hErr *client.HTTPError
    if errors.As(err, &hErr) {
        fmt.Println(hErr.StatusCode, string(hErr.Body))
    }
    return err
}
```

`Do` (and its alias `Send`) never returns `HTTPError` — a non-2xx response is a valid response. Opt in via `resp.Error()` (or `Decode`, which does it for you).

