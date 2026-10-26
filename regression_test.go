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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindQueryAndFormScalarFields(t *testing.T) {
	type input struct {
		Name    string   `query:"name" form:"name"`
		Page    int      `query:"page" form:"page"`
		Active  bool     `query:"active" form:"active"`
		Tags    []string `query:"tags" form:"tags"`
		IDs     []int    `query:"ids" form:"ids"`
		Limit   int      `query:"limit" form:"limit" default:"20"`
		Legacy  string   `json:"legacy_name"`
		Skipped string   `query:"-" form:"-"`
	}
	want := input{Name: "dune", Page: 2, Active: true, Tags: []string{"a", "b", "c"}, IDs: []int{1, 2}, Limit: 20, Legacy: "kept"}

	c, _ := NewTestContext(http.MethodGet, "/?name=dune&page=2&active=true&tags=a&tags=b,c&ids=1,2&legacy_name=kept&Skipped=x", nil)
	var q input
	require.NoError(t, c.BindQuery(&q))
	assert.Equal(t, want, q)

	body := strings.NewReader("name=dune&page=2&active=true&tags=a&tags=b,c&ids=1,2&LEGACY_NAME=kept")
	c, _ = NewTestContext(http.MethodPost, "/", body)
	c.request.Header.Set("Content-Type", constFormURLEncoded)
	var f input
	require.NoError(t, c.BindForm(&f))
	assert.Equal(t, want, f)

	c, _ = NewTestContext(http.MethodGet, "/?page=two", nil)
	var bad input
	err := c.BindQuery(&bad)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Page")
}

func TestOpenAPIDocsCopiesExternalDocsAndComponentSchemas(t *testing.T) {
	o := New().WithOpenAPIDocs(OpenAPI{
		Title:        "Books",
		ExternalDocs: &ExternalDocs{URL: "https://example.com/docs", Description: "Guide"},
		ComponentSchemas: map[string]*SchemaInfo{
			"Money": {Schema: reflectToSchemaWithInfo(responseDocBook{}).Schema},
		},
	})
	spec := o.OpenAPISpec30()
	require.NotNil(t, spec.ExternalDocs)
	assert.Equal(t, "https://example.com/docs", spec.ExternalDocs.URL)
	assert.Contains(t, spec.Components.Schemas, "Money")
}

func TestOpenAPIAutoSecuritySchemesKeepBoth(t *testing.T) {
	o := newSpecTestApp("Both auth")
	o.Get("/a", anyHandler, DocBearerAuth())
	o.Get("/b", anyHandler, DocBasicAuth())
	spec := o.OpenAPISpec30()
	assert.Contains(t, spec.Components.SecuritySchemes, "BearerAuth")
	assert.Contains(t, spec.Components.SecuritySchemes, "BasicAuth")
	assert.NoError(t, o.ValidateOpenAPI(), "every route's scheme is declared")
}

func TestValidateAddrAcceptsHostnames(t *testing.T) {
	for addr, want := range map[string]bool{
		":8080":               true,
		"0.0.0.0:8080":        true,
		"[::1]:8080":          true,
		"localhost:8080":      true,
		"api.example.com:443": true,
		"my-host:80":          true,
		"localhost":           false,
		"-bad:8080":           false,
		"bad_host:8080":       false,
		"localhost:0":         false,
		"localhost:70000":     false,
		"local host:80":       false,
	} {
		assert.Equal(t, want, ValidateAddr(addr), addr)
	}
}

func TestCopyKeepsApplicationSettings(t *testing.T) {
	o := New().WithMaxMultipartMemory(1 << 20)
	c := NewContext(o, httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	c.Set("user", "jonas")

	cp := c.Copy()
	assert.NotPanics(t, func() { cp.Logger().Info("from a copy") })
	assert.Equal(t, int64(1<<20), cp.MaxMultipartMemory())
	assert.Equal(t, "jonas", cp.GetString("user"))

	// A context without an application falls back to the defaults
	bare, _ := NewTestContext(http.MethodGet, "/", nil)
	assert.NotPanics(t, func() { bare.Logger().Info("no application") })
	assert.Equal(t, int64(defaultMaxMemory), bare.MaxMultipartMemory())
}

func TestSetMaxMultipartMemoryIsPerRequest(t *testing.T) {
	o := New().WithMaxMultipartMemory(1 << 20)
	first := NewContext(o, httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	second := NewContext(o, httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	first.SetMaxMultipartMemory(64 << 20)
	assert.Equal(t, int64(64<<20), first.MaxMultipartMemory())
	assert.Equal(t, int64(1<<20), second.MaxMultipartMemory(), "other requests keep the application's limit")
	assert.Equal(t, int64(1<<20), o.maxMultipartMemory)
}

func TestShouldBindHandlesBodyField(t *testing.T) {
	type input struct {
		Page int `query:"page"`
		Body struct {
			Name string `json:"name"`
		}
	}
	c, _ := NewTestContext(http.MethodPost, "/?page=2", strings.NewReader(`{"name":"Dune"}`))
	c.request.Header.Set("Content-Type", constJSON)
	var in input
	ok, err := c.ShouldBind(&in)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 2, in.Page)
	assert.Equal(t, "Dune", in.Body.Name)
}

func TestSliceBodyPayloadIsValidated(t *testing.T) {
	type item struct {
		Name string `json:"name" required:"true"`
	}
	for name, bind := range map[string]func(c *Context) error{
		"values": func(c *Context) error {
			var in struct{ Body []item }
			return c.Bind(&in)
		},
		"pointers": func(c *Context) error {
			var in struct{ Body []*item }
			return c.Bind(&in)
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/", strings.NewReader(`[{"name":"Dune"},{"name":""}]`))
			c.request.Header.Set("Content-Type", constJSON)
			err := bind(c)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "[1]")

			c, _ = NewTestContext(http.MethodPost, "/", strings.NewReader(`[{"name":"Dune"}]`))
			c.request.Header.Set("Content-Type", constJSON)
			assert.NoError(t, bind(c))
		})
	}
}

func TestNoRouteAndNoMethodRunGlobalMiddleware(t *testing.T) {
	o := New()
	o.Use(func(c *Context) error {
		c.SetHeader("X-Global", "yes")
		return c.Next()
	})
	o.Get("/books", anyHandler)
	o.NoRoute(func(c *Context) error { return c.String(http.StatusNotFound, "custom 404") })
	o.NoMethod(func(c *Context) error { return c.String(http.StatusMethodNotAllowed, "custom 405") })

	// Served directly, without starting the server
	for path, want := range map[string]int{"/missing": http.StatusNotFound} {
		rec := httptest.NewRecorder()
		o.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, want, rec.Code)
		assert.Equal(t, "custom 404", rec.Body.String())
		assert.Equal(t, "yes", rec.Header().Get("X-Global"))
	}
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/books", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Equal(t, "custom 405", rec.Body.String())
	assert.Equal(t, "yes", rec.Header().Get("X-Global"))
}
