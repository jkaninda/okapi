/*
 *  MIT License
 *
 * Copyright (c) 2025 Jonas Kaninda
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in all
 *  copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 *  SOFTWARE.
 */

package main

import (
	"fmt"
	"github.com/jkaninda/okapi"
	"log/slog"
	"net/http"
)

type Book struct {
	ID     int    `json:"id" param:"id" query:"id" form:"id" xml:"id" max:"50" `
	Name   string `json:"name" form:"name"  max:"50" default:"anonymous" required:"true" description:"Book name"`
	Price  int    `json:"price" form:"price" query:"price" yaml:"price" `
	Qty    int    `json:"qty" form:"qty" query:"qty" yaml:"qty" required:"true"`
	Author Author `json:"author" form:"author"`
}
type Author struct {
	Name string `json:"name" form:"name"  max:"50" default:"anonymous" required:"true"`
}

var (
	books = []*Book{
		{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100},
		{ID: 2, Name: "Learning Go", Price: 25, Qty: 50},
		{ID: 3, Name: "Go in Action", Price: 40, Qty: 75},
		{ID: 4, Name: "Go Web Programming", Price: 35, Qty: 60},
		{ID: 5, Name: "Go Design Patterns", Price: 45, Qty: 80},
	}
)

func main() {
	// Example usage of middlewares handling in Okapi
	// Create a new Okapi instance
	// Disable access log for cleaner output in this example
	o := okapi.New().WithOpenAPIDocs()

	o.Get("/", func(c okapi.Context) error {
		return c.JSON(http.StatusOK, okapi.M{"message": "Welcome to Okapi!"})
	}, okapi.DocSummary("Home "))

	o.Get("/books/{id}", findById,
		okapi.DocSummary("Get book by ID"),
		okapi.DocPathParam("id", "int", "Book ID"),
		okapi.DocQueryParam("country", "string", "Country Name", false),
		okapi.DocHeader("Key", "1234", "API Key", false),
		okapi.DocTag("bookController"),
		okapi.DocBearerAuth(),
		okapi.DocRequest(Book{}),
		okapi.DocResponse(Book{}))
	// ******* Admin Routes | Restricted Area ********
	basicAuth := okapi.BasicAuthMiddleware{
		Username: "admin",
		Password: "password",
		Realm:    "Restricted Area",
	}
	// Create a new group with a base path for API routes
	api := o.Group("/api")

	// Create a new group with a base path for admin routes and apply basic auth middleware
	adminApi := api.Group("/admin", basicAuth.Middleware) // This group will require basic authentication
	adminApi.Put("/books/:id", adminUpdate)
	adminApi.Post("/books", adminStore,
		okapi.DocSummary("Store books"),
		okapi.DocResponse(Book{}),
		okapi.DocRequest(Book{}))

	// ******* Public API Routes ********
	v1 := api.Group("/v1")
	// Apply custom middleware to the v1 group
	v1.Use(customMiddleware)

	// Define routes for the v1 group
	v1.Get("/books", index, okapi.DocSummary("Get all books"), okapi.DocResponse([]Book{}))
	v1.Get("/books/:id", findById, okapi.DocSummary("Get book by Id"), okapi.DocResponse(Book{})).Name = "show_book"

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}

// ***** Handlers *****

func adminStore(c okapi.Context) error {
	var newBook Book
	if ok, err := c.ShouldBind(&newBook); !ok {
		return c.AbortBadRequest(fmt.Sprintf("Failed to bind book: %v", err))
	}
	// Get username
	username := c.GetString("username")
	fmt.Printf("Current user: %s\n", username)
	// Add the new book to the list
	newBook.ID = len(books) + 1 // Simple ID assignment
	books = append(books, &newBook)
	// Respond with the created book
	return c.JSON(http.StatusCreated, newBook)
}
func adminUpdate(c okapi.Context) error {
	var newBook Book
	if ok, err := c.ShouldBind(&newBook); !ok {
		return c.AbortWithError(400, "Bad request", err)
	}
	for _, book := range books {
		if book.ID == newBook.ID {
			book.Name = newBook.Name
			book.Price = newBook.Price
			book.Qty = newBook.Qty
			// Respond with the updated book
			return c.JSON(http.StatusOK, book)
		}
	}
	return c.AbortNotFound("Book not found")
}
func index(c okapi.Context) error {
	return c.JSON(http.StatusOK, books)
}
func findById(c okapi.Context) error {
	var newBook Book
	// Bind the book ID from the request parameters using `param` tags
	// You can also use c.Param("id") to get the ID from the URL
	if ok, err := c.ShouldBind(&newBook); !ok {
		return c.AbortBadRequest(fmt.Sprintf("Failed to bind book: %v", err))
	}
	// time.Sleep(2 * time.Second) // Simulate a delay for demonstration purposes

	for _, book := range books {
		if book.ID == newBook.ID {
			return c.JSON(http.StatusOK, book)
		}
	}
	return c.JSON(http.StatusNotFound, okapi.M{"error": "Book not found"})
}

func customMiddleware(next okapi.HandleFunc) okapi.HandleFunc {
	return func(c okapi.Context) error {
		slog.Info("Custom middleware executed", "path", c.Request.URL.Path, "method", c.Request.Method)
		// Call the next handler in the chain
		if err := next(c); err != nil {
			// If an error occurs, log it and return a generic error response
			slog.Error("Error in custom middleware", "error", err)
			return c.JSON(http.StatusInternalServerError, okapi.M{"error": "Internal Server Error"})
		}
		return nil
	}
}
