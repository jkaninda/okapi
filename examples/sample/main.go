package main

import (
	"github.com/jkaninda/okapi"
	"strconv"
)

type Book struct {
	ID    int    `json:"id"`
	Name  string `json:"name"  maxLength:"100" minLength:"2" required:"true" description:"Book name" pattern:"^[A-Za-z]+$"`
	Price int    `json:"price" max:"100" min:"5"  yaml:"price" required:"true" description:"Book price"`
	Year  int    `json:"year"  yaml:"year" description:"Book price" deprecated:"true" hidden:"true"`
	Qty   int    `json:"qty" yaml:"qty" description:"Book quantity" `
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
		return c.OK(okapi.M{"message": "Hello from Okapi Web Framework!", "License": "MIT"})
	})
	// Using Body and Status
	o.Get("/books", func(c *okapi.Context) error {
		output := &BooksResponse{
			Body:    books,
			Version: `1.0.0`,
		}
		return c.Respond(output)
	}).WithOutput(&BooksResponse{})

	o.Post("/books", func(c okapi.C) error {
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
	o.Get("/books/{id:int}", func(c *okapi.Context) error {
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
