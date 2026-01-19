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

package services

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/middlewares"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"net/http"
	"strconv"
)

type BookService struct{}
type CommonService struct{}
type AuthService struct{}

var (
	books = []*models.Book{
		{Id: 1, Name: "The Go Programming Language ", Price: 100},
		{Id: 2, Name: "Building REST/API With Okapi ", Price: 50},
		{Id: 3, Name: "Learning Go", Price: 200},
		{Id: 4, Name: "Go Web Programming", Price: 300},
		{Id: 5, Name: "Go in Action", Price: 150},
	}
	ApiVersion = "V1"
)

// ****************** Services *****************

func (hc *CommonService) Home(c *okapi.Context) error {
	return c.OK(okapi.M{"message": "Welcome to the Okapi Web Framework!"})
}
func (hc *CommonService) Version(c *okapi.Context) error {
	return c.OK(okapi.M{"version": ApiVersion})
}
func (bc *BookService) List(c *okapi.Context) error {
	// Simulate fetching books from a database
	return c.OK(models.SuccessResponse("Books fetched successfully", books))
}

func (bc *BookService) Create(c *okapi.Context) error {
	// Simulate creating a book in a database
	book := &models.Book{}
	err := c.Bind(book)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse("Bad Request", err))
	}
	book.Id = len(books) + 1
	books = append(books, book)
	return c.OK(models.SuccessResponse("Book created successfully", book))
}
func (bc *BookService) Get(c *okapi.Context) error {
	id := c.Param("id")
	i, err := strconv.Atoi(id)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse("Bad Request", err))
	}
	// Simulate a fetching book from a database

	for _, book := range books {
		if book.Id == i {
			return c.OK(book)
		}
	}
	return c.ErrorNotFound(models.ErrorResponse("Not found", fmt.Errorf("book not found with id %s", id)))
}
func (bc *BookService) Delete(c *okapi.Context) error {
	id := c.Param("id")
	i, err := strconv.Atoi(id)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponseDto{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
	}

	// Simulate deleting a book from a database
	for index, book := range books {
		if book.Id == i {
			books = append(books[:index], books[index+1:]...)
			return c.OK(models.Response{
				Success: true,
				Message: "Book deleted successfully",
			})
		}
	}
	return c.AbortNotFound("Book not found")
}

// Example of Okapi using Body Field Style

func (bc *BookService) Update(c *okapi.Context) error {
	bookRequest := &models.BookUpdateRequest{}
	if err := c.Bind(bookRequest); err != nil {
		return c.ErrorBadRequest(models.ErrorResponse("Bad Request", err))
	}
	// Simulate updating a book from a database
	for i, book := range books {
		if book.Id == bookRequest.ID {
			book.Name = bookRequest.Body.Name
			book.Price = bookRequest.Body.Price
			books[i] = book
			return c.Respond(&models.BookResponse{
				RequestId: uuid.NewString(),
				Status:    200,
				Body:      *book,
			})
		}
	}
	return c.ErrorNotFound(models.ErrorResponse("Not found", fmt.Errorf("book not found with id %d", bookRequest.ID)))

}

// ******************** AuthService *****************

func (bc *AuthService) Login(c *okapi.Context) error {
	authRequest := &models.AuthRequest{}
	err := c.Bind(authRequest)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse("Bad Request", err))
	}
	// Validate the authRequest and generate a JWT token
	authResponse, err := middlewares.Login(authRequest)
	if err != nil {
		return c.ErrorUnauthorized(models.ErrorResponse("Unauthorized", err))
	}
	message := "Welcome back " + authRequest.Username

	return c.OK(models.SuccessResponse(message, authResponse))
}
func (bc *AuthService) WhoAmI(c *okapi.Context) error {
	// Get User Information from the context, shared by the JWT middleware using forwardClaims
	email := c.GetString("email")
	if email == "" {
		return c.ErrorUnauthorized(models.ErrorResponse("Unauthorized", fmt.Errorf("user not authenticated")))

	}

	// Respond with the current user information
	return c.OK(models.SuccessResponse("Who am I'm logged in", email))
}
