---
title: Routing
layout: default
parent: Core Concepts
nav_order: 1
---

# Routing

Okapi provides a simple and intuitive routing system that supports all standard HTTP methods and flexible path patterns.

## HTTP Methods

Okapi supports all standard HTTP methods:

```go
o.Get("/books", getBooks)
o.Post("/books", createBook)
o.Get("/books/:id", getBook)
o.Put("/books/:id", updateBook)
o.Delete("/books/:id", deleteBook)
o.Patch("/books/:id", patchBook)
o.Options("/books", optionsBooks)
```

## Path Syntax

Okapi supports flexible and expressive route path patterns, including named parameters and wildcards:

```go
o.Get("/books/{id}", getBook)       // Named path parameter using curly braces
o.Get("/books/{id:int}", getBook)    // Named path parameter, "id" documented as integer
o.Get("/books/:id", getBook)        // Named path parameter using colon prefix
o.Get("/*", getBook)                // Catch-all wildcard (matches everything)
o.Get("/*any", getBook)             // Catch-all with named parameter (name is ignored)
o.Get("/*path", getBook)            // Catch-all with named parameter
```

Use whichever syntax feels most natural — Okapi normalizes both `{}` and `:` styles for named parameters and supports glob-style wildcards for flexible matching.

## Enabling and Disabling Routes

Okapi allows routes and route groups to be **dynamically enabled or disabled** without commenting out code.

### Features

| Type               | HTTP Response   | Swagger Docs | Affects Child Routes |
|--------------------|-----------------|--------------|----------------------|
| **Disabled Route** | `404 Not Found` | Hidden       | N/A                  |
| **Disabled Group** | `404 Not Found` | Hidden       | Yes — all nested     |

### Use Cases

* Temporarily removing endpoints during maintenance
* Controlling access based on feature flags
* Deprecating old API versions
* Creating toggleable test or staging routes

### Example

```go
func main() {
    app := okapi.Default()
    
    app.Get("/", func(c *okapi.Context) error {
    return c.OK(okapi.M{"version": "v1"})
    }).Disable() // return 404 and are hidden from docs
    
    // Deprecated route example
    app.Get("/deprecated", func(c *okapi.Context) error {
    return c.OK(okapi.M{"message": "This route is deprecated"})
    }).Deprecated() // mark route as deprecated in docs
    
    // Hiden route example
    app.Get("/hidden", func(c *okapi.Context) error {
    return c.OK(okapi.M{"message": "This route is hidden"})
    }).Hide() // hide route from docs
    
    // Start the server
    if err := app.Start(); err != nil {
    panic(err)
    }
}


```

To re-enable any route or group, simply call the `.Enable()` method or remove the `.Disable()` call.


