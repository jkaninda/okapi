---
title: Route Definition
sidebar_position: 4
---

# Route Definition

Okapi provides a clean, declarative way to define and register HTTP routes. It supports all standard HTTP methods — `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS` — plus `ANY` (or `"*"`) for a route that matches every method.

Routes can be defined individually or grouped and registered in bulk using `okapi.RegisterRoutes` and the `RouteDefinition` struct. This is especially useful when organizing routes by **controller**, **service**, or **feature module**.

---

## Defining Routes with `RouteDefinition`

For better structure and maintainability, routes are typically defined as a slice of `okapi.RouteDefinition`. This pattern fits well with controller/service-based architectures.

### Example: Book Service

```go
type BookService struct{}

func (bc *BookService) GetBooks(c *okapi.Context) error {
	// Simulate fetching books from a database
	return c.OK(okapi.M{
		"success": true,
		"message": "Books retrieved successfully",
	})
}

func (bc *BookService) CreateBook(c *okapi.Context) error {
	// Simulate creating a book in a database
	return c.Created(okapi.M{
		"success": true,
		"message": "Book created successfully",
	})
}
```

---

## Defining Service Routes

You can attach routes directly to your service/controller by returning a slice of `okapi.RouteDefinition`:

```go
func (bc *BookService) BookRoutes() []okapi.RouteDefinition {
	apiGroup := &okapi.Group{Prefix: "/api"}

	return []okapi.RouteDefinition{
		{
			Method:      http.MethodPut,
			Path:        "/books",
			Handler:     bc.UpdateBook,
			Group:       apiGroup,
			Summary:     "Update Book",
			Description: "Update an existing book",
			Request:     &BookRequest{},   // documents the request body and params
			Response:    &BooksResponse{}, // documents the 200 response
		},
		{
			Method:      http.MethodPost,
			Path:        "/books",
			Handler:     bc.CreateBook,
			Group:       apiGroup,
			Middlewares: []okapi.Middleware{customMiddleware},
			Security:    bearerAuthSecurity, // Apply Bearer Auth security scheme

			// Alternative way to define OpenAPI metadata using RouteOptions
			Options: []okapi.RouteOption{
				okapi.DocSummary("Create Book"),
				okapi.DocDescription("Create a new book"),
				okapi.DocRequestBody(&models.Book{}),
				okapi.DocResponse(http.StatusCreated, &models.Book{}), // 201 Created
				okapi.DocResponse(http.StatusUnauthorized, models.AuthResponse{}),
			},
		},
	}
}
```

`okapi.DocResponse(v)` with a single argument documents the **200** response.
Pass the status code — as above — whenever the operation answers with something
else; `okapi.DocResponse(&models.Book{})` next to a `201` handler would document
`200`.

---

## OpenAPI & Documentation Fields

Each `RouteDefinition` can directly enrich your OpenAPI documentation:

| Field         | Description                                                      |
|---------------|------------------------------------------------------------------|
| `OperationId` | OperationId is an optional unique identifier for the route       |
| `Summary`     | Short summary displayed in the documentation UI.                 |
| `Description` | Detailed description of the endpoint.                            |
| `Tags`        | Tags used to group the operation.                                |
| `Request`     | Request body and parameter schema for OpenAPI documentation.     |
| `Response`    | Default success response schema (status `200`).                  |
| `Security`    | Security requirements (e.g. Bearer, API Key, OAuth2).            |
| `Options`     | Advanced documentation and behavior using `RouteOption` helpers. |
| `Middlewares` | Middlewares applied to this route only.                          |
| `Disabled`    | Registers the route disabled: `404 Not Found`, left out of the document. |

> 💡 You can mix `Summary`, `Description`, `Request`, and `Response` with `RouteOption` helpers depending on your preference.

`Request` shapes the documentation only. Binding and validation happen where the
handler reads the request — `c.Bind(&in)`, or automatically with `okapi.Handle` /
`okapi.H`.

`Disabled` is read once, at registration, which makes it a good fit for
configuration and feature flags read at startup. To toggle a route while the
process is running, keep the `*Route` that the verb methods return and call
`Disable()` / `Enable()` on it.

```go
routes := []okapi.RouteDefinition{
    {
        Method:   http.MethodGet,
        Path:     "/experimental",
        Handler:  experimentalHandler,
        Disabled: !cfg.EnableExperimental,
    },
}
```

`Method`, `Handler`, and either `Path` or `Group` are required: a definition
missing one of them, or naming a method Okapi does not support, panics at
registration rather than failing silently.

---

## Registering Routes

Once routes are defined, you can register them using one of the following approaches:

```go
app := okapi.Default()
bookService := &BookService{}

// Method 1: Register directly on the app instance
app.Register(bookService.BookRoutes()...)

// Method 2: Use the global helper
okapi.RegisterRoutes(app, bookService.BookRoutes())
```

Both methods produce the same result—use whichever best fits your project style.

You can also register routes under an existing group:

```go
apiGroup := app.Group("/api")
apiGroup.Register(bookService.BookRoutes()...)
```

`Group.Register` goes through the same code path as the group's verb methods, so
the group's middlewares — **including authentication** — tags, tag info,
deprecation, security requirements and disabled state all apply, together with the
definition's own documentation fields. A definition whose `Group` field is nil is
attached to the group you call `Register` on; one that names a `Group` is
registered on that group instead. The definitions you pass are copied, so your
slice is left untouched.

### Matching Every Method

`Method` also accepts `ANY` or `"*"`:

```go
okapi.RegisterRoutes(app, []okapi.RouteDefinition{
    {
        Method:  "ANY", // or "*"
        Path:    "/webhook",
        Handler: handleWebhook,
    },
})
```

Such a route is documented under `get`, `post`, `put`, `patch` and `delete`.

---

## Full Example

A complete, runnable example is available in:

[https://github.com/jkaninda/okapi/tree/main/examples/route-definition](https://github.com/jkaninda/okapi/tree/main/examples/route-definition)

## Features Demonstrated

A complete, runnable example with Docker images is available in:

[https://github.com/jkaninda/okapi-example](https://github.com/jkaninda/okapi-example)

This example demonstrates:

* Grouped routes
* Controllers/services
* Middleware usage
* OpenAPI documentation generation
* Enabling and disabling routes
