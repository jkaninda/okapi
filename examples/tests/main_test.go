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
	"github.com/jkaninda/okapi/okapitest"
	"testing"
)

// ************* Using the Test Client (Recommended) ****************

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

// ************* Using Standalone Request Helpers ****************

func TestGetBookHandler(t *testing.T) {
	server := okapi.NewTestServerOn(t, 8000)
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
	server := okapi.NewTestServerOn(t, 8000)
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
