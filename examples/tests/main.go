/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
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
	"github.com/jkaninda/okapi"
	"net/http"
	"strconv"
)

type Book struct {
	ID    int    `json:"id"`
	Name  string `json:"name"  maxLength:"100" minLength:"2" required:"true" description:"Book name"`
	Price int    `json:"price" max:"100" min:"5"  yaml:"price" required:"true" description:"Book price"`
	Year  int    `json:"year"  yaml:"year" description:"Book price" deprecated:"true" hidden:"true"`
	Qty   int    `json:"qty" yaml:"qty" description:"Book quantity"`
}
type Books []Book
type BookRequest struct {
	Tags []string `query:"tags"`
	Body Book
}
type BooksResponse struct {
	Version string `header:"X-version"`
	Status  int
	Body    []Book
}

var books = Books{
	{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100, Year: 2014},
	{ID: 2, Name: "Learning Go", Price: 25, Qty: 50, Year: 2021},
	{ID: 3, Name: "Go in Action", Price: 40, Qty: 75, Year: 2015},
	{ID: 4, Name: "Go Web Programming", Price: 35, Qty: 60, Year: 2016},
	{ID: 5, Name: "Go Design Patterns", Price: 45, Qty: 80, Year: 2017},
}

func main() {
	// Create a new Okapi instance with default config
	o := okapi.Default()

	o.Get("/", func(c *okapi.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	// Using Body style
	o.Get("/books", GetBooksHandler).WithOutput(&BooksResponse{})

	o.Post("/books", CreateBookHandler).WithInput(&BookRequest{}).WithOutput(&Book{})

	o.Get("/books/{id:int}", GetBookHandler).WithOutput(Book{})
	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
func GetBooksHandler(c *okapi.Context) error {
	// Using Body style
	output := &BooksResponse{
		Status:  http.StatusOK,
		Body:    books,
		Version: `1.0.0`,
	}
	return c.Respond(output)
}
func GetBookHandler(c *okapi.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	for _, book := range books {
		if book.ID == id {
			return c.OK(book)
		}
	}
	return c.AbortNotFound("Book not found")
}
func CreateBookHandler(c *okapi.Context) error {
	book := &Book{}
	err := c.Bind(book)
	if err != nil {
		return c.AbortBadRequest("Bad Request", err)
	}
	book.ID = len(books) + 1
	books = append(books, *book)
	return c.Created(book)
}
