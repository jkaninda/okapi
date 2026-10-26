/*
 *  MIT License
 *
 * Copyright (c) 2024 Jonas Kaninda
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
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dxBookInput struct {
	ID   int `path:"id"`
	Body struct {
		Name string `json:"name"`
	}
}

type dxBookOutput struct {
	Status int
	Body   responseDocBook
}

func dxCreateBook(_ *Context, _ *dxBookInput) (*dxBookOutput, error) { return &dxBookOutput{}, nil }

func TestTypedRouteRegistrar(t *testing.T) {
	o := newSpecTestApp("Typed routes")
	Post(o, "/books/{id}", dxCreateBook, Summary("Create a book"))
	g := o.Group("/v1")
	Put(g, "/books/{id}", dxCreateBook)
	o.buildOpenAPISpec()

	for _, path := range []string{"/books/{id}", "/v1/books/{id}"} {
		item := o.openapiSpec.Paths.Value(path)
		require.NotNil(t, item, path)
		op := item.Post
		if op == nil {
			op = item.Put
		}
		require.NotNil(t, op, path)
		assert.NotNil(t, op.RequestBody, "the input is documented")
		assert.NotNil(t, op.Parameters.GetByInAndName("path", "id"))
		require.NotNil(t, op.Responses.Value("200"), "the output is documented")
		assert.NotNil(t, op.Responses.Value("400"), "a binding failure is documented")
	}
}

func TestTypedRouteRegistrarKeepsDeclaredDocs(t *testing.T) {
	o := newSpecTestApp("Typed routes override")
	Post(o, "/books/{id}", dxCreateBook, Response(dxBookOutput{Status: 201}))
	op := requestOperation(t, o, "/books/{id}", "POST")
	assert.NotNil(t, op.Responses.Value("201"))
	assert.Nil(t, op.Responses.Value("200"), "a declared 2xx response replaces the inferred one")
}

func TestBindingHandlerDocuments400(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	o := newSpecTestApp("Binding 400")
	o.Post("/handle", Handle(func(c *Context, in *input) error { return nil }))
	o.Post("/io", HandleIO(func(c *Context, in *input) (*responseDocBook, error) { return nil, nil }))
	o.Post("/plain", anyHandler)
	o.Post("/empty", Handle(func(c *Context, in *struct{}) error { return nil }))
	o.Post("/declared", H(func(c *Context, in *input) error { return nil }), DocResponse(400, responseDocError{}))
	o.buildOpenAPISpec()

	for _, path := range []string{"/handle", "/io"} {
		resp := o.openapiSpec.Paths.Value(path).Post.Responses.Value("400")
		require.NotNil(t, resp, path)
		media := resp.Value.Content.Get(constJSON)
		require.NotNil(t, media, path)
		assert.Contains(t, resolveSchema(t, o, media.Schema).Properties, "message", path)
	}
	assert.Nil(t, o.openapiSpec.Paths.Value("/plain").Post.Responses.Value("400"))
	assert.Nil(t, o.openapiSpec.Paths.Value("/empty").Post.Responses.Value("400"), "nothing to bind")

	declared := o.openapiSpec.Paths.Value("/declared").Post.Responses.Value("400")
	assert.NotContains(t, resolveSchema(t, o, declared.Value.Content.Get(constJSON).Schema).Properties, "code")
}

func TestBindingHandler400FollowsErrorFormat(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	handler := Handle(func(c *Context, in *input) error { return nil })

	problem := newSpecTestApp("Problem details").WithSimpleProblemDetailErrorHandler()
	problem.Post("/books", handler)
	op := requestOperation(t, problem, "/books", "POST")
	media := op.Responses.Value("400").Value.Content.Get(constJSONProblem)
	require.NotNil(t, media)
	assert.Contains(t, resolveSchema(t, problem, media.Schema).Properties, "title")

	custom := newSpecTestApp("Custom errors").WithErrorHandler(func(c *Context, code int, message string, err error) error {
		return c.String(code, message)
	})
	custom.Post("/books", handler)
	op = requestOperation(t, custom, "/books", "POST")
	resp := op.Responses.Value("400")
	require.NotNil(t, resp)
	assert.Nil(t, resp.Value.Content, "a custom error handler's output is unknown")
}

func TestNamedExamples(t *testing.T) {
	o := newSpecTestApp("Named examples")
	o.Post("/books", anyHandler, Doc().
		RequestBody(responseDocBook{}).
		RequestExamples(map[string]any{"dune": responseDocBook{Name: "Dune"}, "emma": responseDocBook{Name: "Emma"}}).
		Response(201, responseDocBook{}).
		ResponseExample(201, responseDocBook{Name: "Dune"}).
		Response(404, responseDocError{}).
		ResponseExamples(404, map[string]any{"missing": responseDocError{Message: "not found"}}).
		Build())
	op := requestOperation(t, o, "/books", "POST")

	request := op.RequestBody.Value.Content.Get(constJSON)
	assert.Len(t, request.Examples, 2)
	assert.Contains(t, request.Examples, "dune")

	created := op.Responses.Value("201").Value.Content.Get(constJSON)
	assert.Equal(t, map[string]any{"name": "Dune"}, created.Example)

	notFound := op.Responses.Value("404").Value.Content.Get(constJSON)
	require.Contains(t, notFound.Examples, "missing")
	assert.Equal(t, map[string]any{"message": "not found"}, notFound.Examples["missing"].Value.Value)
}

func TestRequestExampleWithNamedExamples(t *testing.T) {
	o := newSpecTestApp("Mixed examples")
	o.Post("/books", anyHandler,
		DocRequestBody(responseDocBook{}),
		DocRequestExample(responseDocBook{Name: "Dune"}),
		DocRequestExamples(map[string]any{"emma": responseDocBook{Name: "Emma"}}),
	)
	op := requestOperation(t, o, "/books", "POST")
	media := op.RequestBody.Value.Content.Get(constJSON)
	assert.Nil(t, media.Example, "a media type cannot have both example and examples")
	assert.Contains(t, media.Examples, "default")
	assert.Contains(t, media.Examples, "emma")
}

type dxPage[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

func TestGenericSchemaNames(t *testing.T) {
	o := newSpecTestApp("Generics")
	o.Get("/books", anyHandler, DocResponse(dxPage[responseDocBook]{}))
	o.Get("/errors", anyHandler, DocResponse(dxPage[map[string]*responseDocError]{}))
	o.buildOpenAPISpec()

	assert.Contains(t, o.openapiSpec.Components.Schemas, "dxPageResponseDocBook")
	assert.Contains(t, o.openapiSpec.Components.Schemas, "dxPageMapStringResponseDocError")
	assert.Equal(t, "dxPageResponseDocBook", o.openapiSpec.Components.Schemas["dxPageResponseDocBook"].Value.Title)
}

type dxDecimal struct{ v string }

func (dxDecimal) OpenAPISchema() *openapi3.Schema {
	return &openapi3.Schema{Type: &openapi3.Types{"string"}, Format: "decimal"}
}

type dxPet struct{}

func (*dxPet) OpenAPISchema() *openapi3.Schema {
	return &openapi3.Schema{OneOf: openapi3.SchemaRefs{
		openapi3.NewSchemaRef("", openapi3.NewStringSchema()),
		openapi3.NewSchemaRef("", openapi3.NewIntegerSchema()),
	}}
}

func TestSchemaProvider(t *testing.T) {
	type order struct {
		Price dxDecimal  `json:"price" description:"Unit price"`
		Tax   *dxDecimal `json:"tax"`
		Pet   dxPet      `json:"pet"`
	}
	schema := reflectToSchemaWithInfo(order{}).Schema.Value

	price := schema.Properties["price"].Value
	assert.Equal(t, "decimal", price.Format)
	assert.Equal(t, "Unit price", price.Description, "field tags apply on top")
	assert.True(t, schema.Properties["tax"].Value.Nullable)
	assert.Len(t, schema.Properties["pet"].Value.OneOf, 2, "a pointer receiver provides the schema too")

	// The provided schema is not shared between fields
	assert.Empty(t, schema.Properties["tax"].Value.Description)
}

func TestDefaultOperationIDs(t *testing.T) {
	o := newSpecTestApp("Operation IDs")
	o.Get("/books/{id}", anyHandler)
	o.Get("/books", anyHandler, OperationId("get-books-id"))
	o.Post("/books", anyHandler, Summary("Create book"))
	o.buildOpenAPISpec()

	paths := o.openapiSpec.Paths
	assert.Equal(t, "get-books-id", paths.Value("/books").Get.OperationID, "declared ids are kept")
	assert.Equal(t, "get-books-id-2", paths.Value("/books/{id}").Get.OperationID, "generated ids are unique")
	assert.Equal(t, "create-book", paths.Value("/books").Post.OperationID)
}

func TestValidateOpenAPI(t *testing.T) {
	o := newSpecTestApp("Valid")
	Post(o, "/books/{id}", dxCreateBook)
	o.Get("/books", anyHandler, DocResponse(dxPage[responseDocBook]{}))
	assert.NoError(t, o.ValidateOpenAPI())

	invalid := newSpecTestApp("Invalid")
	invalid.Get("/a", anyHandler, OperationId("same"))
	invalid.Get("/b", anyHandler, OperationId("same"))
	err := invalid.ValidateOpenAPI()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same operation id")
}

func TestOpenAPISpecAccessors(t *testing.T) {
	o := newSpecTestApp("Accessors")
	o.Get("/books", anyHandler, DocResponse(http.StatusOK, responseDocBook{}))
	assert.Equal(t, "3.1.0", o.OpenAPISpec().OpenAPI)
	assert.Equal(t, "3.0.3", o.OpenAPISpec30().OpenAPI)
	assert.NotNil(t, o.OpenAPISpec().Paths.Value("/books"))
}

func TestDocSummaryAndDocHide(t *testing.T) {
	o := newSpecTestApp("Doc aliases")
	o.Get("/books", anyHandler, DocSummary("List books"))
	o.Get("/hidden", anyHandler, DocHide())
	o.buildOpenAPISpec()

	assert.Equal(t, "List books", o.openapiSpec.Paths.Value("/books").Get.Summary)
	assert.Nil(t, o.openapiSpec.Paths.Value("/hidden"))
}
