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
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

var testBaseURL = "http://localhost:8080"

type Book struct {
	ID    int    `json:"id" param:"id" query:"id" form:"id"  xml:"id" max:"50" `
	Name  string `json:"name" form:"name"  maxLength:"50"`
	Price int    `json:"price" form:"price" query:"price" yaml:"price"`
	Qty   int    `json:"qty" form:"qty" query:"qty" yaml:"qty"`
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
	o.NoRoute(func(c Context) error {
		return c.String(http.StatusNotFound, pageNotFound)
	})
	o.NoMethod(func(c Context) error {
		return c.String(http.StatusMethodNotAllowed, methodNotAllowed)
	})

	o.Get("/", func(c Context) error {
		return c.OK(M{"message": "Welcome to Okapi!"})
	})
	o.Get("hello", helloHandler)
	o.Post("hello", helloHandler)
	o.Put("hello", helloHandler)
	o.Patch("hello", helloHandler)
	o.Delete("hello", helloHandler)
	o.Options("hello", helloHandler)
	o.Head("hello", helloHandler)
	o.Trace("hello", helloHandler)
	o.Connect("hello", helloHandler)

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
		DocTags("Book Tag"),
	)

	v1 := api.Group("/v1")
	v1.Use(customMiddleware)
	v1.Get("/books", func(c Context) error { return c.OK(books) })
	v1.Get("/books/:id", show)

	v2 := api.Group("/v2").Disable()
	v2.Get("/books", func(c Context) error { return c.OK(books) })
	v2.Get("/books/:id", show)

	v1.Get("/any/*any", func(c Context) error {
		return c.OK(M{"message": "Tested Any"})
	})
	v1.Get("/all/*", func(c Context) error {
		return c.OK(M{"message": "Tested Any"})
	})

	o.StaticFile("/favicon.ico", "./favicon.ico")

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()

	waitForServer()
	assertStatus(t, "GET", "http://localhost:8080/", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/v1/books", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/v1/books/1", nil, nil, "", http.StatusOK)
	// Docs
	assertStatus(t, "GET", "http://localhost:8080/openapi.json", nil, nil, "", http.StatusOK)

	// API V2
	assertStatus(t, "GET", "http://localhost:8080/api/v2/books/1", nil, nil, "", http.StatusNotFound)
	// Any
	assertStatus(t, "GET", "http://localhost:8080/api/v1/any/request", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/v1/all/request", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/favicon.ico", nil, nil, "", http.StatusNotFound)

	assertStatus(t, "GET", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "POST", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "PUT", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "PATCH", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "DELETE", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "HEAD", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "TRACE", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "CONNECT", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)

	assertStatus(t, "GET", "http://localhost:8080/api/standard-http", nil, nil, "", http.StatusNotFound)
	assertStatus(t, "GET", fmt.Sprintf("%s/api/standard-http", testBaseURL), nil, nil, "", http.StatusNotFound)

	// NoRoute and NotMethod
	assertStatus(t, "GET", fmt.Sprintf("%s/api/standard-http", testBaseURL), nil, nil, "", http.StatusNotFound)
	assertResponse(t, "GET", fmt.Sprintf("%s/custom", testBaseURL), nil, nil, "", http.StatusNotFound, pageNotFound)
	assertResponse(t, "POST", fmt.Sprintf("%s/standard", testBaseURL),
		nil, nil, "",
		http.StatusMethodNotAllowed, methodNotAllowed)

	// Unauthorized admin Post
	body := `{"id":5,"name":"The Go Programming Language","price":30,"qty":100}`
	assertStatus(t, "POST",
		"http://localhost:8080/api/admin/books", nil,
		strings.NewReader(body), "application/json",
		http.StatusUnauthorized)

	// Authorized admin Post
	body = `{"id":6,"name":"Advanced Go Programming","price":50,"qty":200}`
	req, err := http.NewRequest("POST", "http://localhost:8080/api/admin/books", strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create Post request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "password")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}
func TestWithServer(t *testing.T) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	// Initialize the appropriate handler based on format preference

	logger := slog.New(slog.NewJSONHandler(DefaultWriter, opts))
	cors := Cors{AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"}, AllowedOrigins: []string{"*"}}
	o := New()
	o.With(WithPort(81), WithIdleTimeout(15),
		WithWriteTimeout(10), WithReadTimeout(15),
		WithMaxMultipartMemory(20>>10), WithCors(cors),
		WithLogger(logger))

	o.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			handler.ServeHTTP(w, r)
		})

	})
	o.Get("/", func(c Context) error {
		c.Response.BodyBytesSent()

		return c.OK(Book{})
	})
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()
	waitForServer()
	assertStatus(t, "GET", "http://localhost:81", nil, nil, "", http.StatusOK)

}
func TestWithAddr(t *testing.T) {

	o := New()
	o.With(WithAddr(":81"), WithStrictSlash(true)).DisableAccessLog()

	o.Get("/", func(c Context) error { return c.OK(Book{}) })
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()
	waitForServer()
	assertStatus(t, "GET", "http://localhost:81", nil, nil, "", http.StatusOK)

}
func TestCustomConfig(t *testing.T) {
	router := mux.NewRouter()
	o := New()
	o.With(WithAddr(":81"),
		WithStrictSlash(true),
		WithOpenAPIDisabled(),
		WithMuxRouter(router),
		WithMux(router)).WithDebug().
		WithOpenAPIDisabled()

	o.Get("/", func(c Context) error { return c.OK(Book{}) })
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()
	waitForServer()
	assertStatus(t, "GET", "http://localhost:81", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:81/openapi.json", nil, nil, "", http.StatusNotFound)

}
func assertStatus(t *testing.T, method, url string,
	headers map[string]string,
	body io.Reader,
	contentType string,
	expected int) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create %s request to %s: %v", method, url, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != expected {
		t.Errorf("Expected status %d for %s %s, got %d", expected, method, url, resp.StatusCode)
	}
}
func assertResponse(t *testing.T, method, url string,
	headers map[string]string,
	body io.Reader,
	contentType string,
	expectedStatus int,
	expectedBody string,
) {
	t.Helper()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create %s request to %s: %v", method, url, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make %s request to %s: %v", method, url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	actualBody := string(bodyBytes)

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d for %s %s, got %d", expectedStatus, method, url, resp.StatusCode)
	}

	if expectedBody != "" && actualBody != expectedBody {
		t.Errorf("Expected body:\n%s\nGot:\n%s", expectedBody, actualBody)
	}
}

func waitForServer() {
	time.Sleep(100 * time.Millisecond)
}

func adminStore(c Context) error {
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
func adminUpdate(c Context) error {
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
func show(c Context) error {
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
	return c.JSON(http.StatusNotFound, M{"error": "Book not found"})
}

func customMiddleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		start := time.Now()
		slog.Info("Custom middleware executed", "path", c.Request.URL.Path, "method", c.Request.Method)
		// Call the next handler in the chain
		if err := next(c); err != nil {
			// If an error occurs, log it and return a generic error response
			slog.Error("Error in custom middleware", "error", err)
			return c.JSON(http.StatusInternalServerError, M{"error": "Internal Server Error"})
		}
		slog.Info("Request took", "duration", time.Since(start))
		return nil
	}
}
