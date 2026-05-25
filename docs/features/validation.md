---
title: Validations
layout: default
parent: Features
nav_order: 1
---
# Validation
Okapi provides a powerful validation system that allows you to easily validate incoming request data against defined rules and constraints. This helps ensure that your API receives well-formed and expected data, improving the robustness and reliability of your application.

## Validation and Default Values

Okapi provides declarative validation and automatic default value assignment using struct tags.

### Basic Validation Tags

| Field Type            | Tag                            | Description                                              |
|-----------------------|--------------------------------|----------------------------------------------------------|
| `string`              | `minLength:"10"`               | Ensures the string has at least 10 characters.           |
| `string`              | `maxLength:"50"`               | Ensures the string does not exceed 50 characters.        |
| `number`              | `min:"5"`                      | Ensures the number is greater than or equal to 5.        |
| `number`              | `max:"100"`                    | Ensures the number is less than or equal to 100.         |
| `number`              | `exclusiveMin:"0"`             | Ensures the number is strictly greater than 0.           |
| `number`              | `exclusiveMax:"100"`           | Ensures the number is strictly less than 100.            |
| `number`              | `multipleOf:"5"`               | Ensures the number is a multiple of the given value.     |
| `slice`               | `maxItems:"5"`                 | Ensures the slice contains at most 5 items.              |
| `slice`               | `minItems:"2"`                 | Ensures the slice contains at least 2 items.             |
| `slice`               | `uniqueItems:"true"`           | Ensures all items in the slice are unique.               |
| `map`                 | `minProperties:"1"`            | Ensures the map has at least 1 entry.                    |
| `map`                 | `maxProperties:"10"`           | Ensures the map has at most 10 entries.                  |
| `any`                 | `required:"true"`              | Marks the field as required.                             |
| `any`                 | `default:"..."`                | Assigns a default value when the field is missing/empty. |
| `string` / `[]string` | `enum:"pending,paid,canceled"` | Restricts the field to one of the listed values.         |
| `string` / `[]string` | `const:"active"`               | Requires the field to equal a fixed value.               |
| `string` / `[]string` | `format:"email"`               | Enables format validation (e.g. `email`, `uuid`, etc.).  |
| `string` / `[]string` | `pattern:"^[a-zA-Z]+$"`        | Validates the field against a regular expression.        |

> **Slices:** `enum`, `const`, `format`, and `pattern` apply to **each element** of a `[]string` field. Failures are reported per index, e.g. `element [2]: ...`.

> **Empty values:** `enum`, `const`, `format`, and `pattern` skip empty strings — combine with `required:"true"` to also enforce presence.



### Data Type & Format Validation

Format validation is enabled with the `format` tag. All formats apply to `string`
fields (and each element of `[]string` fields).

#### Date & time

| Format      | Tag                  | Description                                          |
|-------------|----------------------|-----------------------------------------------------|
| `date`      | `format:"date"`      | Date in `YYYY-MM-DD` form.                          |
| `date-time` | `format:"date-time"` | Date and time (RFC3339).                            |
| `time`      | `format:"time"`      | Time of day (RFC3339 full-time, e.g. `15:04:05Z`). |
| `duration`  | `format:"duration"`  | Go duration (e.g. `1h30m`, `300ms`).               |

#### Network, web & identifiers

| Format          | Tag                     | Description                                                  |
|-----------------|-------------------------|-------------------------------------------------------------|
| `email`         | `format:"email"`        | Valid email address.                                        |
| `hostname`      | `format:"hostname"`     | Valid hostname.                                             |
| `ipv4`          | `format:"ipv4"`         | Valid IPv4 address.                                         |
| `ipv6`          | `format:"ipv6"`         | Valid IPv6 address.                                         |
| `mac`           | `format:"mac"`          | Valid MAC address.                                          |
| `cidr`          | `format:"cidr"`         | CIDR notation (e.g. `192.168.1.0/24`).                     |
| `uri`           | `format:"uri"`          | Valid URI (any scheme).                                    |
| `uri-reference` | `format:"uri-reference"`| URI reference (relative references allowed).               |
| `url`           | `format:"url"`          | Absolute URL using the `http` or `https` scheme.          |
| `uuid`          | `format:"uuid"`         | Valid UUID.                                                 |
| `ulid`          | `format:"ulid"`         | Valid ULID.                                                 |
| `e164` / `phone`| `format:"e164"`         | Phone number in E.164 format (e.g. `+14155552671`).       |
| `credit-card`   | `format:"credit-card"`  | Credit card number (passes the Luhn checksum).            |
| `semver`        | `format:"semver"`       | Semantic version (e.g. `1.2.3-alpha.1`).                  |
| `json-pointer`  | `format:"json-pointer"` | JSON Pointer (RFC 6901).                                   |
| `byte` / `base64`| `format:"byte"`        | Base64-encoded value.                                     |

#### String content

| Format         | Tag                     | Description                                          |
|----------------|-------------------------|-----------------------------------------------------|
| `alpha`        | `format:"alpha"`        | Letters only (`a–z`, `A–Z`).                        |
| `alphanumeric` | `format:"alphanumeric"` | Letters and digits only.                            |
| `numeric`      | `format:"numeric"`      | A numeric string (e.g. `123`, `-12.5`).            |
| `ascii`        | `format:"ascii"`        | ASCII characters only.                              |
| `lowercase`    | `format:"lowercase"`    | No uppercase characters.                            |
| `uppercase`    | `format:"uppercase"`    | No lowercase characters.                            |
| `slug`         | `format:"slug"`         | URL slug (e.g. `my-post-123`).                     |
| `hexcolor`     | `format:"hexcolor"`     | Hex color (`#RGB` or `#RRGGBB`).                   |

#### Custom pattern

| Format  | Tag / Attribute                               | Description                                            |
|---------|-----------------------------------------------|--------------------------------------------------------|
| `regex` | `format:"regex" pattern:"^\+?[1-9]\d{1,14}$"` | Validates the field using a custom regular expression. |

### Example

```go
type CreateUserRequest struct {
    Email    string            `json:"email" required:"true" format:"email" example:"user@example.com"`
    Password string            `json:"password" minLength:"8" description:"User password"`
    Age      int               `json:"age" exclusiveMin:"0" max:"120" default:"18"`
    Website  string            `json:"website" format:"url"`
    Kind     string            `json:"kind" const:"user"`
    Roles    []string          `json:"roles" minItems:"1" uniqueItems:"true" enum:"admin,editor,viewer"`
    Metadata map[string]string `json:"metadata" minProperties:"1" maxProperties:"10"`
}
```

## Validation and Binding Methods

Okapi provides multiple ways to validate and bind incoming request data, each suited for different use cases.

### Method 1: Using `c.Bind()`

The simplest approach to bind and validate the request data within your handler:

```go
o.Post("/users", func(c *okapi.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.ErrorBadRequest(err)
    }
    // Proceed with creating the user using validated data
    return c.JSON(http.StatusOK, req)
})
```

### Method 2: Using `okapi.Handle()` (Input Validation)

Use `okapi.Handle()` when you want automatic input binding and validation with a typed handler signature:

```go
type Book struct {
    ID     int    `json:"id" path:"id"`
    Name   string `json:"name" form:"name" maxLength:"50" required:"true"`
    Price  int    `json:"price" form:"price" min:"0" max:"500" default:"0"`
    Status string `json:"status" enum:"paid,unpaid,canceled" required:"true"`
}

o.Post("/books", okapi.Handle(func(c *okapi.Context, book *Book) error {
    book.ID = generateID()
    return c.Created(book)
}),
    okapi.DocRequestBody(&Book{}),
    okapi.DocResponse(&Book{}),
)
```

### Method 3: Using `okapi.H()` (Shorthand for Handle)

`okapi.H()` is a shorter version of `okapi.Handle()` when you only need input validation:

```go
type BookDetailInput struct {
    ID int `json:"id" path:"id"`
}

o.Get("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
    book := findBookByID(input.ID)
    if book == nil {
        return c.AbortNotFound("Book not found")
    }
    return c.OK(book)
}),
    okapi.DocResponse(&Book{}),
)
```

### Method 4: Using `okapi.HandleIO()` (Input and Output)

Use `okapi.HandleIO()` when you want to define both input and output structs separately. This is useful for complex operations where the response structure differs from the input:

```go
type BookEditInput struct {
    ID   int  `json:"id" path:"id" required:"true"`
    Body Book `json:"body"`
}

type BookOutput struct {
    Status int
    Body   Book
}

o.Put("/books/{id:int}", okapi.HandleIO(func(c *okapi.Context, input *BookEditInput) (*BookOutput, error) {
    book := updateBook(input.ID, input.Body)
    if book == nil {
        return nil, c.AbortNotFound("Book not found")
    }
    return &BookOutput{Body: *book}, nil
})).WithIO(&BookEditInput{}, &BookOutput{})
```

> **Note:** `WithIO()` generates OpenAPI documentation for both input and output schemas. The output struct should follow the body style convention.

### Method 5: Using `okapi.HandleO()` (Output Only)

Use `okapi.HandleO()` when you only need a custom output struct without specific input validation:

```go
type BooksResponse struct {
    Body []Book `json:"books"`
}

o.Get("/books", okapi.HandleO(func(c *okapi.Context) (*BooksResponse, error) {
    return &BooksResponse{Body: getAllBooks()}, nil
})).WithOutput(&BooksResponse{})
```

> **Note:** The output struct must follow the body style convention. The response content type is based on the `Accept` header requested by the client, defaulting to `application/json`.

## Input Sources

Okapi can bind data from multiple sources based on struct tags:

| Tag      | Source            | Example                                    |
|----------|-------------------|--------------------------------------------|
| `json`   | Request body      | `json:"name"`                              |
| `form`   | Form data         | `form:"name"`                              |
| `query`  | Query parameters  | `query:"page"`                             |
| `path`   | Path parameters   | `path:"id"`                                |
| `header` | Request headers   | `header:"Authorization"`                   |

You can combine multiple source tags on the same field:

```go
type BookInput struct {
    ID     int    `json:"id" path:"id"`
    Name   string `json:"name" form:"name" query:"name"`
    Price  int    `json:"price" form:"price" query:"price"`
}
```

## OpenAPI Documentation Helpers

Okapi provides helper methods to generate OpenAPI documentation:

| Method                       | Description                                              |
|------------------------------|----------------------------------------------------------|
| `okapi.DocRequestBody(&T{})` | Documents the request body schema                        |
| `okapi.DocResponse(&T{})`    | Documents the response schema                            |
| `.WithInput(&T{})`           | Documents input schema (for `okapi.H()`)                 |
| `.WithOutput(&T{})`          | Documents output schema (for `okapi.HandleO()`)          |
| `.WithIO(&In{}, &Out{})`     | Documents both input and output (for `okapi.HandleIO()`) |

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/jkaninda/okapi"
)

type Book struct {
    ID     int    `json:"id" path:"id"`
    Name   string `json:"name" form:"name" maxLength:"50" example:"The Go Programming Language" required:"true"`
    Price  int    `json:"price" form:"price" query:"price" min:"0" default:"0" max:"500"`
    Qty    int    `json:"qty" form:"qty" query:"qty" default:"0"`
    Status string `json:"status" form:"status" enum:"paid,unpaid,canceled" required:"true" example:"paid"`
}

type BookEditInput struct {
    ID   int  `json:"id" path:"id" required:"true"`
    Body Book `json:"body"`
}

type BookDetailInput struct {
    ID int `json:"id" path:"id"`
}

type BookOutput struct {
    Status int
    Body   Book
}

type BooksResponse struct {
    Body []Book `json:"books"`
}

var books = []Book{
    {ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100},
}

func main() {
    o := okapi.Default()
    api := o.Group("api")

    // CREATE - Using okapi.Handle with automatic validation
    api.Post("/books", okapi.Handle(func(c *okapi.Context, book *Book) error {
        book.ID = len(books) + 1
        books = append(books, *book)
        return c.Created(book)
    }),
        okapi.DocRequestBody(&Book{}),
        okapi.DocResponse(&Book{}),
    )

    // READ ONE - Using okapi.H (shorthand)
    api.Get("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
        for _, b := range books {
            if b.ID == input.ID {
                return c.OK(b)
            }
        }
        return c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
    }),
        okapi.DocResponse(&Book{}),
    )

    // READ ALL - Using okapi.HandleO for custom output
    api.Get("/books", okapi.HandleO(func(c *okapi.Context) (*BooksResponse, error) {
        return &BooksResponse{Body: books}, nil
    })).WithOutput(&BooksResponse{})

    // UPDATE - Using okapi.HandleIO for input/output
    api.Put("/books/{id:int}", okapi.HandleIO(func(c *okapi.Context, input *BookEditInput) (*BookOutput, error) {
        for i, b := range books {
            if b.ID == input.ID {
                books[i] = input.Body
                books[i].ID = input.ID
                return &BookOutput{Body: books[i]}, nil
            }
        }
        return nil, c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
    })).WithIO(&BookEditInput{}, &BookOutput{})

    // DELETE - Using okapi.H with path parameter
    api.Delete("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
        for i, b := range books {
            if b.ID == input.ID {
                books = append(books[:i], books[i+1:]...)
                return c.NoContent()
            }
        }
        return c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
    })).WithInput(&BookDetailInput{})

    if err := o.Start(); err != nil {
        panic(err)
    }
}
```