---
title: Okapi vs Huma
layout: default
parent: Comparison
nav_order: 1
---

# Okapi vs Huma

Both **[Okapi](https://github.com/jkaninda/okapi)** and **[Huma](https://github.com/danielgtaylor/huma)** aim to improve developer experience in Go APIs with strong typing and OpenAPI integration. The key difference is **philosophy**: Okapi is a *batteries-included web framework*, while Huma is an *API layer designed to sit on top of existing routers*.

| Feature / Aspect             | **Okapi**                                                              | **Huma**                                                       |
|------------------------------|------------------------------------------------------------------------|----------------------------------------------------------------|
| **Positioning**              | Full web framework                                                     | API framework built on top of existing routers                 |
| **Router**                   | Built-in high-performance router                                       | Uses external routers (Chi, httprouter, Fiber, etc.)           |
| **OpenAPI Generation**       | Native, framework-level (Swagger UI & Redoc included)                  | Native, schema-first API design                                |
| **Request Binding**          | Unified binder for JSON, XML, forms, query, headers, path params       | Struct tags + resolver pattern for headers, query, path params |
| **Validation**               | Tag-based (min, max, enum, required, default, pattern, etc.)           | Included                                                       |
| **Response Modeling**        | Output structs with `Body` pattern; headers & status via struct fields | Strongly typed response models with similar patterns           |
| **Middleware**               | Built-in + custom middleware, groups, per-route middleware             | Router middleware + Huma-specific middleware and transformers  |
| **Authentication**           | Built-in JWT, Basic Auth, security schemes for OpenAPI                 | Security schemes via OpenAPI; middleware via router            |
| **Dynamic Route Management** | Enable/disable routes & groups at runtime                              | Not a core feature                                             |
| **Templating / HTML**        | Built-in rendering (HTML templates, static files)                      | API-focused; not intended for HTML apps                        |
| **CLI Integration**          | Built-in CLI support (flags, env config)                               | Included                                                       |
| **Testing Utilities**        | Built-in test server and fluent HTTP assertions                        | Relies on standard Go testing tools                            |
| **Learning Curve**           | Very approachable for Go web developers                                | Slightly steeper (requires OpenAPI-first mental model)         |
| **Use Case Fit**             | Full web apps, APIs, gateways, microservices                           | Pure API services, schema-first API design                     |
| **Philosophy**               | "FastAPI-like DX for Go, batteries included"                           | "OpenAPI-first typed APIs on top of your router of choice"     |

---

## Quick Comparison

**Okapi** — define a route with built-in validation and OpenAPI metadata:

```go
app:=okapi.Default()
app.Register(okapi.RouteDefinition{
     Method:      http.MethodPost,
     Path:        "/users",
     Handler:     createUser,
     OperationId: "create-user",
     Summary:     "Create a new user", 
     Tags: []string{"users"},
     Request: &UserRequest{},
     Response:    &User{},
})
```

**Huma** — similar concept, different style:

```go
huma.Register(api, huma.Operation{
    OperationID: "create-user",
    Method:      http.MethodPost,
    Path:        "/users",
    Summary:     "Create a new user",
    Tags:        []string{"Users"},
}, createUser)
```

Both approaches generate OpenAPI documentation automatically.

---

## When to Choose Which?

### Choose Okapi if you want:

- A **batteries-included web framework** with routing, middleware, auth, OpenAPI, templates, and CLI in one cohesive package
- **FastAPI-like developer experience** that feels idiomatic in Go
- **Dynamic route control** — enable or disable routes and groups at runtime
- To build APIs **and** serve HTML pages or static assets from the same application

### Choose Huma if you want:

- A **schema-first, OpenAPI-driven API layer** where the spec drives your implementation
- To **keep using your existing router** (Chi, Fiber, Echo, etc.) without adopting a new framework
- **Strict typed request/response contracts** as your primary design model
- A **minimal, API-only stack** without broader web framework concerns

---

## Community & Maturity

- **Huma**: More established with a larger community and extensive production usage
- **Okapi**: Newer and rapidly evolving, with a smaller but growing community

Both are actively maintained. Choose based on your architectural preferences and project needs rather than stability concerns alone.

> **Note**: If you're already using Huma with Chi or another router and it's working well for you, there's no urgent reason to switch. Okapi is ideal for new projects or when you want a more integrated, batteries-included framework experience.

