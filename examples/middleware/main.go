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
	"strings"
	"time"
)

type Book struct {
	ID     int    `json:"id" param:"id" max:"50" `
	Name   string `json:"name" minLength:"5"  maxLength:"50" default:"anonymous" description:"Book name"`
	Price  int    `json:"price"  description:"Book price"`
	Qty    int    `json:"qty" yaml:"qty"`
	Author Author `json:"author" description:"Author"`
}
type Author struct {
	Name string `json:"name"  max:"50" default:"anonymous" description:"Author name"`
}
type LoginRequest struct {
	Username string `json:"username" required:"true"`
	Password string `json:"password" query:"password" required:"true"`
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
	jwtClaims = jwt.MapClaims{
		"sub": "12345",
		"aud": "okapi.jkaninda.dev",
		"user": map[string]string{
			"name":  "",
			"role":  "",
			"email": "admin@example.com",
		},
		"email_verified": true,
		"tags":           []string{"vip", "premium", "gold"},
		"exp":            time.Now().Add(2 * time.Hour).Unix(),
	}
	signingSecret      = "supersecret"
	bearerAuthSecurity = []map[string][]string{
		{
			"bearerAuth": {},
		},
	}
	basicAuthSecurity = []map[string][]string{
		{
			"basicAuth": {},
		},
	}
)

func main() {
	// Example usage of middlewares handling in Okapi
	// Create a new Okapi instance
	o := okapi.New()
	o.WithOpenAPIDocs(okapi.OpenAPI{
		Title:   "Okapi Web Framework Example",
		Version: "1.0.0",
		License: okapi.License{
			Name: "MIT",
		},
		SecuritySchemes: okapi.SecuritySchemes{
			{
				Name:   "basicAuth",
				Type:   "http",
				Scheme: "basic",
			},
			{
				Name:         "bearerAuth",
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
			{
				Name: "OAuth2",
				Type: "oauth2",
				Flows: &okapi.OAuthFlows{
					AuthorizationCode: &okapi.OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
		},
	})

	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Welcome to Okapi!"})
	}, okapi.DocSummary("Home "))

	// Create a new group with a base path for API routes
	api := o.Group("/api")

	// *************** V1 Group with Basic Auth middleware ********************
	v1 := api.Group("/v1")
	// ******* Admin Routes | Restricted Area ********
	basicAuth := okapi.BasicAuth{
		Username: "admin",
		Password: "password",
		Realm:    "Restricted Area",
	}
	// Create a new group with a base path for admin routes and apply basic auth middleware
	adminApi := v1.Group("/admin", basicAuth.Middleware).WithSecurity(basicAuthSecurity) // This group will require basic authentication
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
	// Setup JWT authentication middleware with claims expression
	jwtAuth := okapi.JWTAuth{
		SigningSecret:    []byte(signingSecret),
		TokenLookup:      "header:Authorization",
		Audience:         "okapi.jkaninda.dev",
		ClaimsExpression: "Equals(`email_verified`, `true`) && OneOf(`user.role`, `admin`, `owner`) && Contains(`tags`, `vip`, `premium`, `gold`)",
		ForwardClaims: map[string]string{
			"email": "user.email",
			"role":  "user.role",
			"name":  "user.name",
		},
		ValidateClaims: func(c okapi.Context, claims jwt.Claims) error {
			method := c.Request().Method
			slog.Info("Validating JWT claims", "method", method)
			slog.Info("Validating JWT claims for role using custom function")
			mapClaims, ok := claims.(jwt.MapClaims)
			if !ok {
				return errors.New("invalid claims type")
			}
			role, ok := mapClaims["user"].(map[string]interface{})["role"]
			if !ok || (role != "admin" && role != "user" && role != "owner") {
				return errors.New("unauthorized role")
			}
			slog.Info("Role validation successful", "role", role)
			return nil
		},
	}

	// Create a new group with a base path for admin routes and apply jwt auth middleware
	adminApiV2 := v2.Group("/admin", jwtAuth.Middleware).WithSecurity(bearerAuthSecurity) // This group will require jwt authentication
	adminApiV2.Put("/books/:id", adminUpdate,
		okapi.DocResponse(Book{}),
		okapi.DocRequestBody(Book{}),
		okapi.DocBearerAuth())
	adminApiV2.Post("/books", adminCreateBook,
		okapi.DocSummary("create a book"),
		okapi.DocResponse(Book{}),
		okapi.DocRequestBody(Book{}))

	adminApiV2.Delete("/books/:id", func(c okapi.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.AbortBadRequest("invalid request", err)
		}
		for i, book := range books {
			if book.ID == id {
				books = append(books[:i], books[i+1:]...)
			}
		}
		return c.OK(okapi.M{"message": "Book deleted"})
	})
	adminApiV2.Get("/whoami", whoAmI,
		okapi.DocSummary("Get current user information"),
		okapi.DocResponse(okapi.M{
			"email": "",
			"role":  "",
			"name":  "",
		}))

	// ******* Public API Routes ********
	// Define routes for the v2 group
	v2.Post("/login", func(c okapi.Context) error {
		loginRequest := &LoginRequest{}
		if err := c.Bind(loginRequest); err != nil {
			return c.AbortBadRequest("invalid request", err)
		}
		fmt.Println(loginRequest.Username, loginRequest.Password)
		if loginRequest.Username != "admin" && loginRequest.Password != "password" ||
			loginRequest.Username != "owner" && loginRequest.Password != "password" {

			return c.AbortUnauthorized("username or password is wrong")

		}
		// Update JWT claims with user information
		if _, ok := jwtClaims["user"].(map[string]string); ok {
			slog.Info("Updating JWT claims for user", "username", loginRequest.Username)
			jwtClaims["user"].(map[string]string)["name"] = strings.ToUpper(loginRequest.Username)
			jwtClaims["user"].(map[string]string)["role"] = loginRequest.Username

		}
		// Set the expiration time for the JWT token
		expireAt := 30 * time.Minute
		jwtClaims["exp"] = time.Now().Add(expireAt).Unix()

		token, err := okapi.GenerateJwtToken(jwtAuth.SigningSecret, jwtClaims, expireAt)
		if err != nil {
			return c.AbortInternalServerError("Internal server error", err)
		}
		return c.OK(LoginResponse{token, time.Now().Add(expireAt).Unix()})

	},
		okapi.Doc().
			Summary("Login").
			RequestBody(LoginRequest{}).
			Response(LoginResponse{}).
			Build(),
	)
	v2.Get("/books", getBooks, okapi.DocSummary("Get all books"), okapi.DocResponse([]Book{}))
	v2.Get("/books/:id", findById, okapi.DocSummary("Get book by Id"), okapi.DocResponse(Book{})).Name = "show_book"
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
		return c.AbortBadRequest("Bad request", err)
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
func whoAmI(c okapi.Context) error {
	email := c.GetString("email")
	if email == "" {
		return c.AbortUnauthorized("Unauthorized", fmt.Errorf("user not authenticated"))
	}

	// Respond with the current user information
	return c.JSON(http.StatusOK, okapi.M{
		"email": email,
		"role":  c.GetString("role"),
		"name":  c.GetString("name"),
	},
	)
}

func customMiddleware(next okapi.HandleFunc) okapi.HandleFunc {
	return func(c okapi.Context) error {
		slog.Info("Custom middleware executed", "path", c.Request().URL.Path, "method", c.Request().Method)
		// Call the next handler in the chain
		if err := next(c); err != nil {
			// If an error occurs, log it and return a generic error response
			slog.Error("Error in custom middleware", "error", err)
			return c.JSON(http.StatusInternalServerError, okapi.M{"error": "Internal Server Error"})
		}
		c.Response().StatusCode()
		slog.Info("Response sent", "status", c.Response().StatusCode())

		return nil
	}
}
