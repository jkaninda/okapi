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
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/jkaninda/okapi/okapitest"
	"github.com/stretchr/testify/assert"
)

type ExampleController struct{}

func (ec *ExampleController) Hello(c *Context) error {
	return c.OK("Hello, World!")
}
func (ec *ExampleController) Ping(c *Context) error {
	return c.OK("Pong")
}
func (ec *ExampleController) Routes() []RouteDefinition {

	group := &Group{Prefix: "/api"}
	return []RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/hello",
			Handler:     ec.Hello,
			Group:       group,
			Summary:     "Hello, World!",
			Description: `Hello, World!`,
			Tags:        []string{"hello"},
			Response:    &Book{},
			Request:     &Book{},
			Security:    bearerAuthSecurity,
		},
		{
			Method:  http.MethodPut,
			Path:    "/hello",
			Handler: ec.Hello,
			Group:   group,
		},
		{
			Method:  http.MethodDelete,
			Path:    "/hello",
			Handler: ec.Hello,
			Group:   group,
		},
		{
			Method:  http.MethodPatch,
			Path:    "/hello",
			Handler: ec.Hello,
			Group:   group,
		},
		{
			Method:  http.MethodOptions,
			Path:    "/hello",
			Handler: ec.Hello,
			Group:   group,
		},
		{
			Method:  http.MethodHead,
			Path:    "/hello",
			Handler: ec.Hello,
			Group:   group,
		},

		{
			Method:  http.MethodPost,
			Path:    "/hello",
			Handler: ec.Hello,
			Options: []RouteOption{
				DocSummary("Helleo World Endpoint"),
			},
		},
	}
}
func TestRouteDefinition(t *testing.T) {
	controller := &ExampleController{}
	routes := controller.Routes()

	if len(routes) == 0 {
		t.Fatal("Expected routes to be defined, got none")
	}

	for _, route := range routes {
		if route.Method == "" || route.Path == "" || route.Handler == nil {
			t.Errorf("Invalid route definition: %+v", route)
		}
	}

	t.Logf("Defined %d routes successfully", len(routes))
	app := Default()
	RegisterRoutes(app, routes)

	// Start server in background
	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(app *Okapi) {
		err := app.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(app)

	waitForServer()

	okapitest.GET(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.PUT(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.PATCH(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.OPTIONS(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.DELETE(t, "http://localhost:8080/api/hello").ExpectStatusOK()

}

// TestRouteDefinition_AttachDocOptions verifies that each declarative field
// on RouteDefinition translates into the corresponding RouteOption(s) and
// ends up on the Route when registered.
func TestRouteDefinition_AttachDocOptions(t *testing.T) {
	// A struct with no explicit binding tags so the whole thing becomes
	// the request body schema (populates route.request).
	type CreateBookBody struct {
		Title string `json:"title"`
	}

	o := New()
	o.Register(RouteDefinition{
		Method:      http.MethodPost,
		Path:        "/books",
		Handler:     func(c *Context) error { return c.OK(Book{}) },
		OperationId: "create-book",
		Summary:     "Create a book",
		Description: "Creates a new book",
		Tags:        []string{"books"},
		Request:     &CreateBookBody{},
		Response:    &Book{},
		Security:    bearerAuthSecurity,
	})

	if !assert.Len(t, o.routes, 1) {
		return
	}
	r := o.routes[0]
	assert.Equal(t, "create-book", r.operationId)
	assert.Equal(t, "Create a book", r.summary)
	assert.Equal(t, "Creates a new book", r.description)
	assert.Equal(t, []string{"books"}, r.tags)
	assert.NotNil(t, r.request, "Request should populate body schema for non-bound struct")
	assert.NotEmpty(t, r.responses, "Response should populate route responses")
	assert.Equal(t, bearerAuthSecurity, r.security)
}

// TestRouteDefinition_RequestWithBindingTagsPopulatesParams verifies that a
// request struct with explicit binding tags (query/param/header) flows into
// the corresponding parameter slices instead of the body schema.
func TestRouteDefinition_RequestWithBindingTagsPopulatesParams(t *testing.T) {
	o := New()
	o.Register(RouteDefinition{
		Method:  http.MethodGet,
		Path:    "/books/{id}",
		Handler: func(c *Context) error { return nil },
		Request: &Book{},
	})
	if !assert.Len(t, o.routes, 1) {
		return
	}
	r := o.routes[0]
	assert.Nil(t, r.request, "bound struct should not populate body schema")
	assert.NotEmpty(t, r.queryParams, "query-tagged fields should become query params")
}

// TestRegisterRoutes_RootVsGroup verifies that routes without a Group land
// on the root Okapi instance, while those with a Group are prefixed and
// inherit the group's Okapi pointer even when unset.
func TestRegisterRoutes_RootVsGroup(t *testing.T) {
	o := New()
	group := &Group{Prefix: "/api"} // note: okapi reference unset

	RegisterRoutes(o, []RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/root",
			Handler: func(c *Context) error { return nil },
		},
		{
			Method:  http.MethodGet,
			Path:    "/scoped",
			Handler: func(c *Context) error { return nil },
			Group:   group,
		},
	})

	assert.Same(t, o, group.okapi, "Group without Okapi should be attached to root instance")
	if assert.Len(t, o.routes, 2) {
		paths := []string{o.routes[0].Path, o.routes[1].Path}
		assert.ElementsMatch(t, []string{"/root", "/api/scoped"}, paths)
	}
}

// TestRegisterRoutes_AppliesMiddlewares ensures RouteDefinition.Middlewares
// are wired into the route's handler chain.
func TestRegisterRoutes_AppliesMiddlewares(t *testing.T) {
	var hits int32
	mw := func(c *Context) error {
		atomic.AddInt32(&hits, 1)
		return c.Next()
	}

	o := New()
	RegisterRoutes(o, []RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/ping",
			Handler:     func(c *Context) error { return c.OK("pong") },
			Middlewares: []Middleware{mw},
		},
	})

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer func() { _ = o.Stop() }()
	waitForServer()

	okapitest.GET(t, "http://localhost:8080/ping").ExpectStatusOK()
	assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "custom middleware should run exactly once")
}

// TestRegisterRoutes_PanicsOnInvalidDefinition covers the three panic paths
// in RegisterRoutes: missing method, missing handler, missing both path
// and group, and an unsupported HTTP verb.
func TestRegisterRoutes_PanicsOnInvalidDefinition(t *testing.T) {
	cases := []struct {
		name string
		def  RouteDefinition
	}{
		{
			name: "missing path and group",
			def: RouteDefinition{
				Method:  http.MethodGet,
				Handler: func(c *Context) error { return nil },
			},
		},
		{
			name: "missing method",
			def: RouteDefinition{
				Path:    "/x",
				Handler: func(c *Context) error { return nil },
			},
		},
		{
			name: "missing handler",
			def: RouteDefinition{
				Method: http.MethodGet,
				Path:   "/x",
			},
		},
		{
			name: "unsupported method on root",
			def: RouteDefinition{
				Method:  "TRACE",
				Path:    "/x",
				Handler: func(c *Context) error { return nil },
			},
		},
		{
			name: "unsupported method on group",
			def: RouteDefinition{
				Method:  "TRACE",
				Path:    "/x",
				Handler: func(c *Context) error { return nil },
				Group:   &Group{Prefix: "/g"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := New()
			assert.Panics(t, func() {
				RegisterRoutes(o, []RouteDefinition{tc.def})
			})
		})
	}
}

// TestRouteDefinition_MethodsDispatch verifies the verb-to-registration
// switch in RegisterRoutes covers every supported method for both root and
// group paths.
func TestRouteDefinition_MethodsDispatch(t *testing.T) {
	methods := []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions,
	}

	o := New()
	group := &Group{Prefix: "/api"}
	defs := make([]RouteDefinition, 0, len(methods)*2)
	for _, m := range methods {
		defs = append(defs,
			RouteDefinition{
				Method:  m,
				Path:    "/root",
				Handler: func(c *Context) error { return nil },
			},
			RouteDefinition{
				Method:  m,
				Path:    "/scoped",
				Handler: func(c *Context) error { return nil },
				Group:   group,
			},
		)
	}
	RegisterRoutes(o, defs)

	assert.Len(t, o.routes, len(methods)*2)
	for _, m := range methods {
		var rootFound, scopedFound bool
		for _, r := range o.routes {
			if r.Method == m && r.Path == "/root" {
				rootFound = true
			}
			if r.Method == m && r.Path == "/api/scoped" {
				scopedFound = true
			}
		}
		assert.True(t, rootFound, "root route missing for %s", m)
		assert.True(t, scopedFound, "group route missing for %s", m)
	}
}
