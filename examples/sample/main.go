package main

import (
	"github.com/jkaninda/okapi"
	"strconv"
)

type Book struct {
	ID    int    `json:"id"`
	Name  string `json:"name"  maxLength:"100" minLength:"2" required:"true" description:"Book name"`
	Price int    `json:"price" max:"100" min:"5"  yaml:"price" required:"true" description:"Book price"`
}
type Books []Book
type BookRequest struct {
	Tags []string `query:"tags"`
	Body Book
}
type BooksResponse struct {
	Version string `header:"X-Version"`
	Status  int
	Body    []Book
}

var books = Books{}

func main() {
	// Create a new Okapi instance with default config
	o := okapi.Default()

	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Hello from Okapi Web Framework!", "License": "MIT"})
	})
	// Using Body and Status
	o.Get("/books", func(c okapi.Context) error {
		output := &BooksResponse{
			Body:    books,
			Version: `1.0.0`,
		}
		return c.Respond(output)
	}).WithOutput(&BooksResponse{})

	o.Post("/books", func(c okapi.Context) error {
		book := &Book{}
		err := c.Bind(book)
		if err != nil {
			return c.AbortBadRequest("Bad Request", err)
		}
		book.ID = len(books) + 1
		books = append(books, *book)
		return c.OK(book)
	},
		// OpenAPI Documentation
		okapi.OperationId("NewBook"),
		okapi.Summary("Create a Book"),
		okapi.Description("Create a new Book"),
		okapi.Request(&BookRequest{}),
		okapi.Response(&Book{}), // Success Response body
	)
	o.Get("/books/:id", func(c okapi.Context) error {
		id, _ := strconv.Atoi(c.Param("id"))
		for _, book := range books {
			if book.ID == id {
				return c.OK(book)
			}
		}
		return c.AbortNotFound("Book not found")
	}).WithOutput(Book{})
	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
