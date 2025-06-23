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

package controllers

import (
	"fmt"
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/middlewares"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"net/http"
	"strconv"
)

type BookController struct{}
type CommonController struct{}
type AuthController struct{}

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

// ****************** Controllers *****************

func (hc *CommonController) Home(c okapi.Context) error {
	return c.OK(okapi.M{"message": "Welcome to the Okapi Web Framework!"})
}
func (hc *CommonController) Version(c okapi.Context) error {
	return c.OK(okapi.M{"version": ApiVersion})
}
func (bc *BookController) GetBooks(c okapi.Context) error {
	// Simulate fetching books from a database
	return c.OK(books)
}

func (bc *BookController) CreateBook(c okapi.Context) error {
	// Simulate creating a book in a database
	book := &models.Book{}
	err := c.Bind(book)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
	}
	book.Id = len(books) + 1
	books = append(books, book)
	response := models.Response{
		Success: true,
		Message: "Book created successfully",
		Data:    *book,
	}
	return c.OK(response)
}
func (bc *BookController) GetBook(c okapi.Context) error {
	id := c.Param("id")
	i, err := strconv.Atoi(id)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
	}
	// Simulate a fetching book from a database

	for _, book := range books {
		if book.Id == i {
			return c.OK(book)
		}
	}
	return c.AbortNotFound("Book not found")
}
func (bc *BookController) DeleteBook(c okapi.Context) error {
	id := c.Param("id")
	i, err := strconv.Atoi(id)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
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

// ******************** AuthController *****************

func (bc *AuthController) Login(c okapi.Context) error {
	authRequest := &models.AuthRequest{}
	err := c.Bind(authRequest)
	if err != nil {
		return c.ErrorBadRequest(models.ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
	}
	// Validate the authRequest and generate a JWT token
	authResponse, err := middlewares.Login(authRequest)
	if err != nil {
		return c.ErrorUnauthorized(authResponse)
	}
	return c.OK(authResponse)
}
func (bc *AuthController) WhoAmI(c okapi.Context) error {
	// Get User Information from the context, shared by the JWT middleware using forwardClaims
	email := c.GetString("email")
	if email == "" {
		return c.AbortUnauthorized("Unauthorized", fmt.Errorf("user not authenticated"))
	}

	// Respond with the current user information
	return c.OK(models.UserInfo{
		Email: email,
		Role:  c.GetString("role"),
		Name:  c.GetString("name"),
	},
	)
}
