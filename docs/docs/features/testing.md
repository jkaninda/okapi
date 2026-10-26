---
title: Testing
sidebar_position: 9
---

# Testing

Okapi provides comprehensive testing utilities through the `okapitest` package to help you write robust tests for your handlers and middleware.

## Quick Start

```go
import (
    "testing"

    "github.com/jkaninda/okapi"
    "github.com/jkaninda/okapi/okapitest"
)

func TestGetBooksHandler(t *testing.T) {
    // Create a test server
    server := okapi.NewTestServer(t)
    server.Get("/books", GetBooksHandler)

    // Make request and assert response
    okapitest.GET(t, server.BaseURL+"/books").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")
}
```

## Test Servers

| Constructor                        | Description                                                            |
|------------------------------------|------------------------------------------------------------------------|
| `okapi.NewTestServer(t)`           | An `okapi.New()` instance behind an `httptest.Server`, closed on cleanup |
| `okapi.DefaultTestServer(t)`       | The same, from `okapi.Default()` — documentation routes included        |
| `okapi.NewTestServerWithOkapi(t, o)` | Wraps an instance you configured yourself                            |
| `okapi.NewTestServerOn(t, port)`   | Starts a real listener on `port` (for tests that need a fixed address); a port `<= 0` picks a free one |

All of them return a `*okapi.TestServer`, which embeds `*okapi.Okapi` — so you
register routes and middleware on it exactly as on an application — and exposes
`BaseURL`.

The constructors take an `okapi.TestingT` rather than `*testing.T`: any value with
`Helper()`, `Cleanup(func())`, `Errorf(...)` and `Fatalf(...)` will do, so
`*testing.T` and `*testing.B` both work.

To run an application you built yourself on a real port, use `StartForTest`. It
starts the server in the background, waits for it to accept connections (failing
the test after two seconds), stops it on cleanup, and returns the base URL:

```go
func TestApp(t *testing.T) {
    app := okapi.New(okapi.WithPort(8089))
    app.Get("/health", HealthHandler)

    baseURL := app.StartForTest(t)

    okapitest.GET(t, baseURL+"/health").ExpectStatusOK()
}
```

When you start the server yourself, `app.WaitForServer(timeout)` blocks until the
server's address accepts TCP connections and returns that address (such as
`":8089"`), or `""` if the timeout expires first.

For a handler test that needs no server at all, `okapi.NewTestContext(method, url, body)`
returns a `*okapi.Context` and the `*httptest.ResponseRecorder` behind it.

## Testing Approaches

### 1. Using the Test Client (Recommended)

The test client provides a fluent API for making multiple requests with shared configuration:

```go
func TestBooksAPI(t *testing.T) {
    // Setup test server
    server := okapi.NewTestServer(t)
    server.Get("/books", GetBooksHandler)
    server.Get("/books/:id", GetBookHandler)
    server.Post("/books", CreateBookHandler)

    // Create reusable client
    client := okapitest.NewClient(t, server.BaseURL)

    // Test listing books
    client.GET("/books").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language").
        ExpectHeader("X-Version", "1.0.0")

    // Test getting a specific book
    client.GET("/books/1").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")

    // Test book not found
    client.GET("/books/999").
        ExpectStatusNotFound().
        ExpectBodyContains("Book not found")

    // Test creating a book
    newBook := Book{
        ID:    6,
        Name:  "Sample Book",
        Price: 20,
        Year:  2024,
        Qty:   5,
    }
    client.POST("/books").
        JSONBody(newBook).
        ExpectStatusCreated().
        ExpectBodyContains("Sample Book")
}
```

`okapitest.NewClient(t, baseURL)` returns a `*okapitest.TestClient`. Its methods —
`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD` and `OPTIONS` — take a path that is
appended to `BaseURL`, and return the same `*RequestBuilder` as the standalone
helpers. Headers set on `client.Headers` are merged into every request the client
builds.

### 2. Using Standalone Request Helpers

For simpler test cases or one-off requests:

```go
func TestGetBookHandler(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Get("/books/:id", GetBookHandler)

    // Test successful retrieval
    okapitest.GET(t, server.BaseURL+"/books/1").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")

    // Test not found scenario
    okapitest.GET(t, server.BaseURL+"/books/999").
        ExpectStatusNotFound().
        ExpectBodyContains("Book not found")
}

func TestCreateBookHandler(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Post("/books", CreateBookHandler)

    book := Book{
        ID:    6,
        Name:  "Sample Book",
        Price: 20,
        Year:  2024,
        Qty:   5,
    }

    okapitest.POST(t, server.BaseURL+"/books").
        JSONBody(book).
        ExpectStatusCreated().
        ExpectBodyContains("Sample Book")
}
```

`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD` and `OPTIONS` all take
`(t, url)`. `okapitest.Request(t)` returns a bare builder when you want to set the
method and URL yourself, and `okapitest.FromRecorder(t, rec)` asserts against a
response you already recorded — for example one written through
`okapi.NewTestContext`:

```go
func TestHealthHandler(t *testing.T) {
    ctx, rec := okapi.NewTestContext(http.MethodGet, "/health", nil)
    if err := HealthHandler(ctx); err != nil {
        t.Fatal(err)
    }

    okapitest.FromRecorder(t, rec).
        ExpectStatusOK().
        ExpectBodyContains("ok")
}
```

## Available Assertions

```go
// Status code assertions
.ExpectStatus(code int)     // Custom status code
.ExpectStatusOK()           // 200
.ExpectStatusCreated()      // 201
.ExpectStatusAccepted()     // 202
.ExpectStatusNoContent()    // 204
.ExpectStatusBadRequest()   // 400
.ExpectStatusUnauthorized() // 401
.ExpectStatusForbidden()    // 403
.ExpectStatusNotFound()     // 404
.ExpectStatusConflict()     // 409
.ExpectStatusInternalServerError() // 500

// Body assertions
.ExpectBody(expected string)          // exact body match
.ExpectBodyContains(substr string)
.ExpectContains(substr string)        // alias of ExpectBodyContains
.ExpectBodyNotContains(substr string)
.ExpectEmptyBody()
.ExpectJSON(expected any)             // compares decoded JSON
.ExpectJSONPath(path string, expected any) // dot-path lookup, e.g. "data.name"

// Header, cookie and content-type assertions
.ExpectHeader(key, value string)
.ExpectHeaderContains(key, substr string)
.ExpectHeaderExists(key string)
.ExpectCookie(key, value string)
.ExpectCookieExist(key string)
.ExpectContentType(contentType string)
```

`ExpectBody` is the exact-match assertion (there is no `ExpectBodyEquals`), and
`ExpectJSON` is the JSON comparison (there is no `ExpectJSONBody`).

To inspect the response yourself, use `.ParseJSON(&target)` inside the chain, or
`.Execute()` to get the `*http.Response` and the body bytes.

## Request Building

```go
okapitest.Request(t).
    Method(http.MethodPost).
    URL(baseURL).
    Path("/books").           // appends a path segment to the URL
    Header("X-Trace-Id", "abc").
    Headers(map[string]string{"X-A": "1", "X-B": "2"}).
    QueryParam("dry_run", "true").
    QueryParams(map[string]string{"page": "1"}).
    SetBearerAuth("token123").
    SetBasicAuth("user", "pass").
    JSONBody(payload).        // or FormBody(map[string]string), Body(io.Reader)
    Timeout(5 * time.Second).
    ExpectStatusCreated()
```

### Deprecated Helpers

`okapitest.AssertHTTPStatus(t, method, url, headers, body, contentType, expected)`
and `okapitest.AssertHTTPResponse(t, method, url, headers, body, contentType, expectedStatus, expectedBody)`
are kept for compatibility and are deprecated in favour of the request builder.
`AssertHTTPResponse` skips the body check when `expectedBody` is empty.

## Testing with Custom Headers

```go
func TestAuthenticatedRequest(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Get("/protected", ProtectedHandler)

    client := okapitest.NewClient(t, server.BaseURL)

    client.GET("/protected").
        Header("Authorization", "Bearer token123").
        ExpectStatusOK()
}
```

## Testing Middleware

```go
func TestAuthMiddleware(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Use(AuthMiddleware)
    server.Get("/protected", ProtectedHandler)

    client := okapitest.NewClient(t, server.BaseURL)

    // Test without auth - should fail
    client.GET("/protected").
        ExpectStatus(401)

    // Test with auth - should succeed
    client.GET("/protected").
        Header("Authorization", "Bearer valid-token").
        ExpectStatusOK()
}
```

## Testing a Server That Runs Until Signalled

`okapitest.GracefulExitAfter(d)` sends the current process `SIGTERM` after `d`,
which is how you exercise a server that only stops on a signal — such as one
started with `okapicli.RunServer`:

```go
func TestRunServer(t *testing.T) {
    okapitest.GracefulExitAfter(500 * time.Millisecond)

    if err := cli.RunServer(); err != nil {
        t.Fatal(err)
    }
}
```

The helper compiles on Windows, where a process cannot signal itself, but panics
with an explanation if it is called there.

## Golden OpenAPI Document

`okapitest.AssertOpenAPIGolden` compares the application's OpenAPI 3.1 document
with a file checked into the repository, so a change to the documented API shows
up in the pull request's diff instead of going unnoticed:

```go
func TestOpenAPI(t *testing.T) {
    okapitest.AssertOpenAPIGolden(t, newApp(), "testdata/openapi.json")
}
```

The second argument is an `okapitest.OpenAPIProvider` — any value with an
`OpenAPISpec() *openapi3.T` method. `*okapi.Okapi` implements it.

On a mismatch the test fails with the first differing lines. Run the tests with
`OKAPI_UPDATE_GOLDEN=1` (the `okapitest.UpdateGoldenEnv` constant) to write or update the golden file:

```bash
OKAPI_UPDATE_GOLDEN=1 go test ./...
```
