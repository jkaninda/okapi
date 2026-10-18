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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type responseDocBook struct {
	Name string `json:"name"`
}

type responseDocCreated struct {
	Status   int             `description:"Book created"`
	Location string          `header:"Location" description:"URL of the new book"`
	Body     responseDocBook `json:"body"`
}

type responseDocError struct {
	Message string `json:"message"`
}

func TestOpenAPIResponseStatusWithBody(t *testing.T) {
	o := newSpecTestApp("Response status")
	o.Post("/books", anyHandler,
		Response(responseDocCreated{Status: 201}),
		DocResponse(400, responseDocError{}),
	)
	op := requestOperation(t, o, "/books", "POST")

	assert.Nil(t, op.Responses.Value("200"), "a Status with a Body is not documented as 200")
	created := op.Responses.Value("201")
	require.NotNil(t, created)
	assert.Equal(t, "Book created", *created.Value.Description)
	require.NotNil(t, created.Value.Content.Get(constJSON))
	require.Contains(t, created.Value.Headers, "Location")
	assert.Equal(t, "URL of the new book", created.Value.Headers["Location"].Value.Description)
	assert.Empty(t, created.Value.Headers["Location"].Value.Name, "OpenAPI forbids a name on a header object")

	badRequest := op.Responses.Value("400")
	require.NotNil(t, badRequest)
	assert.Equal(t, "Bad Request", *badRequest.Value.Description)
	assert.NotContains(t, badRequest.Value.Headers, "Location", "struct headers stay on their status")
}

func TestOpenAPIUserDocumented500IsKept(t *testing.T) {
	o := newSpecTestApp("Documented 500")
	o.Get("/books", anyHandler, DocResponse(500, responseDocError{}))
	op := requestOperation(t, o, "/books", "GET")

	resp := op.Responses.Value("500")
	require.NotNil(t, resp)
	assert.NotNil(t, resp.Value.Content.Get(constJSON), "the declared 500 schema is not overwritten")
}

func TestOpenAPIDefault500(t *testing.T) {
	o := newSpecTestApp("Default 500")
	o.Get("/books", anyHandler, DocResponse(responseDocBook{}))
	op := requestOperation(t, o, "/books", "GET")

	resp := op.Responses.Value("500")
	require.NotNil(t, resp)
	assert.Equal(t, "Internal Server Error", *resp.Value.Description)
	assert.Nil(t, resp.Value.Content)
}

func TestOpenAPIResponseHeaders(t *testing.T) {
	o := newSpecTestApp("Response headers")
	o.Post("/books", anyHandler, Doc().
		Response(201, responseDocBook{}).
		Response(429, responseDocError{}).
		ResponseHeader("X-Request-ID", "string", "Request ID").
		ResponseHeaderFor(429, "Retry-After", "int", "Seconds to wait").
		Build())
	op := requestOperation(t, o, "/books", "POST")

	created := op.Responses.Value("201").Value
	assert.Contains(t, created.Headers, "X-Request-ID")
	assert.NotContains(t, created.Headers, "Retry-After")

	limited := op.Responses.Value("429").Value
	assert.Contains(t, limited.Headers, "X-Request-ID")
	assert.Contains(t, limited.Headers, "Retry-After")
}

func TestOpenAPIResponseDescription(t *testing.T) {
	o := newSpecTestApp("Response description")
	o.Delete("/books/{id}", anyHandler, Doc().
		ResponseDescription(204, "Book deleted").
		Response(404, responseDocError{}).
		ResponseDescription(404, "No such book").
		Build())
	op := requestOperation(t, o, "/books/{id}", "DELETE")

	noContent := op.Responses.Value("204")
	require.NotNil(t, noContent, "a description alone documents a response")
	assert.Equal(t, "Book deleted", *noContent.Value.Description)
	assert.Nil(t, noContent.Value.Content)

	notFound := op.Responses.Value("404")
	require.NotNil(t, notFound)
	assert.Equal(t, "No such book", *notFound.Value.Description)
	assert.NotNil(t, notFound.Value.Content.Get(constJSON))
}

func TestOpenAPIHeaderOnlyResponse(t *testing.T) {
	type accepted struct {
		Status   int
		Location string `header:"Location"`
	}
	o := newSpecTestApp("Header-only response")
	o.Post("/jobs", anyHandler, Response(accepted{Status: 202}))
	op := requestOperation(t, o, "/jobs", "POST")

	resp := op.Responses.Value("202")
	require.NotNil(t, resp)
	assert.Contains(t, resp.Value.Headers, "Location")
	assert.Nil(t, resp.Value.Content)
}
