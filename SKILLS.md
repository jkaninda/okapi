# Okapi - AI Agent Skills Reference

> A modern, minimalist HTTP web framework for Go inspired by FastAPI's elegant design philosophy.
> Built on `gorilla/mux` with `net/http` compatibility. Module: `github.com/jkaninda/okapi`

---

## Project Structure

```
okapi.go          - Core framework (Okapi struct, constructors, server lifecycle, route registration)
context.go        - Request/response context (data store, binding, responses, SSE, errors)
route.go          - Route and RouteDefinition types, route options
group.go          - Route groups with prefix, middleware, security
binder.go         - Request binding (JSON, XML, YAML, Protobuf, form, multipart, query, header, path, cookie)
validator.go      - Struct tag validation (min, max, pattern, enum, format, etc.)
openapi.go        - OpenAPI spec generation, doc options, DocBuilder
doc.go            - Swagger UI / ReDoc HTML templates and endpoints
middlewares.go    - Built-in middleware (logger, BasicAuth, JWTAuth, BodyLimit)
jwt.go            - JWT token generation, validation, key resolution
jwks.go           - JWKS loading (remote URL, file, base64)
jwt_claims_expression.go - DSL for JWT claim validation (Equals, Contains, OneOf, And, Or, Not)
cors.go           - CORS middleware configuration
sse.go            - Server-Sent Events (Message, streaming, serializers)
template.go       - HTML template loading (files, directory, embedded FS, config)
renderer.go       - Renderer interface and RendererFunc adapter
errors.go         - Error types (ErrorResponse, ValidationError, ProblemDetail RFC 7807)
static.go         - Static file serving with directory listing prevention
tests.go          - TestServer and NewTestContext for unit tests
util.go           - Utility functions (TLS config loading, helpers)
helper.go         - Internal utilities
constants.go      - Default values, tag names, format types
var.go            - Package-level variables
version.go        - Version constant
okapitest/        - Fluent HTTP test client (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS + assertions)
okapicli/         - CLI integration (flags, env config, subcommands, struct-based config, signal handling)
examples/         - Example applications (sample, group, middleware, tls, sse, template, cli, etc.)
```

---

## Core Types

| Type | Description |
|------|-------------|
| `Okapi` | Main application struct |
| `Context` (aliases: `C`, `Ctx`) | Per-request context with store, binding, response helpers |
| `Route` | Registered route with metadata |
| `RouteDefinition` | Declarative route definition struct |
| `Group` | Route group with shared prefix, middleware, security |
| `HandlerFunc` | `func(*Context) error` |
| `Middleware` | `func(next HandlerFunc) HandlerFunc` |
| `RouteOption` | `func(*Route)` - composable route configuration |
| `OptionFunc` | `func(*Okapi)` - composable app configuration |
| `M` | `map[string]any` shorthand |
| `ResponseWriter` | Extended `http.ResponseWriter` with status tracking, byte counting, hijack, flush, push |
| `Renderer` | Interface: `Render(io.Writer, string, interface{}, *Context) error` |
| `ErrorHandler` | `func(*Context, int, string, error) error` |

---

## Constructors

- `okapi.New(options ...OptionFunc) *Okapi` - Minimal instance (no docs by default)
- `okapi.Default() *Okapi` - With logger middleware and OpenAPI docs enabled

---

## App Configuration (OptionFunc / Chainable)

All available as both `OptionFunc` (for `New()`) and chainable methods on `*Okapi`:

| Method | Description |
|--------|-------------|
| `WithPort(port int)` | Server port (default: 8080) |
| `WithAddr(addr string)` | Server address |
| `WithTLS(tlsConfig *tls.Config)` | Enable TLS |
| `WithTLSServer(addr string, tlsConfig *tls.Config)` | Separate TLS server |
| `WithCors(cors Cors)` / `WithCORS(cors Cors)` | CORS configuration |
| `WithLogger(logger *slog.Logger)` | Structured logger |
| `WithContext(ctx context.Context)` | Application context |
| `WithDebug()` | Debug mode |
| `DisableAccessLog()` / `WithAccessLogDisabled()` | Disable access logging |
| `WithWriteTimeout(seconds int)` | HTTP write timeout |
| `WithReadTimeout(seconds int)` | HTTP read timeout |
| `WithIdleTimeout(seconds int)` | HTTP idle timeout |
| `WithStrictSlash(strict bool)` | Trailing slash behavior |
| `WithMaxMultipartMemory(max int64)` | Multipart memory limit (default: 32MB) |
| `WithMuxRouter(router *mux.Router)` | Custom gorilla/mux router |
| `WithServer(server *http.Server)` | Custom HTTP server |
| `WithOpenAPIDocs(cfg ...OpenAPI)` | Enable/configure OpenAPI docs |
| `WithOpenAPIDisabled()` | Disable OpenAPI docs |
| `WithRenderer(renderer Renderer)` | Set template renderer |
| `WithDefaultRenderer(templatePath string)` | Load templates from path |
| `WithRendererFromFS(fsys fs.FS, pattern string)` | Load from embedded FS |
| `WithRendererFromDirectory(dir string, ext ...string)` | Load from directory |
| `WithRendererConfig(config TemplateConfig)` | Load with config |
| `WithErrorHandler(handler ErrorHandler)` | Custom error handler |
| `WithDefaultErrorHandler()` | Reset to default error handler |
| `WithProblemDetailErrorHandler(config *ErrorHandlerConfig)` | RFC 7807 errors |
| `WithSimpleProblemDetailErrorHandler()` | RFC 7807 with defaults |

---

## Server Lifecycle

- `Start() error` - Start on configured port
- `StartOn(port int) error` - Start on specific port
- `StartServer(server *http.Server) error` - Start custom server
- `Stop() error` - Graceful shutdown
- `StopWithContext(ctx context.Context) error` - Shutdown with context
- `Shutdown(server *http.Server, ctx ...context.Context) error` - Shutdown specific server
- `GetContext() context.Context` / `SetContext(ctx context.Context)` - Application context access

---

## Route Registration

### HTTP Methods on `*Okapi` and `*Group`

```go
Get(path, handler, ...RouteOption) *Route
Post(path, handler, ...RouteOption) *Route
Put(path, handler, ...RouteOption) *Route
Delete(path, handler, ...RouteOption) *Route
Patch(path, handler, ...RouteOption) *Route
Head(path, handler, ...RouteOption) *Route
Options(path, handler, ...RouteOption) *Route
Any(path, handler, ...RouteOption) *Route   // All methods (Okapi only)
```

### Standard Library Handlers

```go
// On *Okapi and *Group
HandleStd(method, path string, h func(http.ResponseWriter, *http.Request), opts ...RouteOption)
HandleHTTP(method, path string, h http.Handler, opts ...RouteOption)
```

### Path Parameter Types

```
/books/{id}         - string parameter
/books/{id:int}     - integer parameter
/books/{id:uuid}    - UUID parameter
```

### Declarative Registration

```go
app.Register(okapi.RouteDefinition{
    Method:      http.MethodPost,
    Path:        "/books",
    Handler:     createBook,
    Group:       apiGroup,
    OperationId: "create-book",
    Summary:     "Create a book",
    Description: "Detailed description",
    Tags:        []string{"Books"},
    Request:     &BookRequest{},
    Response:    &BookResponse{},
    Security:    []map[string][]string{{"bearerAuth": {}}},
    Options:     []okapi.RouteOption{...},
    Middlewares:  []okapi.Middleware{authMiddleware},
})
```

### Route Methods

- `route.Hide()` - Hide from OpenAPI docs
- `route.Disable()` / `route.Enable()` - Runtime enable/disable
- `route.Use(middlewares...)` - Per-route middleware
- `route.WithIO(req, res)` - Set request/response schemas
- `route.WithInput(req)` - Set request schema
- `route.WithOutput(res)` - Set response schema

### Route Introspection

- `app.Routes() []Route` - List all registered routes

---

## Generic Handler Wrappers

```go
// Auto-bind input struct
okapi.Handle[I](func(c *Context, input *I) error) HandlerFunc
okapi.H[I](func(c *Context, input *I) error) HandlerFunc         // Shortcut

// Bind input + return typed output
okapi.HandleIO[I, O](func(c *Context, input *I) (*O, error)) HandlerFunc

// Return typed output only
okapi.HandleO[O](func(c *Context) (*O, error)) HandlerFunc
```

---

## Route Groups

```go
group := app.Group("/api", middleware1, middleware2)
sub   := group.Group("/v1")

group.Disable() / group.Enable()       // Runtime toggle
group.WithBearerAuth()                  // Require Bearer auth
group.WithBasicAuth()                   // Require Basic auth
group.WithTags([]string{"API"})         // OpenAPI tags
group.WithSecurity(schemes)             // Security schemes
group.Deprecated()                      // Mark deprecated
group.Use(middleware)                   // Add middleware
group.UseMiddleware(stdMiddleware)      // Standard http middleware
group.Register(routes...)               // Bulk registration
group.HandleStd(method, path, handler)  // Standard http handler
group.HandleHTTP(method, path, handler) // http.Handler
```

---

## Request Binding

### Struct Tags

| Tag | Source | Example |
|-----|--------|---------|
| `json:"name"` | JSON body | `Name string \`json:"name"\`` |
| `xml:"name"` | XML body | `Name string \`xml:"name"\`` |
| `query:"name"` | Query parameter | `Page int \`query:"page"\`` |
| `path:"id"` / `param:"id"` | Path parameter | `ID int \`path:"id"\`` |
| `header:"X-Key"` | HTTP header | `Key string \`header:"X-Key"\`` |
| `cookie:"session"` | Cookie | `Sess string \`cookie:"session"\`` |
| `form:"file"` | Form field / file | `File string \`form:"file"\`` |

### Binding Methods on Context

```go
c.Bind(&v)            // Auto-bind (Body field or flat struct)
c.B(&v)               // Shortcut for Bind
c.ShouldBind(&v)      // Returns (bool, error)
c.BindJSON(&v)        // JSON body
c.BindXML(&v)         // XML body
c.BindYAML(&v)        // YAML body
c.BindProtoBuf(&msg)  // Protobuf body
c.BindQuery(&v)       // Query params
c.BindForm(&v)        // Form data
c.BindMultipart(&v)   // Multipart form
```

### Body Field Pattern

Separate payload from metadata:

```go
type CreateBookRequest struct {
    Body   Book   `json:"body"`                          // Request payload
    ID     int    `param:"id" query:"id"`                // Path or query param
    APIKey string `header:"X-API-Key" required:"true"`   // Header
}
```

---

## Validation Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `required:"true"` | Field is required | `Name string \`required:"true"\`` |
| `min:"5"` | Minimum numeric value | `Price int \`min:"5"\`` |
| `max:"100"` | Maximum numeric value | `Price int \`max:"100"\`` |
| `minLength:"3"` | Minimum string length | `Name string \`minLength:"3"\`` |
| `maxLength:"50"` | Maximum string length | `Name string \`maxLength:"50"\`` |
| `pattern:"^[A-Z]"` | Regex pattern | `Code string \`pattern:"^[A-Z]+$"\`` |
| `enum:"a,b,c"` | Allowed values | `Status string \`enum:"active,inactive"\`` |
| `default:"val"` | Default value | `Status string \`default:"active"\`` |
| `format:"email"` | Format validation | `Email string \`format:"email"\`` |
| `multipleOf:"5"` | Must be multiple of | `Qty int \`multipleOf:"5"\`` |
| `minItems:"1"` | Minimum slice length | `Tags []string \`minItems:"1"\`` |
| `maxItems:"10"` | Maximum slice length | `Tags []string \`maxItems:"10"\`` |
| `uniqueItems:"true"` | Unique slice items | `Tags []string \`uniqueItems:"true"\`` |
| `deprecated:"true"` | Mark as deprecated | `Old string \`deprecated:"true"\`` |
| `description:"text"` | Field description | `Name string \`description:"User name"\`` |
| `example:"value"` | Example value | `Name string \`example:"John"\`` |

### Supported Formats

`email`, `date-time` (RFC3339), `date` (YYYY-MM-DD), `duration`, `ipv4`, `ipv6`, `hostname`, `uri`, `uuid`, `regex`

---

## Response Methods on Context

### JSON Responses

```go
c.OK(data)            // 200
c.Created(data)       // 201
c.NoContent()         // 204
c.JSON(code, data)    // Custom status
```

### Other Formats

```go
c.XML(code, data)
c.YAML(code, data)
c.Text(code, data) / c.String(code, data)
c.Data(code, contentType, []byte)
c.HTML(code, file, data)
c.HTMLView(code, templateStr, data)
c.Render(code, name, data)         // Uses configured Renderer
c.Redirect(code, location)
```

### File Serving

```go
c.ServeFile(path)
c.ServeFileFromFS(filepath, fs)
c.ServeFileAttachment(path, filename)   // Download
c.ServeFileInline(path, filename)       // Inline
```

### Structured Response (Body/Status/Header pattern)

```go
type BookResponse struct {
    Status    int    // HTTP status code
    Body      Book   // Response payload
    RequestID string `header:"X-Request-ID"`  // Response header
}
c.Respond(&response) / c.Return(&response)
```

### Response Control

```go
c.WriteStatus(code)              // Set HTTP status code
c.SetHeader(key, value)          // Set response header
c.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
c.Response()                     // Access ResponseWriter
```

---

## ResponseWriter Extensions

The `ResponseWriter` interface extends `http.ResponseWriter` with:

```go
StatusCode() int                          // Get written HTTP status code
BytesWritten() int                        // Get total bytes written
Close()                                   // Close the writer
Hijack() (net.Conn, *bufio.ReadWriter, error)  // Upgrade to raw TCP (WebSockets, proxies)
Flush()                                   // Flush buffered data (streaming, SSE, gzip)
Push(target string, opts *http.PushOptions) error  // HTTP/2 server push
```

---

## Error Handling

### Abort Methods (use configured ErrorHandler)

Every HTTP status code has a dedicated method:

```go
c.AbortBadRequest(msg, ...err)           // 400
c.AbortUnauthorized(msg, ...err)         // 401
c.AbortForbidden(msg, ...err)            // 403
c.AbortNotFound(msg, ...err)             // 404
c.AbortConflict(msg, ...err)             // 409
c.AbortValidationError(msg, ...err)      // 422
c.AbortValidationErrors([]ValidationError, ...msg)  // 422 detailed
c.AbortTooManyRequests(msg, ...err)      // 429
c.Abort(err)                             // 500
c.AbortInternalServerError(msg, ...err)  // 500
c.AbortServiceUnavailable(msg, ...err)   // 503
// ... and all other 4xx/5xx codes
```

### Raw Error Methods (direct JSON write)

```go
c.ErrorBadRequest(message)
c.ErrorNotFound(message)
c.ErrorInternalServerError(message)
// ... etc.
```

### RFC 7807 Problem Details

```go
detail := okapi.NewProblemDetail(400, "https://example.com/errors/bad-input", "Invalid input").
    WithInstance("/books/123").
    WithExtension("field", "name").
    WithTimestamp()
c.AbortWithProblemDetail(detail)
```

### Error Handler Configuration

```go
app.WithErrorHandler(customHandler)
app.WithProblemDetailErrorHandler(&ErrorHandlerConfig{...})
app.WithSimpleProblemDetailErrorHandler()
app.WithDefaultErrorHandler()
```

---

## OpenAPI Documentation

### Endpoints (when enabled)

- `/docs` - Swagger UI
- `/redoc` - ReDoc
- `/openapi.json` - OpenAPI spec

### Route Documentation Options (Composable)

```go
okapi.DocSummary("List books")
okapi.DocDescription("Detailed description")
okapi.DocOperationId("list-books")
okapi.DocTags("Books", "Public")
okapi.DocPathParam(name, type, description)
okapi.DocQueryParam(name, type, description, required)
okapi.DocHeader(name, type, description, required)
okapi.DocRequestBody(&BookRequest{})
okapi.DocResponse(&Book{})
okapi.DocResponse(201, &Book{})
okapi.DocErrorResponse(400, &ErrorResponse{})
okapi.DocResponseHeader(name, type, ...description)
okapi.DocBearerAuth()
okapi.DocBasicAuth()
okapi.DocDeprecated()
okapi.DocHide()
okapi.Request(&BookRequest{})
okapi.Response(&BookResponse{})
```

### Fluent DocBuilder

```go
okapi.Doc().
    Summary("Create a book").
    Tags("Books").
    BearerAuth().
    RequestBody(BookRequest{}).
    Response(201, Book{}).
    Response(400, ErrorResponse{}).
    ResponseHeader("X-Request-ID", "string", "Request ID").
    Build()
```

### Fluent Route Methods

```go
route.WithIO(&BookRequest{}, &BookResponse{})
route.WithInput(&BookRequest{})
route.WithOutput(&BookResponse{})
```

### OpenAPI Configuration

```go
app.WithOpenAPIDocs(okapi.OpenAPI{
    Title:           "My API",
    Version:         "1.0.0",
    Servers:         okapi.Servers{{URL: "https://api.example.com"}},
    License:         okapi.License{Name: "MIT"},
    Contact:         okapi.Contact{Name: "Team", Email: "team@example.com"},
    SecuritySchemes: schemes,
    ExternalDocs:    &okapi.ExternalDocs{URL: "https://docs.example.com"},
})
```

---

## Authentication

### JWT Authentication

```go
jwtAuth := okapi.JWTAuth{
    SigningSecret:    []byte("secret"),          // HMAC secret
    RsaKey:          &publicKey,                 // OR RSA public key
    JwksUrl:         "https://.../.well-known/jwks.json", // OR remote JWKS
    JwksFile:        &okapi.Jwks{...},           // OR static JWKS
    Algo:            "RS256",                    // Expected algorithm
    Audience:        "my-api",                   // Expected audience
    Issuer:          "https://auth.example.com", // Expected issuer
    TokenLookup:     "header:Authorization",     // Or "query:token", "cookie:jwt"
    ContextKey:      "user",                     // Store claims in context
    ForwardClaims:   map[string]string{"email": "email", "role": "realm_access.roles"},
    ClaimsExpression: `And(Equals("email_verified","true"), OneOf("role","admin","editor"))`,
    ValidateClaims:  func(c *Context, claims jwt.Claims) error { ... },
    OnUnauthorized:  func(c *Context) error { ... },
}
protected := app.Group("/api", jwtAuth.Middleware).WithBearerAuth()
```

### Claims Expression DSL

```go
Equals(claimKey, expected)
Prefix(claimKey, prefix)
Contains(claimKey, ...values)
OneOf(claimKey, ...values)
And(left, right)
Or(left, right)
Not(expr)
```

### JWT Token Generation

```go
token, err := okapi.GenerateJwtToken(secret, jwt.MapClaims{"sub": "123"}, 24*time.Hour)
```

### JWKS Loading

```go
jwks, err := okapi.LoadJWKSFromFile("path/to/jwks.json")  // Or base64 string
```

### Basic Authentication

```go
basicAuth := okapi.BasicAuth{
    Username:   "admin",
    Password:   "password",
    Realm:      "Admin Area",
    ContextKey: "user",
}
admin := app.Group("/admin", basicAuth.Middleware)
```

---

## Middleware

### Built-in

- `okapi.LoggerMiddleware` - Access logging
- `okapi.BasicAuth{}.Middleware` - Basic authentication
- `okapi.JWTAuth{}.Middleware` - JWT authentication
- `okapi.BodyLimit{MaxBytes: 1<<20}.Middleware` - Request body size limit
- `okapi.Cors{}.CORSHandler` - CORS handling

### Global Middleware

```go
app.Use(middleware1, middleware2)
app.UseMiddleware(stdHttpMiddleware)  // func(http.Handler) http.Handler
```

### Per-Route / Per-Group Middleware

```go
route.Use(cacheMiddleware)
group.Use(authMiddleware)
okapi.UseMiddleware(middleware)  // As RouteOption
```

### Middleware Chaining

```go
// Context-based middleware with Next()
func myMiddleware(next okapi.HandlerFunc) okapi.HandlerFunc {
    return func(c *okapi.Context) error {
        // before
        err := c.Next()
        // after
        return err
    }
}
```

---

## CORS

```go
app.WithCORS(okapi.Cors{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    ExposeHeaders:    []string{"X-Request-ID"},
    MaxAge:           3600,
    AllowCredentials: true,
})
```

---

## Server-Sent Events (SSE)

### Single Events

```go
c.SSEvent("message", data)
c.SSESendEvent(id, eventType, data)
c.SSESendData(data)
c.SSESendJSON(data)
c.SSESendText(text)
c.SSESendBinary([]byte)
c.SendSSECustom(data, serializer)
```

### Streaming

```go
c.SSEStream(ctx, messageChan)
c.SSEStreamWithOptions(ctx, messageChan, &okapi.StreamOptions{
    Serializer:   &okapi.JSONSerializer{},
    PingInterval: 30 * time.Second,
    OnError:      func(err error) { log.Println(err) },
})
```

### Serializers

- `okapi.JSONSerializer` - JSON format
- `okapi.TextSerializer` - Plain text
- `okapi.Base64Serializer` - Base64 encoding

---

## Template Rendering

```go
// From directory
tmpl, _ := okapi.NewTemplateFromDirectory("views", ".html")
app.WithRenderer(tmpl)

// From embedded FS
//go:embed views/*
var Views embed.FS
tmpl, _ := okapi.NewTemplate(Views, "views/*.html")
app.WithRenderer(tmpl)

// Render in handler
c.Render(http.StatusOK, "home", okapi.M{"title": "Welcome"})
c.HTML(http.StatusOK, "home.html", data)
```

---

## Static Files

```go
app.Static("/assets", "public/assets")       // Serve directory
app.StaticFile("/favicon.ico", "favicon.ico") // Serve single file
app.StaticFS("/assets", http.FS(embedFS))     // Serve from http.FileSystem
```

Directory listing is disabled by default for security.

---

## TLS / HTTPS

```go
// Load TLS config from cert/key files with optional client auth
tlsConfig, err := okapi.LoadTLSConfig(certFile, keyFile, caFile, clientAuth)

// Enable TLS on the main server
app.WithTLS(tlsConfig)

// Run a separate TLS server alongside HTTP
app.WithTLSServer(":443", tlsConfig)
```

Supports dual protocol (HTTP + HTTPS simultaneously), client certificate authentication, and custom `*tls.Config`.

---

## Testing

### Test Server

```go
server := okapi.NewTestServer(t)
server.Get("/books", handler)
// server.BaseURL gives the test server URL
```

### Test Context (unit tests)

```go
ctx, recorder := okapi.NewTestContext("GET", "/books", nil)
```

### Fluent HTTP Client (`okapitest` package)

```go
import "github.com/jkaninda/okapi/okapitest"

okapitest.GET(t, url).
    Header("Authorization", "Bearer token").
    ExpectStatusOK().
    ExpectBodyContains("Go Programming").
    ExpectHeader("Content-Type", "application/json")

okapitest.POST(t, url).
    JSONBody(map[string]any{"name": "Book"}).
    ExpectStatus(201)
```

### TestClient (reusable with base URL + default headers)

```go
client := okapitest.NewClient(t, server.BaseURL)
client.GET("/books").ExpectStatusOK()
client.POST("/books").JSONBody(book).ExpectStatusCreated()
```

### Request Builder Methods

```go
// Request construction
rb.Method(method)                      // Set HTTP method
rb.URL(url)                            // Set full URL
rb.Path(path)                          // Append path segment
rb.Header(key, value)                  // Set single header
rb.Headers(map[string]string{...})     // Set multiple headers
rb.QueryParam(key, value)              // Add query parameter
rb.QueryParams(map[string]string{...}) // Add multiple query parameters
rb.SetBasicAuth(username, password)    // Set Basic auth header
rb.SetBearerAuth(token)               // Set Bearer auth header
rb.Body(reader)                        // Set raw body
rb.JSONBody(data)                      // Set JSON body (auto-marshal)
rb.FormBody(values)                    // Set form-encoded body
rb.Timeout(duration)                   // Set request timeout

// Execution
rb.Execute() (*http.Response, []byte)  // Execute and return raw response
```

### Response Assertions

```go
// Status codes
rb.ExpectStatus(code)
rb.ExpectStatusOK()                    // 200
rb.ExpectStatusCreated()               // 201
rb.ExpectStatusAccepted()              // 202
rb.ExpectStatusNoContent()             // 204
rb.ExpectStatusBadRequest()            // 400
rb.ExpectStatusUnauthorized()          // 401
rb.ExpectStatusForbidden()             // 403
rb.ExpectStatusNotFound()              // 404
rb.ExpectStatusConflict()              // 409
rb.ExpectStatusInternalServerError()   // 500

// Body
rb.ExpectBody(expected)                // Exact match
rb.ExpectBodyContains(substr)          // Contains substring
rb.ExpectContains(substr)              // Alias for ExpectBodyContains
rb.ExpectBodyNotContains(substr)       // Does not contain
rb.ExpectEmptyBody()                   // Body is empty

// JSON
rb.ExpectJSON(expected)                // Deep-equal JSON comparison
rb.ExpectJSONPath("path.to.field", v)  // Assert specific JSON path value
rb.ParseJSON(&target)                  // Unmarshal response into struct

// Headers
rb.ExpectHeader(key, value)            // Exact header value match
rb.ExpectHeaderContains(key, substr)   // Header value contains substring
rb.ExpectHeaderExists(key)             // Header is present
rb.ExpectContentType(contentType)      // Shortcut for Content-Type header

// Cookies
rb.ExpectCookieExist(name)             // Cookie exists with non-empty value
rb.ExpectCookie(name, value)           // Cookie has exact value
```

### FromRecorder (for direct handler testing)

```go
okapitest.FromRecorder(t, recorder).
    ExpectStatusOK().
    ExpectBodyContains("success")
```

### Utilities

```go
okapitest.GracefulExitAfter(duration)  // Send SIGTERM after duration (for integration tests)
```

---

## CLI Integration (`okapicli` package)

### Basic Usage

```go
import "github.com/jkaninda/okapi/okapicli"

cli := okapicli.New(app, "My API").
    String("config", "c", "config.yaml", "Config file path").
    Int("port", "p", 8000, "Server port").
    Bool("debug", "d", false, "Enable debug mode").
    Float("rate", "r", 1.0, "Rate limit").
    Duration("timeout", "t", 30*time.Second, "Request timeout")

cli.Parse()
app.WithPort(cli.GetInt("port"))
cli.Run()
```

### Struct-Based Configuration

```go
type Config struct {
    Port    int           `cli:"port"    short:"p" desc:"Server port"    env:"PORT"    default:"8080"`
    Debug   bool          `cli:"debug"   short:"d" desc:"Debug mode"     env:"DEBUG"`
    Config  string        `cli:"config"  short:"c" desc:"Config file"    env:"CONFIG"`
    Timeout time.Duration `cli:"timeout" short:"t" desc:"Timeout"        env:"TIMEOUT" default:"30s"`
}

cfg := &Config{}
cli := okapicli.New(app, "My API").FromStruct(cfg)
cli.Parse()
// cfg fields are populated with resolved values (CLI > env > default)
```

### Subcommands

```go
cli.Command("serve", "Start the HTTP server", func(cmd *okapicli.Command) error {
    port := cmd.GetInt("port")
    cmd.Okapi().WithPort(port)
    return cmd.CLI().Run()
}).Int("port", "p", 8080, "HTTP server port")

cli.Command("migrate", "Run database migrations", func(cmd *okapicli.Command) error {
    // migration logic
    return nil
}).String("dsn", "", "", "Database connection string")

cli.DefaultCommand("serve")  // Run "serve" when no subcommand is specified
cli.Execute()
```

### Command Methods

```go
cmd.Name() string              // Command name
cmd.CLI() *CLI                 // Parent CLI instance
cmd.Okapi() *okapi.Okapi       // Okapi instance from parent CLI
cmd.Args() []string            // Non-flag arguments
cmd.GetString(name) string
cmd.GetInt(name) int
cmd.GetBool(name) bool
cmd.GetFloat(name) float64
cmd.GetDuration(name) time.Duration
cmd.FromStruct(v)              // Register flags from struct tags
```

### Server Lifecycle via CLI

```go
// Simple run with default options (30s shutdown timeout, SIGINT/SIGTERM)
cli.Run()

// Run with custom options
cli.RunServer(&okapicli.RunOptions{
    ShutdownTimeout: 10 * time.Second,
    Signals:         []os.Signal{okapicli.SIGINT, okapicli.SIGTERM},
    OnStart:         func() { log.Println("Starting...") },
    OnStarted:       func() { log.Println("Server ready") },
    OnShutdown:      func() { log.Println("Shutting down...") },
})
```

### Configuration File Loading

```go
var cfg AppConfig
cli.LoadConfig("config.yaml", &cfg)  // Supports .json, .yaml, .yml
```

### Flag Retrieval

```go
cli.GetString(name) string
cli.GetInt(name) int
cli.GetBool(name) bool
cli.GetFloat(name) float64
cli.GetDuration(name) time.Duration
cli.Get(name) interface{}
cli.MustParse() *CLI             // Parse or panic
cli.Okapi() *okapi.Okapi         // Access underlying Okapi instance
cli.MatchedCommand() *Command    // Get matched subcommand after Execute()
```

---

## Context Utilities

### Data Store (Thread-Safe)

```go
c.Set("key", value)
c.Get("key")          // (any, bool)
c.GetString("key")
c.GetBool("key")
c.GetInt("key")
c.GetInt64("key")
c.GetTime("key")      // (time.Time, bool)
```

### Request Inspection

```go
c.Request()            // Underlying *http.Request
c.Context()            // Underlying context.Context
c.RealIP()             // Client IP (proxy-aware)
c.Path()               // Request path
c.ContentType()        // Content-Type header
c.Accept()             // Accept header values
c.AcceptLanguage()     // Accept-Language values
c.Referer()            // Referer header
c.IsWebSocketUpgrade() // WebSocket check
c.IsSSE()              // SSE check
c.Header("key")        // Single header
c.Headers()            // All headers
c.Logger()             // *slog.Logger for structured logging
c.Copy()               // Deep copy context (safe for goroutines)
```

### Path & Query Parameters

```go
c.PathParam("id") / c.Param("id")
c.Params()            // All path params
c.Query("key")
c.QueryArray("key")
c.QueryMap()
```

### Form Data

```go
c.Form("key")                    // Form field value
c.FormValue("key")               // Alias for Form
c.FormFile("key")                // (*multipart.FileHeader, error)
```

### Cookies

```go
c.Cookie("name")
c.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
```

### Middleware Flow

```go
c.Next()              // Call next middleware/handler in chain
```
