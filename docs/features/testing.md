---
title: Testing
layout: default
parent: Features
nav_order: 6
---

# Testing

Okapi provides comprehensive testing utilities through the `okapitest` package to help you write robust tests for your handlers and middleware.

## Quick Start

```go
import (
    "testing"
    "github.com/jkaninda/okapi"
    "github.com/jkaninda/okapi/okapitest"
)

func TestGetBooksHandler(t *testing.T) {
    // Create a test server
    server := okapi.NewTestServer(t)
    server.Get("/books", GetBooksHandler)
    
    // Make request and assert response
    okapitest.GET(t, server.BaseURL+"/books").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")
}
```

## Testing Approaches

### 1. Using the Test Client (Recommended)

The test client provides a fluent API for making multiple requests with shared configuration:

```go
func TestBooksAPI(t *testing.T) {
    // Setup test server
    server := okapi.NewTestServer(t)
    server.Get("/books", GetBooksHandler)
    server.Get("/books/:id", GetBookHandler)
    server.Post("/books", CreateBookHandler)
    
    // Create reusable client
    client := okapitest.NewClient(t, server.BaseURL)
    
    // Test listing books
    client.GET("/books").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language").
        ExpectHeader("X-Version", "1.0.0")

    // Test getting a specific book
    client.GET("/books/1").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")
    
    // Test book not found
    client.GET("/books/999").
        ExpectStatusNotFound().
        ExpectBodyContains("Book not found")
    
    // Test creating a book
    newBook := Book{
        ID:    6,
        Name:  "Sample Book",
        Price: 20,
        Year:  2024,
        Qty:   5,
    }
    client.POST("/books").
        JSONBody(newBook).
        ExpectStatusCreated().
        ExpectBodyContains("Sample Book")
}
```

### 2. Using Standalone Request Helpers

For simpler test cases or one-off requests:

```go
func TestGetBookHandler(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Get("/books/:id", GetBookHandler)
    
    // Test successful retrieval
    okapitest.GET(t, server.BaseURL+"/books/1").
        ExpectStatusOK().
        ExpectBodyContains("The Go Programming Language")
    
    // Test not found scenario
    okapitest.GET(t, server.BaseURL+"/books/999").
        ExpectStatusNotFound().
        ExpectBodyContains("Book not found")
}

func TestCreateBookHandler(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Post("/books", CreateBookHandler)
    
    book := Book{
        ID:    6,
        Name:  "Sample Book",
        Price: 20,
        Year:  2024,
        Qty:   5,
    }
    
    okapitest.POST(t, server.BaseURL+"/books").
        JSONBody(book).
        ExpectStatusCreated().
        ExpectBodyContains("Sample Book")
}
```

## Available Assertions

The test utilities support various assertions:

```go
// Status code assertions
.ExpectStatusOK()           // 200
.ExpectStatusCreated()      // 201
.ExpectStatusNotFound()     // 404
.ExpectStatus(code int)     // Custom status code

// Body assertions
.ExpectBodyContains(text string)
.ExpectBodyEquals(text string)
.ExpectJSONBody(expected interface{})

// Header assertions
.ExpectHeader(key, value string)
```

## Testing with Custom Headers

```go
func TestAuthenticatedRequest(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Get("/protected", ProtectedHandler)
    
    client := okapitest.NewClient(t, server.BaseURL)
    
    client.GET("/protected").
        Header("Authorization", "Bearer token123").
        ExpectStatusOK()
}
```

## Testing Middleware

```go
func TestAuthMiddleware(t *testing.T) {
    server := okapi.NewTestServer(t)
    server.Use(AuthMiddleware)
    server.Get("/protected", ProtectedHandler)
    
    client := okapitest.NewClient(t, server.BaseURL)
    
    // Test without auth - should fail
    client.GET("/protected").
        ExpectStatus(401)
    
    // Test with auth - should succeed
    client.GET("/protected").
        Header("Authorization", "Bearer valid-token").
        ExpectStatusOK()
}
```