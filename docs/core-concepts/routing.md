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

## Route Groups

Route groups allow you to organize your routes under a common path prefix, apply middleware selectively, and control group-level behaviors like deprecation or disabling.

### Features

* **Nesting**: Define sub-groups within a parent group to build hierarchical route structures
* **Middleware**: Attach middleware to a group to apply it to all nested routes
* **Deprecation**: Mark a group as deprecated to indicate it's being phased out
* **Disabling**: Temporarily disable a group to return `404 Not Found` for all its routes
* **Tagging**: Automatically tag routes in OpenAPI documentation based on group names

### Example

```go
o := okapi.Default()

// Create the main API group
api := o.Group("/api")

// Versioned subgroups
v1 := api.Group("/v1").Deprecated()        // Marked as deprecated
v2 := api.Group("/v2")                     // Active version
v3 := api.Group("/v3", testMiddleware).Disable() // Disabled, returns 404

// Define routes
v1.Get("/books", getBooks)
v2.Get("/books", v2GetBooks)
v3.Get("/books", v3GetBooks) // Will not be accessible

// Admin subgroup with middleware
admin := api.Group("/admin", adminMiddleware)
admin.Get("/dashboard", getDashboard)
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
app := okapi.Default()

// Create the root API group
api := app.Group("api")

// Define and disable v1 group
v1 := api.Group("v1").Disable() // All v1 routes return 404 and are hidden from docs
v1.Get("/", func(c *okapi.Context) error {
    return c.OK(okapi.M{"version": "v1"})
})

// Define active v2 group
v2 := api.Group("v2")
v2.Get("/", func(c *okapi.Context) error {
    return c.OK(okapi.M{"version": "v2"})
})

// Start the server
if err := app.Start(); err != nil {
    panic(err)
}
```

To re-enable any route or group, simply call the `.Enable()` method or remove the `.Disable()` call.


