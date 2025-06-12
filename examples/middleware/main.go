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
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jkaninda/okapi"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type Book struct {
	ID     int    `json:"id" param:"id" query:"id" form:"id" xml:"id" max:"50" `
	Name   string `json:"name" form:"name"  max:"50" default:"anonymous" description:"Book name"`
	Price  int    `json:"price" form:"price" query:"price" yaml:"price" description:"Book price"`
	Qty    int    `json:"qty" form:"qty" query:"qty" yaml:"qty"`
	Author Author `json:"author" form:"author" yaml:"author" description:"Author"`
}
type Author struct {
	Name string `json:"name" form:"name"  max:"50" default:"anonymous" description:"Author name"`
}
type LoginRequest struct {
	Username string `json:"username" form:"username" query:"username" required:"true"`
	Password string `json:"password" form:"password" query:"password" required:"true"`
}
type LoginResponse struct {
	Token    string `json:"token"`
	ExpireAt int64  `json:"expire_at"`
}

var (
	books = []*Book{
		{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100},
		{ID: 2, Name: "Learning Go", Price: 25, Qty: 50},
		{ID: 3, Name: "Go in Action", Price: 40, Qty: 75},
		{ID: 4, Name: "Go Web Programming", Price: 35, Qty: 60},
		{ID: 5, Name: "Go Design Patterns", Price: 45, Qty: 80},
	}
	adminClaims = jwt.MapClaims{
		"sub":  "12345",
		"role": "admin",
		"exp":  time.Now().Add(2 * time.Hour).Unix(),
	}
)

func main() {
	// Example usage of middlewares handling in Okapi
	// Create a new Okapi instance
	o := okapi.New().WithOpenAPIDocs()

	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Welcome to Okapi!"})
	}, okapi.DocSummary("Home "))

	// Create a new group with a base path for API routes
	api := o.Group("/api")

	// *************** V1 Group with Basic Auth middleware ********************
	v1 := api.Group("/v1")
	// ******* Admin Routes | Restricted Area ********
	basicAuth := okapi.BasicAuthMiddleware{
		Username: "admin",
		Password: "password",
		Realm:    "Restricted Area",
	}
	// Create a new group with a base path for admin routes and apply basic auth middleware
	adminApi := v1.Group("/admin", basicAuth.Middleware) // This group will require basic authentication
	adminApi.Put("/books/:id", adminUpdate, okapi.DocResponse(Book{}), okapi.DocRequestBody(Book{}))
	adminApi.Post("/books", adminCreateBook,
		okapi.DocSummary("create book"),
		okapi.DocResponse(Book{}),
		okapi.DocRequestBody(Book{}))

	// ******* Public API Routes ********
	// Apply custom middleware to the v1 group
	v1.Use(customMiddleware)

	// Define routes for the v1 group
	v1.Get("/books", getBooks, okapi.DocSummary("Get all books"), okapi.DocResponse([]Book{}))
	v1.Get("/books/:id", findById, okapi.DocSummary("Get book by Id"), okapi.DocResponse(Book{})).Name = "show_book"

	// *************** V2 Group with JWT Auth middleware ********************
	v2 := api.Group("/v2")

	// ******* Admin API Routes ********
	// Create middleware
	// Setup
	jwtAuth := okapi.JWTAuth{
		SecretKey:   []byte("supersecret"),
		TokenLookup: "header:Authorization",
		ContextKey:  "user",
		// ValidateRole is optional, it's to validate the role of user
		ValidateRole: func(claims jwt.Claims) error {
			mapClaims, ok := claims.(jwt.MapClaims)
			if !ok {
				return errors.New("invalid claims type")
			}
			role, ok := mapClaims["role"].(string)
			if !ok || role != "admin" {
				return errors.New("unauthorized role")
			}
			return nil
		},
	}

	// Create a new group with a base path for admin routes and apply jwt auth middleware
	adminApiV2 := v2.Group("/admin", jwtAuth.Middleware).WithBearerAuth() // This group will require jwt authentication
	adminApiV2.Put("/books/:id", adminUpdate, okapi.DocResponse(Book{}), okapi.DocRequestBody(Book{}), okapi.DocBearerAuth())
	adminApiV2.Post("/books", adminCreateBook,
		okapi.DocSummary("create a book"),
		okapi.DocResponse(Book{}),
		okapi.DocRequestBody(Book{}))

	// ******* Public API Routes ********
	// Define routes for the v1 group
	v2.Post("/login", func(c okapi.Context) error {
		loginRequest := &LoginRequest{}
		if err := c.Bind(loginRequest); err != nil {
			return c.AbortBadRequest("invalid request", "error", err.Error())
		}
		fmt.Println(loginRequest.Username, loginRequest.Password)
		if loginRequest.Username != "admin" && loginRequest.Password != "password" {

			return c.AbortUnauthorized("invalid request", "error", "username or password is wrong")

		}
		expireAt := time.Now().Add(2 * time.Hour).Unix()
		token, err := okapi.GenerateJwtToken(jwtAuth.SecretKey, adminClaims, time.Duration(expireAt))
		if err != nil {
			return c.AbortInternalServerError("Internal server error", "error", err.Error())
		}
		return c.OK(LoginResponse{token, expireAt})

	},
		okapi.Doc().
			Summary("login").
			RequestBody(LoginRequest{}).
			Response(LoginResponse{}).
			Build(),
	)
	v2.Get("/books", getBooks, okapi.DocSummary("Get all books"), okapi.DocResponse([]Book{}))
	v2.Get("/books/:id", findById, okapi.DocSummary("Get book by Id"), okapi.DocResponse(Book{})).Name = "show_book"
	v2.Delete("/books/:id", func(c okapi.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.AbortBadRequest("invalid request", "error", err.Error())
		}
		for i, book := range books {
			if book.ID == id {
				books = append(books[:i], books[i+1:]...)
			}
		}
		return c.OK(okapi.M{"message": "Book deleted"})
	})
	v2.Put("/books/:id", adminUpdate)
	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}

// ***** Handlers *****

func adminCreateBook(c okapi.Context) error {
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
func getBooks(c okapi.Context) error {
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
			return c.OK(book)
		}
	}
	return c.ErrorNotFound(okapi.M{"error": "Book not found"})
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
