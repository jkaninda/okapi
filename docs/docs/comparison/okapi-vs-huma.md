---
title: Okapi vs Huma
sidebar_position: 1
---

# Okapi vs Huma

Both **[Okapi](https://github.com/jkaninda/okapi)** and **[Huma](https://github.com/danielgtaylor/huma)** aim to improve developer experience in Go APIs with strong typing and OpenAPI integration. The key difference is **philosophy**: Okapi is a *batteries-included web framework*, while Huma is an *API layer designed to sit on top of existing routers*.

| Feature / Aspect             | **Okapi**                                                                     | **Huma**                                                       |
|------------------------------|-------------------------------------------------------------------------------|----------------------------------------------------------------|
| **Positioning**              | Full web framework                                                            | API framework built on top of existing routers                 |
| **Router**                   | Built-in high-performance router                                              | Uses external routers (Chi, httprouter, Fiber, etc.)           |
| **OpenAPI Generation**       | Native, framework-level; 3.1 and 3.0 documents, Swagger UI, ReDoc and Scalar included | Native, schema-first API design                                |
| **Request Binding**          | Unified binder for JSON, XML, YAML, protobuf, forms, query, headers, cookies, path params | Struct tags + resolver pattern for headers, query, path params |
| **Validation**               | Tag-based (min, max, enum, required, default, pattern, etc.), applied when the handler binds | Included                                                       |
| **Response Modeling**        | Output structs with `Body` pattern; headers & status via struct fields         | Strongly typed response models with similar patterns           |
| **Middleware**               | Built-in + custom middleware, groups, per-route middleware                    | Router middleware + Huma-specific middleware and transformers  |
| **Authentication**           | Built-in JWT, Basic Auth, security schemes for OpenAPI                         | Security schemes via OpenAPI; middleware via router            |
| **Dynamic Route Management** | Enable/disable routes & groups; the router honours the change immediately      | Not a core feature                                             |
| **Templating / HTML**        | Built-in rendering (HTML templates, static files, SPA hosting)                 | API-focused; not intended for HTML apps                        |
| **CLI Integration**          | Built-in CLI support (flags, env config)                                       | Included                                                       |
| **Testing Utilities**        | Built-in test server and fluent HTTP assertions                                | Relies on standard Go testing tools                            |
| **Learning Curve**           | Very approachable for Go web developers                                        | Slightly steeper (requires OpenAPI-first mental model)         |
| **Use Case Fit**             | Full web apps, APIs, gateways, microservices                                    | Pure API services, schema-first API design                     |
| **Philosophy**               | "FastAPI-like DX for Go, batteries included"                                   | "OpenAPI-first typed APIs on top of your router of choice"     |

## Quick Comparison

**Okapi** — define a route with OpenAPI metadata declared next to it:

```go
app := okapi.Default()
app.Register(okapi.RouteDefinition{
    Method:      http.MethodPost,
    Path:        "/users",
    Handler:     createUser,
    OperationId: "create-user",
    Summary:     "Create a new user",
    Tags:        []string{"users"},
    Request:     &UserRequest{}, // documents the request schema
    Response:    &User{},        // documents the 200 response
    Options: []okapi.RouteOption{
        okapi.DocResponse(401, &ErrorUnauthorized{}),
        okapi.DocResponse(404, &ErrorNotFound{}),
    },
})
```

`Request` and `Response` shape the OpenAPI document. Validation happens where the
handler binds the request — `c.Bind(&in)`, or automatically with
`okapi.Handle` / `okapi.H`:

```go
func createUser(c *okapi.Context) error {
    in := &UserRequest{}
    if err := c.Bind(in); err != nil { // binds and validates
        return c.AbortBadRequest("Invalid request", err)
    }
    return c.Created(saveUser(in))
}
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
