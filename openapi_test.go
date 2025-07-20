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
	"log/slog"
	"net/http"
	"testing"
)

func TestOpenAPI(t *testing.T) {
	o := Default()
	// create api group
	api := o.Group("api").WithBearerAuth()
	v1 := api.Group("v1")
	v2 := api.Group("v2")
	v1.Post("/books", anyHandler,
		DocSummary("Book Summary"),
		DocAutoPathParams(),
		DocQueryParam("auth", "string", "auth name", true),
		DocBearerAuth(),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
	)
	v1.Put("/books", anyHandler,
		DocSummary("Book Summary"),
		DocAutoPathParams(),
		DocQueryParam("auth", "string", "auth name", true),
		DocBearerAuth(),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
	)
	v1.Get("/books", anyHandler,
		DocSummary("Books Summary"),
		DocQueryParam("auth", "string", "auth name", true),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
		DocDeprecated(),
	)
	v1.Delete("/books/{id}", anyHandler,
		DocSummary("Book Summary"),
		DocPathParam("id", "int", "book id"),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
		DocDeprecated(),
	)
	// V2
	v2.Post("/books", anyHandler,
		DocSummary("Book Summary"),
		DocAutoPathParams(),
		DocQueryParam("auth", "string", "auth name", true),
		DocResponseHeader("X-RateLimit-Limit", "int", "The number of requests allowed per minute"),
		DocBearerAuth(),
		DocResponse(201, Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
	)
	v2.Put("/books", anyHandler,
		Doc().Summary("Book Summary").
			BearerAuth().
			Response(Book{}).
			RequestBody(Book{}).
			Tags("Book Tags").
			ErrorResponse(http.StatusBadRequest, M{"": ""}).AsOption(),
	)
	v2.Get("/books", anyHandler,
		Doc().Summary("Book Summary").
			BearerAuth().
			QueryParam("auth", "string", "auth name", true).
			ResponseHeader("X-RateLimit-Limit", "int", "The number of requests allowed per minute").
			Response(200, Book{}).
			Tags("Book Tags").
			ErrorResponse(http.StatusBadRequest, M{"": ""}).Build(),
	)
	v2.Delete("/books/:id", anyHandler,
		Doc().Summary("Delete Book").
			Description("Delete a book by ID").
			BearerAuth().
			PathParam("id", "int", "book id").
			Response(Book{}).
			Tags("Book Tags").
			ErrorResponse(http.StatusBadRequest, M{"": ""}).Build(),
	)
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()

	assertStatus(t, "GET", "http://localhost:8080/docs", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/openapi.json", nil, nil, "", http.StatusOK)

}
func anyHandler(c Context) error {
	slog.Info("Calling route", "path", c.Request().URL.Path, "method", c.request.Method)
	c.SetHeader("X-RateLimit-Limit", "100")
	return c.OK(M{"message": "Hello from Okapi!"})

}
