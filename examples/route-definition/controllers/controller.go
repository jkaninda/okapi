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
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"net/http"
	"strconv"
)

type BookController struct{}
type HomeController struct{}

var (
	books = []*models.Book{
		{Id: 1, Name: "Book One", Price: 100},
		{Id: 2, Name: "Book Two", Price: 200},
		{Id: 3, Name: "Book Three", Price: 300},
	}
)

// ****************** Controllers *****************

func (hc *HomeController) Home(c okapi.Context) error {
	return c.OK(okapi.M{"message": "Welcome to the Okapi Web Framework!"})
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
