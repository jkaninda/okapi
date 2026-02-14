package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/jkaninda/okapi"
)

// Example of using okapi's validation features with struct tags for a simple Book API.
// Okapi provides multiple ways to validate and bind incoming request data, each suited for different use cases.
// See: https://okapi.jkaninda.dev/features/validation.html

type Book struct {
	ID     int    `json:"id" path:"id"`
	Name   string `json:"name" form:"name" maxLength:"50" example:"The Go Programming Language" required:"true"`
	Price  int    `json:"price" form:"price" query:"price" min:"0" default:"0" max:"500"`
	Qty    int    `json:"qty" form:"qty" query:"qty" default:"0"`
	Status string `json:"status" form:"status" enum:"paid,unpaid,canceled" required:"true" example:"paid"`
}

type BookEditInput struct {
	ID   int  `json:"id" path:"id" required:"true"`
	Body Book `json:"body"`
}

type BookDetailInput struct {
	ID int `json:"id" path:"id"`
}

type BookOutput struct {
	Status int
	Body   Book
}

type BooksResponse struct {
	Status     int    // Default 200
	Body       []Book `json:"books"`
	XRequestId string `header:"X-Request-Id"`
}

var books = []Book{
	{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100},
}

func main() {
	o := okapi.Default()
	api := o.Group("api")

	// CREATE
	api.Post("/books", func(c *okapi.Context) error {
		book := &Book{}
		if err := c.Bind(book); err != nil {
			return c.AbortBadRequest("Bad request", err)
		}
		book.ID = len(books) + 1
		books = append(books, *book)
		return c.Created(book)
	},
		// OpenAPi Doc
		okapi.DocRequestBody(&Book{}),
		okapi.DocResponse(&Book{}),
	)

	// READ ONE - Using okapi.H (shorthand)
	api.Get("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
		for _, b := range books {
			if b.ID == input.ID {
				return c.OK(b)
			}
		}
		return c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
	}),
		okapi.DocResponse(&Book{}),
	)

	// READ ALL - Using okapi.HandleO for custom output
	api.Get("/books", okapi.HandleO(func(c *okapi.Context) (*BooksResponse, error) {
		return &BooksResponse{Body: books, XRequestId: uuid.NewString()}, nil
	})).WithOutput(&BooksResponse{})

	// UPDATE - Using okapi.HandleIO for input/output
	api.Put("/books/{id:int}", okapi.HandleIO(func(c *okapi.Context, input *BookEditInput) (*BookOutput, error) {
		for i, b := range books {
			if b.ID == input.ID {
				books[i] = input.Body
				books[i].ID = input.ID
				return &BookOutput{Body: books[i]}, nil
			}
		}
		return nil, c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
	})).WithIO(&BookEditInput{}, &BookOutput{})

	// DELETE - Using okapi.H with path parameter
	api.Delete("/books/{id:int}", okapi.H(func(c *okapi.Context, input *BookDetailInput) error {
		for i, b := range books {
			if b.ID == input.ID {
				books = append(books[:i], books[i+1:]...)
				return c.NoContent()
			}
		}
		return c.AbortNotFound(fmt.Sprintf("Book not found: %d", input.ID))
	})).WithInput(&BookDetailInput{})

	if err := o.Start(); err != nil {
		panic(err)
	}
}
