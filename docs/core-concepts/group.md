---
title: Route Groups
layout: default
parent: Core Concepts
nav_order: 4
---

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

## Enabling and Disabling route groups

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


