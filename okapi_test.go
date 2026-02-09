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

package okapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jkaninda/okapi/okapitest"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
)

var testBaseURL = "http://localhost:8080"

type Book struct {
	ID    int    `json:"id" param:"id" query:"id" form:"id"  xml:"id" max:"50" multipleOf:"1" example:"1"`
	Name  string `json:"name" form:"name"  maxLength:"50" example:"The Go Programming Language" yaml:"name"`
	Price int    `json:"price" form:"price" query:"price" yaml:"price" min:"0" default:"0" max:"500"`
	Qty   int    `json:"qty" form:"qty" query:"qty" yaml:"qty" default:"0"`
}

var (
	books = []*Book{
		{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100},
		{ID: 2, Name: "Learning Go", Price: 25, Qty: 50},
		{ID: 3, Name: "Go in Action", Price: 40, Qty: 75},
		{ID: 4, Name: "Go Web Programming", Price: 35, Qty: 60},
		{ID: 5, Name: "Go Design Patterns", Price: 45, Qty: 80},
	}
	pageNotFound     = "Page non trouvée"
	methodNotAllowed = "Cette Methode n'est pas autorisée"
)

func TestStart(t *testing.T) {
	basicAuth := BasicAuth{
		Username: "admin",
		Password: "password",
		Realm:    "Restricted Area",
	}
	o := Default()
	o.NoRoute(func(c C) error {
		return c.String(http.StatusNotFound, pageNotFound)
	})
	o.NoMethod(func(c Ctx) error {
		return c.String(http.StatusMethodNotAllowed, methodNotAllowed)
	})
	o.Get("/", func(c C) error {
		return c.OK(M{"message": "Welcome to Okapi!"})
	})
	o.Get("hello", helloHandler)
	o.Post("hello", helloHandler)
	o.Put("hello", helloHandler)
	o.Patch("hello", helloHandler)
	o.Delete("hello", helloHandler)
	o.Options("hello", helloHandler)
	o.Head("hello", helloHandler)

	// Go's standard http.HandlerFunc
	o.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Calling route", "path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("standard standard http.HandlerFunc response"))
		if err != nil {
			return
		}
	})
	o.HandleHTTP("GET", "/standard-http", http.FileServer(http.Dir("./static/")))
	api := o.Group("/api")
	adminApi := api.Group("/admin", basicAuth.Middleware)
	adminApi.Put("/books/:id", adminUpdate)
	adminApi.Post("/books", adminStore,
		DocSummary("Book Summary"),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
	)

	v1 := api.Group("/v1")
	v1.Use(customMiddleware)
	v1.Get("/books", func(c *Context) error { return c.OK(books) })
	v1.Get("/books/:id", show)

	v2 := api.Group("/v2").Disable()
	v2.Get("/books", func(c *Context) error { return c.OK(books) })
	v2.Get("/books/:id", show)

	v1.Get("/any/*any", func(c *Context) error {
		return c.OK(M{"message": "Tested Any"})
	})
	v1.Get("/all/*", func(c *Context) error {
		return c.OK(M{"message": "Tested Any"})
	})

	o.StaticFile("/favicon.ico", "./favicon.ico")

	o.Get("/events", func(c *Context) error {
		// Simulate sending events (you can replace this with real data)
		for i := 0; i < 10; i++ {
			data := M{"name": "Okapi", "License": "MIT", "event": "SSE example"}
			event := "message"

			err := c.SSEvent(event, data)
			if err != nil {
				return c.AbortWithError(http.StatusInternalServerError, err)
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	})
	o.Get("/events_with_id", func(c *Context) error {
		// Simulate sending events (you can replace this with real data)
		for i := 0; i < 10; i++ {
			data := M{"name": "Okapi", "License": "MIT", "event": "SSE example"}
			event := "message"

			err := c.SendSSEvent(strconv.Itoa(i), event, data)
			if err != nil {
				return c.AbortWithError(http.StatusInternalServerError, err)
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	})
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8080/").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/api/v1/books").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/api/v1/books/1").ExpectStatusOK()

	// Docs
	okapitest.GET(t, "http://localhost:8080/openapi.json").ExpectStatusOK()

	// API V2
	okapitest.GET(t, "http://localhost:8080/api/v2/books/1").ExpectStatusNotFound()

	// Any
	okapitest.GET(t, "http://localhost:8080/api/v1/any/request").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/api/v1/all/request").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/favicon.ico").ExpectStatusNotFound()

	okapitest.GET(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.PUT(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.PATCH(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.DELETE(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.OPTIONS(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.HEAD(t, "http://localhost:8080/hello").ExpectStatusOK()

	okapitest.GET(t, "http://localhost:8080/api/standard-httpo").ExpectStatusNotFound()
	okapitest.GET(t, fmt.Sprintf("%s/api/standard-http", testBaseURL)).ExpectStatusNotFound()

	// NoRoute and NotMethod
	okapitest.GET(t, fmt.Sprintf("%s/api/standard-http", testBaseURL)).ExpectStatusNotFound().ExpectBody(pageNotFound)

	okapitest.GET(t, fmt.Sprintf("%s/custom", testBaseURL)).ExpectStatusNotFound().ExpectBody(pageNotFound)

	okapitest.POST(t, fmt.Sprintf("%s/standard", testBaseURL)).ExpectStatus(http.StatusMethodNotAllowed).ExpectBody(methodNotAllowed)

	// Unauthorized admin Post
	body := `{"id":5,"name":"The Go Programming Language","price":30,"qty":100}`
	okapitest.POST(t, fmt.Sprintf("%s/api/admin/books", testBaseURL)).
		Header("Content-Type", "application/json").
		Body(strings.NewReader(body)).
		ExpectStatusUnauthorized()

	// Authorized admin Post
	body = `{"id":6,"name":"Advanced Go Programming","price":50,"qty":200}`
	okapitest.POST(t, fmt.Sprintf("%s/api/admin/books", testBaseURL)).
		Header("Content-Type", "application/json").
		SetBasicAuth("admin", "password").
		Body(strings.NewReader(body)).
		ExpectStatusCreated().ExpectBodyContains(body)

}
func TestWithServer(t *testing.T) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	server := &http.Server{
		Addr: ":8081",
	}
	logger := slog.New(slog.NewJSONHandler(defaultWriter, opts))
	cors := Cors{AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"}, AllowedOrigins: []string{"*"}}
	o := New()
	o.With(WithPort(8081), WithIdleTimeout(15),
		WithWriteTimeout(10), WithReadTimeout(15),
		WithMaxMultipartMemory(20>>10), WithCors(cors),
		WithLogger(logger), WithContext(context.Background()),
		WithServer(server),
	)
	o.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			handler.ServeHTTP(w, r)
		})

	})
	o.Get("/", customResponseWriter)
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()

	okapitest.GET(t, "http://localhost:8081").ExpectStatusOK()

}
func TestWithAddr(t *testing.T) {

	o := New()
	o.With(WithAddr(":8081"), WithStrictSlash(true)).DisableAccessLog()

	o.Get("/", func(c *Context) error { return c.OK(Book{}) })
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8081").ExpectStatusOK()

}
func TestCustomConfig(t *testing.T) {
	err := os.MkdirAll("public", 0777)
	if err != nil {
		return
	}
	router := mux.NewRouter()
	o := New()
	o.With(WithAddr(":8081"),
		WithStrictSlash(true),
		WithOpenAPIDisabled(),
		WithMuxRouter(router)).WithDebug().
		WithOpenAPIDisabled()

	o.Get("/", func(c *Context) error { return c.OK(Book{}) }).Deprecated()
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8081").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8081/openapi.json").ExpectStatusNotFound()
}

type BookController struct{}

func (bc *BookController) GetBooks(c *Context) error {
	// Simulate fetching books from a database
	return c.OK(M{"success": true, "message": "Books retrieved successfully"})
}

func (bc *BookController) CreateBook(c *Context) error {
	// Simulate creating a book in a database
	return c.Created(M{
		"success": true,
		"message": "Book created successfully",
	})
}
func TestRegisterRoutes(t *testing.T) {
	app := New()
	bookController := &BookController{}

	// Method 1: Register directly to the app instance
	app.Register(bookController.Routes()...)
	// Method 2: Register using RegisterRoutes
	RegisterRoutes(app, bookController.Routes())

	// Start server in background
	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.StopWithContext(context.Background())
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(app)

	waitForServer()

	okapitest.GET(t, "http://localhost:8080/core/books").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/core/books").ExpectStatusCreated()

}

func adminStore(c *Context) error {
	var newBook Book
	if ok, err := c.ShouldBind(&newBook); !ok {
		errMessage := fmt.Sprintf("Failed to bind book: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
	}
	// Add the new book to the list
	newBook.ID = len(books) + 1 // Simple ID assignment
	books = append(books, &newBook)
	// Respond with the created book
	return c.JSON(http.StatusCreated, newBook)
}
func adminUpdate(c *Context) error {
	var newBook Book
	if ok, err := c.ShouldBind(&newBook); !ok {
		errMessage := fmt.Sprintf("Failed to bind book: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
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
	return c.JSON(http.StatusNotFound, M{"error": "Book not found"})
}
func show(c *Context) error {
	var newBook Book
	// Bind the book ID from the request parameters using `param` tags
	// You can also use c.Param("id") to get the ID from the URL
	if ok, err := c.ShouldBind(&newBook); !ok {
		errMessage := fmt.Sprintf("Failed to bind book: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
	}
	// time.Sleep(2 * time.Second) // Simulate a delay for demonstration purposes

	fmt.Println(newBook)

	for _, book := range books {
		if book.ID == newBook.ID {
			return c.JSON(http.StatusOK, book)
		}
	}
	resp := c.Response()
	resp.WriteHeader(http.StatusNotFound)
	// If the book is not found, return a 404 Not Found error
	_, err := resp.Write([]byte("Book not found"))
	if err != nil {
		return err
	}
	return nil
}

func customResponseWriter(c *Context) error {
	// Create a custom response writer
	resp := c.ResponseWriter()
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)

	// Write a custom response
	_, err := resp.Write([]byte(`{"message": "This is a custom response"}`))
	if err != nil {
		return err
	}
	return nil
}

func customMiddleware(next HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		request := c.Request()
		start := time.Now()
		slog.Info("Custom middleware executed", "path", request.URL.Path, "method", request.Method)
		// Call the next handler in the chain
		if err := next(c); err != nil {
			// If an error occurs, log it and return a generic error response
			slog.Error("Error in custom middleware", "error", err)
			return c.JSON(http.StatusInternalServerError, M{"error": "Internal Server Error"})
		}
		slog.Info("request took", "duration", time.Since(start))
		return nil
	}
}
func (bc *BookController) Routes() []RouteDefinition {
	coreGroup := &Group{Prefix: "/core", Tags: []string{"CoreGroup"}}
	return []RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "",
			OperationId: "Get",
			Handler:     bc.GetBooks,
			Group:       coreGroup,
			Request:     nil,
			Response:    &Book{},
		}, {
			Method:      http.MethodGet,
			Path:        "/books",
			OperationId: "List",
			Handler:     bc.GetBooks,
			Group:       coreGroup,
		},
		{
			Method:      http.MethodPost,
			Path:        "/books",
			Handler:     bc.CreateBook,
			Group:       coreGroup,
			Middlewares: []Middleware{customMiddleware},
			Options: []RouteOption{
				DocSummary("Create Book"), // OpenAPI documentation
			},
		},
	}
}

func TestWithComponentSchemaRef(t *testing.T) {
	o := Default()
	err := o.RegisterSchemas(map[string]*SchemaInfo{
		"fieldNames": {
			Schema: openapi3.NewSchemaRef("", openapi3.NewStringSchema().WithEnum([]string{"fldA", "fldB"})),
		},
	})
	if err != nil {
		t.Fatalf("Failed to register schema: %v", err)
	}
	o.Register(RouteDefinition{
		Method: "GET",
		Path:   "/example",
		Options: []RouteOption{
			DocQueryParamWithDefault("fields", "enum", "Fields", false, openapi3.NewSchemaRef("#/components/schemas/fieldNames", nil)),
		},
		Handler: func(c *Context) error {
			return nil
		},
	})

	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err = o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8080/docs").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/openapi.json").ExpectStatusOK()
}

type BookTest struct {
	ID     int    `json:"id"`
	Name   string `json:"name" form:"name"  maxLength:"50" example:"The Go Programming Language" yaml:"name" required:"true"`
	Price  int    `json:"price" form:"price" query:"price" yaml:"price" min:"0" default:"0" max:"500"`
	Qty    int    `json:"qty" form:"qty" query:"qty" yaml:"qty" default:"0"`
	Status string `json:"status" form:"status" query:"status" yaml:"status" enum:"paid,unpaid,canceled"  required:"true" example:"available"`
}
type BooksResponse struct {
	Body       []BookTest `json:"books"`
	XRequestId string     `header:"X-Request-Id"`
	Session    string     `cookie:"session"`
}
type BookDetailReq struct {
	ID int `json:"id" path:"id"`
}
type BookOutPut struct {
	Status int
	Body   BookTest
}

var bookTest = BookTest{ID: 1, Name: "The Go Programming Language", Price: 30, Qty: 100, Status: "paid"}

func TestHandle(t *testing.T) {
	o := NewTestServer(t)
	o.Post("/books", Handle(func(c *Context, book *BookTest) error {
		return c.Created(book)
	}))
	o.Put("/books", HandleIO(func(c *Context, book *BookTest) (*BookOutPut, error) {
		return &BookOutPut{Body: *book}, nil
	}))
	o.Get("/books/:id", H(func(c *Context, book *BookDetailReq) error {
		if book.ID != 1 {
			fmt.Println("ID", book.ID)
			return c.AbortNotFound("Book not found")
		}
		return c.OK(bookTest)
	}))
	o.Delete("/books/:id", H(func(c *Context, book *BookDetailReq) error {
		if book.ID != 1 {
			fmt.Println("ID", book.ID)
			return c.AbortNotFound("Book not found")
		}
		return c.NoContent()
	}))
	o.Get("/books", HandleO(func(c *Context) (*BooksResponse, error) {
		books := make([]BookTest, 0, 1)
		books = append(books, bookTest)
		return &BooksResponse{Body: books, XRequestId: uuid.NewString(), Session: "1234"}, nil
	}))

	okapitest.POST(t, o.BaseURL+"/books").ExpectStatusBadRequest()
	okapitest.POST(t, o.BaseURL+"/books").JSONBody(&BookTest{
		Name: "The Go Programming Language"}).ExpectStatusBadRequest()

	okapitest.POST(t, o.BaseURL+"/books").JSONBody(bookTest).ExpectStatusCreated()

	okapitest.PUT(t, o.BaseURL+"/books").JSONBody(&BookTest{}).ExpectStatusBadRequest()
	okapitest.PUT(t, o.BaseURL+"/books").JSONBody(bookTest).ExpectStatusOK().ExpectBodyContains("The Go Programming Language")

	okapitest.GET(t, o.BaseURL+"/books/1").ExpectStatusOK().ExpectBodyContains("The Go Programming Language")
	okapitest.DELETE(t, o.BaseURL+"/books/1").ExpectStatusNoContent().ExpectEmptyBody()
	okapitest.GET(t, o.BaseURL+"/books/1").ExpectStatusOK().ExpectBodyContains("The Go Programming Language")
	okapitest.GET(t, o.BaseURL+"/books").ExpectStatusOK().ExpectBodyContains("The Go Programming Language").ExpectHeaderExists("X-Request-Id").ExpectCookie("session", "1234")

}
