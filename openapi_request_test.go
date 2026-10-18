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
	"mime/multipart"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type requestDocParams struct {
	ID      int       `path:"id" description:"Book ID" min:"1"`
	Tags    []string  `query:"tags" description:"Filter tags"`
	Limit   *int      `query:"limit" default:"20" max:"100"`
	Size    uint      `query:"size"`
	Since   time.Time `query:"since"`
	Sort    string    `query:"sort" enum:"asc,desc" validate:"required"`
	Trace   string    `header:"X-Trace-ID" required:"true" pattern:"^[a-f0-9]+$"`
	Session string    `cookie:"session_id" description:"Session cookie"`
	Body    *struct {
		Name string `json:"name"`
	}
}

type uploadDocRequest struct {
	Title  string                  `form:"title" validate:"required" maxLength:"80"`
	File   *multipart.FileHeader   `form:"file" description:"The file"`
	Extras []*multipart.FileHeader `form:"extras"`
	Locale string                  `query:"locale"`
}

type formDocRequest struct {
	Name string `form:"name"`
	Age  int    `form:"age" min:"0"`
}

// requestOperation builds the spec of o and returns the operation at path for method.
func requestOperation(t *testing.T, o *Okapi, path, method string) *openapi3.Operation {
	t.Helper()
	o.buildOpenAPISpec()
	item := o.openapiSpec.Paths.Value(path)
	require.NotNil(t, item, path)
	op := item.GetOperation(method)
	require.NotNil(t, op, method+" "+path)
	return op
}

func operationParam(t *testing.T, op *openapi3.Operation, in, name string) *openapi3.Parameter {
	t.Helper()
	p := op.Parameters.GetByInAndName(in, name)
	require.NotNil(t, p, in+" "+name)
	return p
}

func TestOpenAPIRequestParams(t *testing.T) {
	o := newSpecTestApp("Request params")
	o.Put("/books/{id}", anyHandler, Request(requestDocParams{}))
	op := requestOperation(t, o, "/books/{id}", "PUT")

	id := operationParam(t, op, "path", "id")
	assert.True(t, id.Required)
	assert.Equal(t, "Book ID", id.Description)
	assert.True(t, id.Schema.Value.Type.Is("integer"))
	assert.Equal(t, float64(1), *id.Schema.Value.Min)

	tags := operationParam(t, op, "query", "tags")
	assert.True(t, tags.Schema.Value.Type.Is("array"))
	assert.True(t, tags.Schema.Value.Items.Value.Type.Is("string"))
	assert.Equal(t, "Filter tags", tags.Description)
	assert.Empty(t, tags.Schema.Value.Description, "description belongs to the parameter")

	limit := operationParam(t, op, "query", "limit")
	assert.True(t, limit.Schema.Value.Type.Is("integer"))
	assert.Equal(t, float64(20), limit.Schema.Value.Default)
	assert.Equal(t, float64(100), *limit.Schema.Value.Max)

	size := operationParam(t, op, "query", "size")
	assert.True(t, size.Schema.Value.Type.Is("integer"))
	assert.Equal(t, float64(0), *size.Schema.Value.Min)

	since := operationParam(t, op, "query", "since")
	assert.Equal(t, constDateTime, since.Schema.Value.Format)

	sort := operationParam(t, op, "query", "sort")
	assert.True(t, sort.Required, "validate:\"required\" marks a parameter required")
	assert.Equal(t, []any{"asc", "desc"}, sort.Schema.Value.Enum)

	trace := operationParam(t, op, "header", "X-Trace-ID")
	assert.True(t, trace.Required)
	assert.Equal(t, "^[a-f0-9]+$", trace.Schema.Value.Pattern)

	session := operationParam(t, op, "cookie", "session_id")
	assert.Equal(t, "Session cookie", session.Description)

	require.NotNil(t, op.RequestBody)
	assert.False(t, op.RequestBody.Value.Required, "a pointer Body is optional")
	assert.NotNil(t, op.RequestBody.Value.Content.Get(constJSON))
}

func TestOpenAPIRequiredParamRuleIsExact(t *testing.T) {
	type req struct {
		Q string `query:"q" validate:"required_if=Other x"`
	}
	o := newSpecTestApp("Required rule")
	o.Get("/search", anyHandler, Request(req{}))
	op := requestOperation(t, o, "/search", "GET")
	assert.False(t, operationParam(t, op, "query", "q").Required)
}

func TestOpenAPIStructPathParamsMerge(t *testing.T) {
	type req struct {
		ID string `path:"id" format:"uuid"`
	}
	o := newSpecTestApp("Path params merge")
	o.Get("/shelves/{shelf}/books/{id}", anyHandler,
		DocPathParam("shelf", "int", "Shelf number"),
		Request(req{}),
	)
	op := requestOperation(t, o, "/shelves/{shelf}/books/{id}", "GET")

	shelf := operationParam(t, op, "path", "shelf")
	assert.Equal(t, "Shelf number", shelf.Description, "earlier path params are kept")
	id := operationParam(t, op, "path", "id")
	assert.Equal(t, constUUID, id.Schema.Value.Format)
	assert.Equal(t, "Id", id.Description)
}

func TestOpenAPIResponseCookieIsNotARequestParam(t *testing.T) {
	type res struct {
		Session string `cookie:"session_id"`
		Body    struct {
			OK bool `json:"ok"`
		}
	}
	o := newSpecTestApp("Response cookie")
	o.Get("/login", anyHandler, Response(res{}))
	op := requestOperation(t, o, "/login", "GET")
	assert.Nil(t, op.Parameters.GetByInAndName("cookie", "session_id"))
}

func TestOpenAPIMultipartRequest(t *testing.T) {
	o := newSpecTestApp("Multipart")
	o.Post("/uploads", anyHandler, Request(uploadDocRequest{}))
	op := requestOperation(t, o, "/uploads", "POST")

	assert.NotNil(t, operationParam(t, op, "query", "locale"))
	require.NotNil(t, op.RequestBody)
	content := op.RequestBody.Value.Content
	assert.Nil(t, content.Get(constJSON))
	media := content.Get(constFormData)
	require.NotNil(t, media)

	schema := resolveSchema(t, o, media.Schema)
	assert.ElementsMatch(t, []string{"title", "file", "extras"}, keys(schema.Properties))
	assert.NotContains(t, schema.Properties, "locale")
	assert.Equal(t, []string{"title"}, schema.Required)
	assert.Equal(t, uint64(80), *schema.Properties["title"].Value.MaxLength)

	file := schema.Properties["file"].Value
	assert.True(t, file.Type.Is("string"))
	assert.Equal(t, "binary", file.Format)
	assert.Equal(t, "The file", file.Description)

	extras := schema.Properties["extras"].Value
	assert.True(t, extras.Type.Is("array"))
	assert.Equal(t, "binary", extras.Items.Value.Format)
}

func TestOpenAPIFormRequest(t *testing.T) {
	o := newSpecTestApp("Form")
	o.Post("/people", anyHandler, DocRequestBody(formDocRequest{}))
	op := requestOperation(t, o, "/people", "POST")

	content := op.RequestBody.Value.Content
	assert.NotNil(t, content.Get(constFormURLEncoded))
	assert.NotNil(t, content.Get(constFormData))
	assert.Nil(t, content.Get(constJSON))
}

func TestOpenAPIGetFormFieldsAreNotABody(t *testing.T) {
	o := newSpecTestApp("GET form")
	o.Get("/people", anyHandler, Request(formDocRequest{}))
	op := requestOperation(t, o, "/people", "GET")
	if op.RequestBody != nil {
		assert.Nil(t, op.RequestBody.Value.Content.Get(constFormData))
		assert.Nil(t, op.RequestBody.Value.Content.Get(constFormURLEncoded))
	}
}

func TestOpenAPIRequestContentTypeAndExample(t *testing.T) {
	type book struct {
		Name string `json:"name"`
	}
	example := map[string]any{"name": "Dune"}
	o := newSpecTestApp("Content type")
	o.Post("/books", anyHandler, Doc().
		RequestBody(book{}).
		RequestContentType(constJSON, constXML).
		RequestExample(example).
		Build())
	op := requestOperation(t, o, "/books", "POST")

	content := op.RequestBody.Value.Content
	require.Len(t, content, 2)
	for _, ct := range []string{constJSON, constXML} {
		require.NotNil(t, content.Get(ct), ct)
		assert.Equal(t, example, content.Get(ct).Example, ct)
	}
	assert.True(t, op.RequestBody.Value.Required)
}

func TestOpenAPIFileHeaderInJSONSchema(t *testing.T) {
	type attachment struct {
		File *multipart.FileHeader `json:"file"`
	}
	schema := reflectToSchemaWithInfo(attachment{}).Schema.Value
	file := schema.Properties["file"].Value
	assert.True(t, file.Type.Is("string"))
	assert.Equal(t, "binary", file.Format)
}

func TestOpenAPISliceOfPointers(t *testing.T) {
	type item struct {
		Name string `json:"name"`
	}
	type list struct {
		Items []*item `json:"items"`
	}
	schema := reflectToSchemaWithInfo(list{}).Schema.Value
	items := schema.Properties["items"].Value.Items.Value
	assert.Contains(t, items.Properties, "name")
}

// resolveSchema follows a component reference in the base spec of o.
func resolveSchema(t *testing.T, o *Okapi, ref *openapi3.SchemaRef) *openapi3.Schema {
	t.Helper()
	if ref.Ref == "" {
		return ref.Value
	}
	name := ref.Ref[len("#/components/schemas/"):]
	component := o.openapiSpec.Components.Schemas[name]
	require.NotNil(t, component, ref.Ref)
	return component.Value
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
