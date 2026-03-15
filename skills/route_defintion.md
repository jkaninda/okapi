## Okapi Route Definition Skills

Okapi's `RouteDefinition` is a declarative alternative to imperative route registration (`.Get()`, `.Post()`, etc.). Instead of chaining method calls, you describe each route as a struct and register it with `app.Register()`. This keeps route metadata — method, path, handler, documentation, middleware — in one place, making routes easier to read, move between files, and generate programmatically.

### RouteDefinition Struct

| Field         | Type                          | Description                                                  |
|---------------|-------------------------------|--------------------------------------------------------------|
| `Method`      | `string`                      | HTTP method (`http.MethodGet`, `http.MethodPost`, etc.)      |
| `Path`        | `string`                      | Route path relative to its group (e.g. `"/books/{id:int}"`)  |
| `Handler`     | `okapi.HandlerFunc`           | Handler function — use `okapi.H()` for generic input binding |
| `Group`       | `*okapi.Group`                | Group this route belongs to (prefix, middleware, tags)        |
| `OperationId` | `string`                      | OpenAPI operation ID                                         |
| `Summary`     | `string`                      | Short description for OpenAPI docs                           |
| `Description` | `string`                      | Detailed description for OpenAPI docs                        |
| `Tags`        | `[]string`                    | OpenAPI tags (overrides group tags if set)                   |
| `Request`     | `any`                         | Request body schema for OpenAPI docs                         |
| `Response`    | `any`                         | Success response schema (defaults to 200)                    |
| `Security`    | `[]map[string][]string`       | OpenAPI security requirements                                |
| `Options`     | `[]okapi.RouteOption`         | Additional route options (error responses, path params, etc.)|
| `Middlewares` | `[]okapi.Middleware`          | Per-route middleware                                         |

### Declarative vs Imperative

**Imperative** (traditional):
```go
group.Post("/books", okapi.H(handler.Create),
    okapi.DocTags("Books"),
    okapi.DocSummary("Create a book"),
    okapi.Request(&CreateBookRequest{}),
    okapi.DocResponse(201, &BookResponse{}),
    okapi.DocErrorResponse(409, &ErrorResponse{}),
)
```

**Declarative** (RouteDefinition):
```go
app.Register(okapi.RouteDefinition{
    Method:  http.MethodPost,
    Path:    "/books",
    Handler: okapi.H(handler.Create),
    Group:   bookGroup,
    Summary: "Create a book",
    Request: &CreateBookRequest{},
    Options: []okapi.RouteOption{
        okapi.DocResponse(201, &BookResponse{}),
        okapi.DocErrorResponse(409, &ErrorResponse{}),
    },
})
```

### Registration

```go
// Single route
app.Register(routeDef)

// Multiple routes (variadic)
app.Register(routeDefs...)
```

### Using Struct Fields vs Options

Use struct fields for common metadata — they're cleaner and more readable:

| Use struct field          | Use `Options` for                              |
|---------------------------|-------------------------------------------------|
| `Summary`, `Description`  | `okapi.DocPathParam(...)` — path parameters     |
| `Tags`                    | `okapi.DocResponse(201, ...)` — non-200 status  |
| `Request`                 | `okapi.DocResponse(204, nil)` — no-content      |
| `Response` (200 only)     | `okapi.DocErrorResponse(...)` — error responses  |
| `Security`                | `okapi.DocHide()` — hide from docs              |
| `Middlewares`             | `okapi.DocQueryParam(...)` — query parameters   |

### Group-Level Tags with `.WithTags()`

Use `.WithTags()` on groups so routes inherit tags automatically — no need to repeat `Tags` on every route. Only set `Tags` on a route when it needs to override the group default.

```go
// All routes in this group inherit the "Books" tag
bookGroup := app.Group("/api/v1/books", authMiddleware).WithTags([]string{"Books"})
bookGroup.WithBearerAuth()

app.Register(
    // Inherits "Books" tag from group
    okapi.RouteDefinition{
        Method:   http.MethodGet,
        Path:     "",
        Handler:  bookService.List,
        Group:    bookGroup,
        Summary:  "List books",
        Response: &BooksResponse{},
    },
    // Overrides group tag
    okapi.RouteDefinition{
        Method:  http.MethodGet,
        Path:    "/categories",
        Handler: categoryService.List,
        Group:   bookGroup,
        Tags:    []string{"Categories"},
        Summary: "List categories",
    },
)
```

### Organizing Routes in a Large Project

For large projects with many routes, use a `Router` struct and split route definitions across multiple files by domain. Each file contains methods that return `[]okapi.RouteDefinition`.

**File structure:**
```
internal/routes/
├── routes.go          # Router struct, InitRoutes, registerRoutes orchestrator
├── auth_routes.go     # Auth & public API routes
├── user_routes.go     # User-scoped routes
└── admin_routes.go    # Admin routes
```

**Router struct** (in `routes.go`):
```go
type Router struct {
    app *okapi.Okapi
    cfg *config.Config
    v1  *okapi.Group

    // Auth middleware
    jwtAuth      okapi.JWTAuth
    jwtAdminAuth okapi.JWTAuth

    // Handlers
    userHandler  *handlers.UserHandler
    bookHandler  *handlers.BookHandler
    adminHandler *handlers.AdminHandler
}

func InitRoutes(app *okapi.Okapi, db *gorm.DB, cfg *config.Config) {
    // Initialize repositories, services, handlers...

    r := &Router{
        app:          app,
        cfg:          cfg,
        v1:           app.Group("/api/v1"),
        jwtAuth:      middlewares.JWTAuth(cfg),
        jwtAdminAuth: middlewares.JWTAdminAuth(cfg),
        userHandler:  handlers.NewUserHandler(userRepo),
        bookHandler:  handlers.NewBookHandler(bookRepo),
        adminHandler: handlers.NewAdminHandler(db),
    }
    r.registerRoutes()
}

func (r *Router) registerRoutes() {
    r.app.Use(okapi.RequestID())

    r.app.Register(r.authRoutes()...)
    r.app.Register(r.userRoutes()...)
    r.app.Register(r.adminRoutes()...)

    // Special routes that don't fit RouteDefinition
    r.app.Static("/assets", "public/assets")
    r.app.NoRoute(spaFallback)
}
```

**Domain-specific route files** (e.g. `auth_routes.go`):
```go
func (r *Router) authRoutes() []okapi.RouteDefinition {
    authGroup := r.v1.Group("/auth").WithTags([]string{"Auth"})

    return []okapi.RouteDefinition{
        {
            Method:   http.MethodPost,
            Path:     "/login",
            Handler:  okapi.H(r.userHandler.Login),
            Group:    authGroup,
            Summary:  "Login",
            Request:  &LoginRequest{},
            Response: &AuthResponse{},
        },
    }
}
```

**Key principles:**
- One `Router` struct holds all handlers, middleware, and config
- Each file returns `[]okapi.RouteDefinition` from methods on `Router`
- Groups are created inside each method with `.WithTags()` to avoid tag repetition
- `registerRoutes()` orchestrates all registration via `app.Register()`
- Conditional routes (dev mode, optional features) use `if` guards in `registerRoutes()`
- Special registrations (static files, NoRoute, global middleware) stay imperative

### Common Patterns

**CRUD resource** — a typical set of routes for a resource:
```go
func (r *Router) bookRoutes() []okapi.RouteDefinition {
    g := r.v1.Group("/books", r.jwtAuth.Middleware).WithTags([]string{"Books"})
    g.WithBearerAuth()

    return []okapi.RouteDefinition{
        {Method: http.MethodGet, Path: "", Handler: okapi.H(r.bookHandler.List), Group: g,
            Summary: "List books", Request: &ListRequest{}, Response: &PageableResponse[Book]{}},
        {Method: http.MethodGet, Path: "/{id:int}", Handler: okapi.H(r.bookHandler.Get), Group: g,
            Summary: "Get book", Response: &Response[Book]{},
            Options: []okapi.RouteOption{okapi.DocPathParam("id", "integer", "Book ID"), okapi.DocErrorResponse(404, &ErrorResponse{})}},
        {Method: http.MethodPost, Path: "", Handler: okapi.H(r.bookHandler.Create), Group: g,
            Summary: "Create book", Request: &CreateBookRequest{},
            Options: []okapi.RouteOption{okapi.DocResponse(201, &Response[Book]{}), okapi.DocErrorResponse(409, &ErrorResponse{})}},
        {Method: http.MethodPut, Path: "/{id:int}", Handler: okapi.H(r.bookHandler.Update), Group: g,
            Summary: "Update book", Request: &UpdateBookRequest{}, Response: &Response[Book]{},
            Options: []okapi.RouteOption{okapi.DocPathParam("id", "integer", "Book ID"), okapi.DocErrorResponse(404, &ErrorResponse{})}},
        {Method: http.MethodDelete, Path: "/{id:int}", Handler: okapi.H(r.bookHandler.Delete), Group: g,
            Summary: "Delete book",
            Options: []okapi.RouteOption{okapi.DocPathParam("id", "integer", "Book ID"), okapi.DocResponse(204, nil), okapi.DocErrorResponse(404, &ErrorResponse{})}},
    }
}
```

**Conditional routes** — only registered when a feature is enabled:
```go
func (r *Router) registerRoutes() {
    r.app.Register(r.publicRoutes()...)
    r.app.Register(r.userRoutes()...)

    if r.cfg.DevMode {
        r.app.Register(r.devRoutes()...)
    }
}
```

**Optional handler** — append to the slice when a dependency is available:
```go
func (r *Router) adminRoutes() []okapi.RouteDefinition {
    g := r.v1.Group("/admin", r.jwtAdminAuth.Middleware).WithTags([]string{"Admin"})
    g.WithBearerAuth()

    routes := []okapi.RouteDefinition{
        // ... standard admin routes
    }

    if r.cronHandler != nil {
        routes = append(routes, okapi.RouteDefinition{
            Method:  http.MethodGet,
            Path:    "/jobs",
            Handler: r.cronHandler.List,
            Group:   g,
            Summary: "List scheduled jobs",
            Response: &Response[[]JobStatus]{},
        })
    }

    return routes
}
```
