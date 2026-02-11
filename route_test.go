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
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"testing"
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
