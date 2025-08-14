package main

import (
	"github.com/jkaninda/okapi"
	"net/http"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    Book   `json:"data"`
}
type Book struct {
	Name  string `json:"name" form:"name"  max:"50" required:"true" description:"Book name"`
	Price int    `json:"price" form:"price" query:"price" yaml:"price" required:"true" description:"Book price"`
}
type ErrorResponse struct {
	Success bool `json:"success"`
	Status  int  `json:"status"`
	Details any  `json:"details"`
}

func main() {
	// Create a new Okapi instance with default config
	o := okapi.Default()

	o.Get("/", func(c okapi.Context) error {
		return c.OK(okapi.M{"message": "Hello from Okapi Web Framework!", "License": "MIT"})
	})
	o.Post("/books", func(c okapi.Context) error {
		book := Book{}
		err := c.Bind(&book)
		if err != nil {
			return c.ErrorBadRequest(ErrorResponse{Success: false, Status: http.StatusBadRequest, Details: err.Error()})
		}
		response := Response{
			Success: true,
			Message: "This is a simple HTTP POST",
			Data:    book,
		}
		return c.OK(response)
	},
		// OpenAPI Documentation
		okapi.DocOperationId("NewBook"),
		okapi.DocSummary("New Book"),
		okapi.DocDescription("Create a new Book"),
		okapi.DocRequestBody(Book{}),
		okapi.DocResponse(Response{}),                             // Success Response body
		okapi.DocResponse(http.StatusBadRequest, ErrorResponse{}), //  Error response body
	)
	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
