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

	"github.com/jkaninda/okapi/okapitest"
	"github.com/stretchr/testify/assert"
)

func TestGroup(t *testing.T) {
	o := Default()
	// create api group
	api := o.Group("/api").setDisabled(false).WithTagInfo(GroupTag{
		Name:        "api",
		Description: "API group",
		ExternalDocs: &ExternalDocs{
			URL:         "http://localhost:8080",
			Description: "Example of External Doc",
		},
	})
	// Okapi's Group Middleware
	api.Use(func(c *Context) error {
		slog.Info("Okapi's Group middleware")
		return c.Next()
	})
	test := o.Group("/test").Disable().Deprecated()
	_okapi := test.Okapi()
	_okapi.With(WithDebug())
	// Go's standard HTTP middleware function
	api.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			handler.ServeHTTP(w, r)
		})

	})
	// Go's standard http.HandlerFunc
	api.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Calling route", "path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("standard standard http.HandlerFunc response"))
		if err != nil {
			return
		}
	})
	api.HandleHTTP("GET", "/standard-http", http.FileServer(http.Dir("static")))
	// Okapi Group HandleFun
	api.Get("hello", helloHandler)
	api.Post("hello", helloHandler)
	api.Put("hello", helloHandler)
	api.Patch("hello", helloHandler)
	api.Delete("hello", helloHandler)
	api.Options("hello", helloHandler)
	api.Head("hello", helloHandler)

	api.Get("/group", func(c *Context) error {
		slog.Info("Calling route", "path", c.request.URL.Path)
		return c.OK(M{"message": "Welcome to Okapi!"})
	})
	newG := NewGroup("group", o, LoggerMiddleware).WithTags([]string{"group"})
	newG.Get("/group", func(c *Context) error {
		slog.Info("Calling route", "path", c.request.URL.Path)
		return c.OK(M{"message": "Welcome to Okapi's new group!"})
	})

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

	okapitest.GET(t, "http://localhost:8080/api/group").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/api/standard").ExpectStatusOK()

	okapitest.GET(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.PUT(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.PATCH(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.DELETE(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.OPTIONS(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.HEAD(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/api/tandard-http").ExpectStatusNotFound()
}
func TestRegister(t *testing.T) {
	app := New()
	coreGroup := app.Group("/core").setDisabled(false).WithTags([]string{"CoreGroup"})

	coreGroup.Use(func(c *Context) error {
		slog.Info("Core Group middleware")
		return c.Next()
	})

	bookController := &BookController{}

	coreGroup.Register(bookController.Routes()...)

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer func(app *Okapi) {
		err := app.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(app)
	waitForServer()
	okapitest.GET(t, "http://localhost:8080/core/books").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/core/books").ExpectStatusCreated()

}
func helloHandler(c *Context) error {
	slog.Info("Calling route", "path", c.request.URL.Path, "method", c.request.Method)
	return c.OK(M{"message": "Hello from Okapi!"})

}

func TestGroupWithTagInfo(t *testing.T) {
	o := New()
	api := o.Group("/api").WithTagInfo(
		GroupTag{
			Name:        "api",
			Description: "API group",
			ExternalDocs: &ExternalDocs{
				URL:         "http://localhost:8080",
				Description: "Example of External Doc",
			},
		},
		GroupTag{Name: "shared", Description: "Shared routes"},
	)
	api.Get("/hello", helloHandler)

	// Tag names are appended to Group.Tags so each route inherits them.
	assert.ElementsMatch(t, []string{"api", "shared"}, api.Tags)

	// The single registered route should carry the tag names and tag infos.
	if assert.Len(t, o.routes, 1) {
		r := o.routes[0]
		assert.ElementsMatch(t, []string{"api", "shared"}, r.tags)
		assert.Len(t, r.tagInfos, 2)
	}

	// Build the spec and assert root-level tags carry descriptions + externalDocs.
	o.buildOpenAPISpec()
	spec := o.openapiSpec
	if assert.NotNil(t, spec) && assert.Len(t, spec.Tags, 2) {
		// Sorted alphabetically: api, shared.
		assert.Equal(t, "api", spec.Tags[0].Name)
		assert.Equal(t, "API group", spec.Tags[0].Description)
		if assert.NotNil(t, spec.Tags[0].ExternalDocs) {
			assert.Equal(t, "http://localhost:8080", spec.Tags[0].ExternalDocs.URL)
			assert.Equal(t, "Example of External Doc", spec.Tags[0].ExternalDocs.Description)
		}
		assert.Equal(t, "shared", spec.Tags[1].Name)
		assert.Equal(t, "Shared routes", spec.Tags[1].Description)
		assert.Nil(t, spec.Tags[1].ExternalDocs)
	}
}

func TestGroupWithTagInfo_DedupAcrossRoutes(t *testing.T) {
	o := New()
	g := o.Group("/v1").WithTagInfo(
		GroupTag{Name: "books", Description: "Books API"},
	)
	g.Get("/a", helloHandler)
	g.Get("/b", helloHandler)
	g.Post("/c", helloHandler)

	o.buildOpenAPISpec()
	spec := o.openapiSpec
	if assert.NotNil(t, spec) && assert.Len(t, spec.Tags, 1) {
		assert.Equal(t, "books", spec.Tags[0].Name)
		assert.Equal(t, "Books API", spec.Tags[0].Description)
	}
}

func TestGroupWithTagInfo_IgnoresEmptyName(t *testing.T) {
	o := New()
	g := o.Group("/x").WithTagInfo(
		GroupTag{Name: "", Description: "should be dropped"},
		GroupTag{Name: "kept", Description: "ok"},
	)
	g.Get("/y", helloHandler)

	assert.Equal(t, []string{"kept"}, g.Tags)

	o.buildOpenAPISpec()
	if assert.Len(t, o.openapiSpec.Tags, 1) {
		assert.Equal(t, "kept", o.openapiSpec.Tags[0].Name)
	}
}

func TestGroupWithTagInfo_NoneProducesNoSpecTags(t *testing.T) {
	o := New()
	o.Group("/z").WithTags([]string{"plain"}).Get("/a", helloHandler)

	o.buildOpenAPISpec()
	assert.Empty(t, o.openapiSpec.Tags, "spec.Tags should stay empty when only WithTags is used")
}

func TestGroupWithTagInfo_RegisterPropagates(t *testing.T) {
	o := New()
	g := o.Group("/api").WithTagInfo(GroupTag{Name: "api", Description: "API"})
	g.Register(RouteDefinition{
		Method:  http.MethodGet,
		Path:    "/ping",
		Handler: helloHandler,
	})

	o.buildOpenAPISpec()
	if assert.Len(t, o.openapiSpec.Tags, 1) {
		assert.Equal(t, "api", o.openapiSpec.Tags[0].Name)
		assert.Equal(t, "API", o.openapiSpec.Tags[0].Description)
	}
}
