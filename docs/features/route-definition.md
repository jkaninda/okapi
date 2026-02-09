---
title: Route Definition
layout: default
parent: Features
nav_order: 4
---

# Route Definition

Okapi provides a clean, declarative way to define and register HTTP routes. It supports all standard HTTP methods, including `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, and `OPTIONS`.

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
````

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
			Request:     &BookRequest{},    // OpenAPI request body 
			Response:    &BooksResponse{},  // OpenAPI success response
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
				okapi.DocResponse(&models.Book{}), // 201 Created
				okapi.DocResponse(http.StatusUnauthorized, models.AuthResponse{}),
			},
		},
	}
}
```

---

## OpenAPI & Documentation Fields

Each `RouteDefinition` can directly enrich your OpenAPI documentation:

| Field         | Description                                                      |
|---------------|------------------------------------------------------------------|
| `OperationId` | OperationId is an optional unique identifier for the route       |
| `Summary`     | Short summary displayed in Swagger UI.                           |
| `Description` | Detailed description of the endpoint.                            |
| `Request`     | Request body schema for OpenAPI documentation.                   |
| `Response`    | Default success response schema.                                 |
| `Security`    | Security requirements (e.g. Bearer, API Key, OAuth2).            |
| `Options`     | Advanced documentation and behavior using `RouteOption` helpers. |

> 💡 You can mix `Summary`, `Description`, `Request`, and `Response` with `RouteOption` helpers depending on your preference.

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
* Dynamic enabling/disabling of routes
