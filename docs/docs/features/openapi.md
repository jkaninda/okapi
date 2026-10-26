---
title: OpenAPI & Swagger
sidebar_position: 3
---

# OpenAPI & Swagger

Okapi provides **automatic OpenAPI documentation generation** with built-in interactive UIs. The documentation is generated from your route definitions, ensuring it stays in sync with your API implementation.

Okapi serves both **OpenAPI 3.1** (the default) and **OpenAPI 3.0**, and ships with three interactive UIs out of the box — **Swagger UI** (default), **ReDoc**, and **Scalar** — with the UI rendered at `/docs` fully selectable.

The document is built when documentation is enabled and again when the server
starts, so it reflects the routes registered up to that point. Routes that are
hidden, disabled, or belong to a disabled group are left out. Toggling a route or
group after the server is running changes what the router answers; it does not
rebuild the published document.

## Quick Start

### Using `okapi.Default()`

Documentation is enabled by default and served at `/docs`:

```go
o := okapi.Default() // Docs available at /docs
```

### Using `okapi.New()` with `WithOpenAPIDocs()`

If you initialize Okapi with `okapi.New()`, documentation is disabled by default. Enable it with `WithOpenAPIDocs()`:

```go
o := okapi.New()

if os.Getenv("ENABLE_DOCS") == "true" {
    o.WithOpenAPIDocs()
}
```

## Custom Configuration

Customize the OpenAPI documentation:

```go
o := okapi.New().WithOpenAPIDocs(
    okapi.OpenAPI{
        Title:   "Example API",
        Version: "1.0.0",
        Contact: okapi.Contact{
            Name:  "API Support",
            Email: "support@example.com",
        },
    },
)
```

`okapi.OpenAPI` also takes `Summary`, `Description`, `TermsOfService`,
`Servers`, `License`, `SecuritySchemes`, `UI`, `StrictDocUI`, `Favicon` (the
URL of the icon the documentation UIs use), `ExternalDocs` (a link to further
documentation) and `ComponentSchemas` (reusable schemas, as
[`RegisterSchemas`](#reusable-component-schemas) registers).

### Disabling documentation

`WithOpenAPIDisabled` turns documentation off, for example on an instance
created with `okapi.Default()`. The documentation and spec routes then answer
`404`:

```go
o := okapi.Default().WithOpenAPIDisabled()
// or: okapi.New(okapi.WithOpenAPIDisabled())
```

## Security Schemes

Define authentication mechanisms for your API:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:   "Okapi Web Framework Example",
    Version: "1.0.0",
    License: okapi.License{Name: "MIT"},
    SecuritySchemes: okapi.SecuritySchemes{
        {
            Name:   "basicAuth",
            Type:   "http",
            Scheme: "basic",
        },
        {
            Name:         "bearerAuth",
            Type:         "http",
            Scheme:       "bearer",
            BearerFormat: "JWT",
        },
        {
            Name: "OAuth2",
            Type: "oauth2",
            Flows: &okapi.OAuthFlows{
                AuthorizationCode: &okapi.OAuthFlow{
                    AuthorizationURL: "https://auth.example.com/authorize",
                    TokenURL:         "https://auth.example.com/token",
                    Scopes: map[string]string{
                        "read":  "Read access",
                        "write": "Write access",
                    },
                },
            },
        },
        {
            Name: "X-API-Key", // for apiKey, also the header/query/cookie name
            Type: "apiKey",
            In:   "header",
        },
    },
})
```

Each `okapi.SecurityScheme` is published under its `Name`. It has `Type`
(`http`, `apiKey`, `oauth2`), `Scheme` and `BearerFormat` for
`http`, `In` (`header`, `query`, `cookie`) for `apiKey`, `Flows` for `oauth2`,
and `Description`. `okapi.OAuthFlows` holds an `*okapi.OAuthFlow` for each of
`Implicit`, `Password`, `ClientCredentials` and `AuthorizationCode`; a flow has
`AuthorizationURL`, `TokenURL`, `RefreshURL` and `Scopes`.

## Applying Security to Routes

A security requirement maps scheme names to the scopes the route needs. It is a
`map[string][]string`, also available as `okapi.SecurityRequirement`. A scheme
not declared in `SecuritySchemes` is dropped from the route, with a warning.

### Single Route

```go
var bearerAuthSecurity = []map[string][]string{
    {"bearerAuth": {}},
}

o.Get("/books", getBooksHandler).WithSecurity(bearerAuthSecurity...)
o.Post("/books", createBookHandler).WithSecurity(okapi.SecurityRequirement{"OAuth2": {"write"}})
```

### Route Group

```go
api := o.Group("/api", jwtMiddleware).WithSecurity(bearerAuthSecurity)
api.Get("/", apiHandler)
```

### `DocBearerAuth` and `DocBasicAuth`

`okapi.DocBearerAuth()` (or `Doc().BearerAuth()`) and `okapi.DocBasicAuth()`
require the schemes named `BearerAuth` and `BasicAuth`. When `SecuritySchemes`
is empty, Okapi declares the schemes the routes use for you (a JWT bearer
token, HTTP basic, or both); otherwise declare them with those exact names
yourself.

```go
o.Get("/admin", adminHandler, okapi.DocBasicAuth())
```

## Documenting Routes

Okapi offers multiple ways to document your routes.

### 1. Composable Functions (Direct Style)

Simple and readable approach for small to medium routes:

```go
o.Get("/books", getBooksHandler,
    okapi.Summary("List all available books"),
    okapi.Tags("Books"),
    okapi.DocQueryParam("author", "string", "Filter by author name", false),
    okapi.DocQueryParam("limit", "int", "Maximum results to return", false),
    okapi.DocResponseHeader("X-Client-Id", "string", "Client ID"),
    okapi.DocResponse([]Book{}),
    okapi.DocResponse(400, ErrorResponse{}),
    okapi.DocResponse(401, ErrorResponse{}),
)
```

`okapi.DocResponse(v)` with a single argument always documents the **200**
response. Pass the status code explicitly — `okapi.DocResponse(201, v)` — for
anything else.

### 2. Fluent Builder Style

For complex or dynamic documentation:

```go
o.Post("/books", createBookHandler,
    okapi.Doc().
        Summary("Add a new book to the inventory").
        Tags("Books").
        BearerAuth().
        ResponseHeader("X-Client-Id", "string", "Client ID").
        RequestBody(BookRequest{}).
        Response(201, Book{}).
        Response(400, ErrorResponse{}).
        Response(401, ErrorResponse{}).
        Build(),
)
```

`Build()` merges the chain into a single `RouteOption`; `AsOption()` is the same
thing under another name.

### 3. Body Field Style

Using structs with a field named `Body`:

```go
type BookRequest struct {
    Body struct {
        Name  string `json:"name" minLength:"4" maxLength:"50" required:"true"`
        Price int    `json:"price" required:"true"`
    }

    ID     int    `param:"id" query:"id"`
    APIKey string `header:"X-API-Key" required:"true"`
}

o.Post("/books", createBookHandler,
    okapi.Request(&BookRequest{}),
    okapi.Response(&BookResponse{}),
)
```

Fields tagged `path`/`param`, `query`, `header` and `cookie` become parameters
of that kind. Their schema follows the field's Go type (`[]string` is an array,
`*int` an integer, `time.Time` a `date-time` string) and carries the same
constraint tags as body fields: `enum`, `min`, `max`, `minLength`, `pattern`,
`format`, `default`, `example` and so on. A parameter is required when it has
`required:"true"` or a `validate:"required"` rule; path parameters always are.
A pointer `Body` documents an optional request body.

`okapi.Request` and `okapi.Response` shape the OpenAPI document. They do **not**
bind or validate the request at runtime: the validation tags on `BookRequest`
take effect where the handler binds it, with `c.Bind(&in)` or through
`okapi.Handle` / `okapi.H`.

On a response struct, the `Status` field sets the status code the body and
headers are documented under, and a `description` tag on it describes that
response (by default the status text, such as `Created`). Header fields apply to
that status only.

```go
type CreateBookResponse struct {
    Status   int    `description:"Book created"`
    Location string `header:"Location" description:"URL of the new book"`
    Body     Book
}

o.Post("/books", createBookHandler, okapi.Response(CreateBookResponse{Status: 201}))
```

To describe a response, or document one without a body such as a `204`:

```go
okapi.Doc().
    ResponseDescription(204, "Book deleted").
    Response(404, ErrorResponse{}).
    ResponseHeaderFor(429, "Retry-After", "int", "Seconds to wait")
```

`DocResponseHeader` / `Doc().ResponseHeader()` adds a header to every documented
response; `ResponseHeaderFor` scopes it to one status. Each route also documents
a `500` response unless it declares its own.

### Forms and file uploads

A request whose fields carry `form` tags is documented as a form instead of
JSON. With a file field (`*multipart.FileHeader`, `[]*multipart.FileHeader` or
`multipart.File`) the body is `multipart/form-data` and files appear as
`type: string, format: binary`; without one it is
`application/x-www-form-urlencoded` or `multipart/form-data`.

```go
type UploadRequest struct {
    Title string                `form:"title" validate:"required"`
    File  *multipart.FileHeader `form:"file" description:"Document to upload"`
    Lang  string                `query:"lang"`
}

o.Post("/uploads", uploadHandler, okapi.Request(&UploadRequest{}))
```

Top-level `form` fields are only documented as a body on methods that carry
one: a `GET` or `HEAD` request reads its form values from the query string.

To document other media types, override the inferred ones:

```go
o.Post("/books", handler, okapi.Doc().
    RequestBody(Book{}).
    RequestContentType("application/json", "application/xml").
    RequestExample(Book{Name: "Dune", Price: 20}).
    Build(),
)
```

### Using `.WithIO()`, `.WithInput()`, `.WithOutput()`

```go
// Both request & response
o.Post("/books", handler).WithIO(&BookRequest{}, &BookResponse{})

// Request only
o.Post("/books", handler).WithInput(&BookRequest{})

// Response only
o.Get("/books", handler).WithOutput(&BooksResponse{})
```

### Typed routes

`okapi.Get`, `Post`, `Put`, `Patch` and `Delete` register a typed handler, as
`okapi.HandleIO` does, and document its input and output types, so they are not
repeated in `Request` and `Response` options. They take an `*Okapi` or a
`*Group`:

```go
func createBook(c *okapi.Context, in *CreateBookInput) (*BookOutput, error) { ... }

okapi.Post(o, "/books", createBook, okapi.Summary("Create a book"))
okapi.Get(api, "/books/{id}", getBook) // api is a *okapi.Group
```

What the options document is kept: a `Request` replaces the inferred input, and
a 2xx response, such as `okapi.Response(BookOutput{Status: 201})`, replaces the
inferred 200.

Routes served by `okapi.Handle`, `okapi.H`, `okapi.HandleIO` or a typed route
also document the `400` response a binding or validation failure gets, unless
they declare their own. Its schema follows the error handler: `ErrorResponse`
by default, Problem Details with `WithProblemDetailErrorHandler`, and no body
with a custom `WithErrorHandler`, whose output is not known.

### Examples

```go
okapi.Doc().
    RequestBody(Book{}).
    RequestExamples(map[string]any{
        "paperback": Book{Name: "Dune", Price: 10},
        "hardcover": Book{Name: "Dune", Price: 30},
    }).
    Response(201, Book{}).
    ResponseExample(201, Book{Name: "Dune", Price: 10}).
    Response(404, ErrorResponse{}).
    ResponseExamples(404, map[string]any{"missing": ErrorResponse{Message: "no such book"}})
```

A single example is the media type's `example`; named ones are listed under
`examples`. With both, the single one is listed as `default`.

### Parameter defaults and enums

`DocPathParamWithDefault`, `DocQueryParamWithDefault` and `DocHeaderWithDefault`
(and the builder's `PathParamWithDefault`, `QueryParamWithDefault` and
`HeaderWithDefault`) take a last argument that sets the parameter's `default`.
With the type `"enum"` and a slice, it lists the allowed values instead. An
`*openapi3.SchemaRef` replaces the schema altogether, and the type is ignored:

```go
o.Get("/books", listBooksHandler,
    okapi.DocQueryParamWithDefault("limit", "int", "Page size", false, 20),
    okapi.DocQueryParamWithDefault("sort", "enum", "Sort order", false, []string{"asc", "desc"}),
    okapi.DocHeaderWithDefault("X-Region", "string", "Region", false,
        openapi3.NewSchemaRef("", openapi3.NewStringSchema().WithPattern("^[a-z]{2}$"))),
)
```

The type names a parameter can take are `string`, `int` / `integer`, `int64`,
`float` / `float32`, `float64` / `double`, `bool` / `boolean`, `uuid`, `date`
and `date-time` / `datetime`; anything else is documented as a string.

### 4. Declarative Route Definition

With `okapi.RouteDefinition`, documentation lives right next to the route. Common
fields — `Summary`, `Description`, `Tags`, `Request`, `Response`, and `Security` —
are set directly on the struct:

```go
routes := []okapi.RouteDefinition{
    {
        Method:      http.MethodPost,
        Path:        "/books",
        Handler:     createBookHandler,
        Summary:     "Add a new book",
        Description: "Create a new book in the inventory",
        Tags:        []string{"Books"},
        Request:     &BookRequest{},      // documents the request body + params
        Response:    &Book{},             // documents the 200 response schema
        Security: []map[string][]string{  // requires bearer auth
            {"bearerAuth": {}},
        },
    },
}

app := okapi.New()
okapi.RegisterRoutes(app, routes)
```

`Request` documents the request; it does not validate it. Bind inside the handler
(`c.Bind`) or wrap the handler in `okapi.Handle` to get validation.

For anything the struct fields don't cover (extra status codes, headers, query
params, …), drop down to the `Options` field with the same `Doc*` helpers used
elsewhere. Struct fields and `Options` can be mixed freely:

```go
routes := []okapi.RouteDefinition{
    {
        Method:  http.MethodGet,
        Path:    "/books/{id:int}",
        Handler: getBookHandler,
        Tags:    []string{"Books"},
        Options: []okapi.RouteOption{
            okapi.Summary("Get a book by ID"),
            okapi.DocPathParam("id", "int", "The ID of the book"),
            okapi.DocResponse(Book{}),                       // 200
            okapi.DocResponse(404, ErrorResponse{}),         // 404
            okapi.DocResponseHeader("X-Request-Id", "string", "Request ID"),
        },
    },
}
```

Attach routes to a group to share a prefix, tags, middleware, and security across
several definitions:

```go
books := &okapi.Group{Prefix: "/api/v1", Tags: []string{"Books"}}

routes := []okapi.RouteDefinition{
    {
        Method:   http.MethodGet,
        Path:     "/books",
        Handler:  listBooksHandler,
        Group:    books,
        Summary:  "List all books",
        Response: &BooksResponse{},
    },
    {
        Method:   http.MethodPost,
        Path:     "/books",
        Handler:  createBookHandler,
        Group:    books,
        Summary:  "Add a new book",
        Request:  &BookRequest{},
        Response: &Book{},
    },
}

app.Register(routes...) // or okapi.RegisterRoutes(app, routes)
```

`RouteDefinition.Method` accepts every standard verb plus `ANY` (or `"*"`), which
registers a route matching every method.

## Available Documentation Options

| Option                                                         | Description                                    |
|----------------------------------------------------------------|------------------------------------------------|
| `Summary()` / `Doc().Summary()`                                | Short endpoint summary                         |
| `Description()` / `Doc().Description()`                        | Long endpoint description                      |
| `OperationId()` / `Doc().OperationId()`                        | Set the `operationId`                          |
| `Tag()`                                                        | Add one tag                                    |
| `Tags()` / `Doc().Tags()`                                      | Group endpoints under tags                     |
| `Deprecated()` / `Doc().Deprecated()`                          | Mark route as deprecated                       |
| `Hide()` / `Doc().Hide()`                                      | Leave the route out of the document            |
| `DocBearerAuth()` / `Doc().BearerAuth()`                       | Require the `BearerAuth` scheme                |
| `DocBasicAuth()`                                               | Require the `BasicAuth` scheme                 |
| `Request()`                                                    | Document params and body from a struct         |
| `Response()`                                                   | Document a response from a struct              |
| `WithIO()`                                                     | `Request()` and `Response()` in one call       |
| `DocRequestBody()` / `Doc().RequestBody()`                     | Document request body schema                   |
| `DocRequestContentType()` / `Doc().RequestContentType()`       | Set the request body media types               |
| `DocRequestExample()` / `Doc().RequestExample()`               | Add an example of the request body             |
| `DocRequestExamples()` / `Doc().RequestExamples()`             | Add named examples of the request body         |
| `DocResponse()` / `Doc().Response()`                           | Document response schema or status codes       |
| `DocResponseExample()` / `Doc().ResponseExample()`             | Add an example of a response                   |
| `DocResponseExamples()` / `Doc().ResponseExamples()`           | Add named examples of a response               |
| `DocResponseDescription()` / `Doc().ResponseDescription()`     | Describe the response for a status             |
| `DocResponseHeader()` / `Doc().ResponseHeader()`               | Document a header on every response            |
| `DocResponseHeaderFor()` / `Doc().ResponseHeaderFor()`         | Document a header on one status                |
| `DocPathParam()` / `Doc().PathParam()`                         | Document path parameters                       |
| `DocPathParamWithDefault()` / `Doc().PathParamWithDefault()`   | Path parameter with a default, enum or schema  |
| `DocQueryParam()` / `Doc().QueryParam()`                       | Document query parameters                      |
| `DocQueryParamWithDefault()` / `Doc().QueryParamWithDefault()` | Query parameter with a default, enum or schema |
| `DocHeader()` / `Doc().Header()`                               | Document request headers                       |
| `DocHeaderWithDefault()` / `Doc().HeaderWithDefault()`         | Request header with a default, enum or schema  |
| `Doc().Build()` / `Doc().AsOption()`                           | Turn the builder into a `RouteOption`          |

## Schema Generation Details

A few things worth knowing about how schemas are derived from your Go types:

- **`example` values carry the field's JSON type.** `example:"1"` on an `int`
  emits `1`, not `"1"`, and a boolean emits `true`. Slice, map, struct and `any`
  fields take the tag as JSON when it is valid JSON. A value that does not parse
  for its type stays a string.
- **`enum` values are always strings.** Whatever the field's Go type, the enum
  entries are emitted as strings — and `enum` is only validated on `string` and
  `[]string` fields at runtime. See [Validation](validation.md#enum-and-const-are-string-only).
- **Embedded structs follow `encoding/json`'s rules.** Fields promoted from an
  unexported embedded struct are included, an embedded struct with a `json` name
  is nested rather than flattened, `json:"-"` on an embedded struct is honoured,
  and a shadowed name is listed once in `required`.
- **Recursive types are supported.** A type that refers to itself in a documented
  request or response is registered as a component and referenced, instead of
  being expanded forever.
- **Generic types are named after their arguments.** `Page[models.Book]` is
  registered as the `PageBook` component, without the package path.
- **A type can describe its own schema** by implementing `okapi.SchemaProvider`,
  for a decimal, a custom UUID or a `oneOf` union. Field tags still apply on top:

  ```go
  func (Decimal) OpenAPISchema() *openapi3.Schema {
      return &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "decimal"}
  }
  ```
- **Every operation has an `operationId`.** One is derived from the summary, or
  else from the method and path (`get-books-id`), and made unique with a
  numeric suffix. A declared `operationId` is kept as is.
- **`Any` routes** are documented under `get`, `post`, `put`, `patch` and
  `delete`. A CORS preflight for such a route lists real method names in
  `Access-Control-Allow-Methods` rather than `*`.

## Reusable Component Schemas

`RegisterSchemas` adds schemas to `components.schemas` under the given names.
Each `okapi.SchemaInfo` carries the `Schema` (an `*openapi3.SchemaRef`) and an
optional `TypeName`, which defaults to the name. Registering a name twice
returns an error.

```go
err := o.RegisterSchemas(map[string]*okapi.SchemaInfo{
    "Money": {Schema: openapi3.NewSchemaRef("", openapi3.NewObjectSchema().
        WithProperty("amount", openapi3.NewFloat64Schema()).
        WithProperty("currency", openapi3.NewStringSchema()))},
})
```

A generated object schema that matches a registered one is replaced by a
`$ref` to it. To reference one explicitly, use
`&openapi3.SchemaRef{Ref: "#/components/schemas/Money"}`, for example as the
last argument of `DocQueryParamWithDefault`. Register schemas before the server
starts, since the document is built then.

## Choosing the Documentation UI

Okapi ships with three interactive UIs: **Swagger UI** (default), **ReDoc**, and **Scalar**.
They are values of the `okapi.DocUI` type: `okapi.SwaggerUI` (`"swagger"`),
`okapi.RedocUI` (`"redoc"`) and `okapi.ScalarUI` (`"scalar"`).
The UI rendered at `/docs` is selectable with the `UI` field on `okapi.OpenAPI`:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title: "My API",
    UI:    okapi.ScalarUI, // okapi.SwaggerUI (default) | okapi.RedocUI | okapi.ScalarUI
})
```

…or with `WithDocUI`, as a chainable method or an option:

```go
o := okapi.New().WithOpenAPIDocs().WithDocUI(okapi.ScalarUI)
// or: okapi.New(okapi.WithDocUI(okapi.RedocUI)).WithOpenAPIDocs()
```

An unset or unrecognised value falls back to Swagger UI.

### `StrictDocUI`: the dedicated UI routes

`/docs` always renders the selected UI. Whether the three **dedicated** routes
`/swagger`, `/redoc` and `/scalar` answer as well is controlled by `StrictDocUI`:

* `StrictDocUI: false` — all three stay reachable, whichever UI `/docs` renders.
* `StrictDocUI: true` — **all three return `404`**, including the one you selected.
  Only `/docs` serves a UI. (The routes are registered either way; when strict,
  they answer `404`.)

The default depends on how you create the instance and how you enable docs:

| Setup                                        | `StrictDocUI` | `/swagger`, `/redoc`, `/scalar` |
|----------------------------------------------|---------------|---------------------------------|
| `okapi.Default()`                            | `false`       | reachable                       |
| `okapi.New().WithOpenAPIDocs()`              | `true`        | `404`                           |
| `okapi.New().WithOpenAPIDocs(okapi.OpenAPI{…})` | taken from the config (so `false` unless you set it) | follows the config |

Passing a config always overwrites `StrictDocUI` with the config's value, so an
`okapi.OpenAPI{}` literal that omits the field turns strict mode **off**:

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:       "My API",
    UI:          okapi.ScalarUI,
    StrictDocUI: true, // only /docs serves a UI; /swagger, /redoc and /scalar all 404
})
```

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:       "My API",
    UI:          okapi.ScalarUI,
    StrictDocUI: false, // /swagger, /redoc and /scalar all stay reachable
})
```

`StrictDocUI` is read per request, so it never affects the spec endpoints
(`/openapi.json` and friends), which are always served while documentation is
enabled.

## OpenAPI 3.1 and 3.0

Okapi serves the same API description as both **OpenAPI 3.1 / JSON Schema 2020-12** and **OpenAPI 3.0**.
The default endpoints (`/openapi.json`, `/openapi.yaml`) serve **3.1**, and the documentation UIs render it.
The 3.0 document remains available at `/openapi-3.0.{json,yaml}`, so 3.0-only consumers stay supported.

Both documents are derived from the same base, and the 3.1 one adds these 3.1 features:

- **Type-array nullability** — pointer fields render as `nullable: true` in 3.0 and as
  `type: ["string", "null"]` in 3.1.
- **`jsonSchemaDialect`** — set to the JSON Schema 2020-12 base dialect on the 3.1 document.
- **SPDX license identifier** — set `License.Identifier` (e.g. `"Apache-2.0"`); it appears only on
  the 3.1 document and is mutually exclusive with `License.URL`.
- **`const`** — the `const:"value"` struct tag becomes a JSON Schema `const` on the 3.1 document,
  on every build, and never leaks into the 3.0 document.
- **Webhooks** — declare outbound callbacks with `o.Webhook(...)`; they appear under the `webhooks`
  field of the 3.1 document only.

```go
o.WithOpenAPIDocs(okapi.OpenAPI{
    Title:   "Example API",
    Version: "1.0.0",
    License: okapi.License{Name: "Apache 2.0", Identifier: "Apache-2.0"},
})

// A webhook is documentation-only: it is not added to the router.
o.Webhook("newBook", http.MethodPost,
    okapi.Summary("Notifies subscribers about a newly added book"),
    okapi.DocRequestBody(Book{}),
    okapi.DocResponse(200, okapi.M{"received": true}),
)
```

## Validating and Exporting the Document

The server validates its OpenAPI documents on startup and logs a warning for
problems such as an undeclared path parameter or a duplicate `operationId`.
To fail a build instead, call `ValidateOpenAPI` from a test:

```go
func TestOpenAPI(t *testing.T) {
    if err := newApp().ValidateOpenAPI(); err != nil {
        t.Fatal(err)
    }
}
```

`OpenAPISpec()` and `OpenAPISpec30()` build the 3.1 and 3.0 documents from the
routes registered so far. To write the document without starting the server,
for a client generator or to check it in, see the
[CLI `openapi` command](cli.md#export-the-openapi-document); to catch unintended
API changes in review, see
[golden OpenAPI tests](testing.md#golden-openapi-document).

## Accessing Documentation

| Route               | Content                                          |
|---------------------|--------------------------------------------------|
| `/docs`             | The selected UI (Swagger UI by default)          |
| `/swagger`          | Swagger UI — `404` when `StrictDocUI` is true     |
| `/redoc`            | ReDoc — `404` when `StrictDocUI` is true          |
| `/scalar`           | Scalar API Reference — `404` when `StrictDocUI` is true |
| `/openapi.json`     | OpenAPI spec (JSON) — **3.1 by default**          |
| `/openapi.yaml`     | OpenAPI spec (YAML) — **3.1 by default**          |
| `/openapi-3.0.json` | OpenAPI **3.0** spec (JSON)                       |
| `/openapi-3.0.yaml` | OpenAPI **3.0** spec (YAML)                       |

![Swagger UI](https://raw.githubusercontent.com/jkaninda/okapi/main/swagger.png)

![Redoc](https://raw.githubusercontent.com/jkaninda/okapi/main/redoc.png)
![Scalar](https://raw.githubusercontent.com/jkaninda/okapi/main/scalar.png)
