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
	"testing"

	"github.com/jkaninda/okapi/okapitest"
)

func TestRegisterDocRoutes(t *testing.T) {
	o := New()
	o.Get("/", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	})

	o.registerDocRoutes(o.openAPI.Title)

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
	okapitest.GET(t, "http://localhost:8080/openapi.json").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/openapi.yaml").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/docs").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/swagger").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/redoc").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/scalar").ExpectStatusOK()
}

// TestDocUISelection verifies that the UI rendered at /docs follows the
// configured selection (defaulting to Swagger UI), while each dedicated UI
// route stays available regardless of the selection.
func TestDocUISelection(t *testing.T) {
	tests := []struct {
		name   string
		ui     DocUI
		marker string // unique substring of the expected UI's HTML
	}{
		{"default is swagger", "", "swagger-ui"},
		{"swagger", SwaggerUI, "swagger-ui"},
		{"redoc", RedocUI, "redoc"},
		{"scalar", ScalarUI, "@scalar/api-reference"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := NewTestServer(t)
			ts.WithOpenAPIDocs(OpenAPI{UI: tt.ui})

			okapitest.GET(t, ts.BaseURL+"/docs").
				ExpectStatusOK().
				ExpectBodyContains(tt.marker)

			// Dedicated per-UI routes are always available.
			okapitest.GET(t, ts.BaseURL+"/swagger").ExpectStatusOK().ExpectBodyContains("swagger-ui")
			okapitest.GET(t, ts.BaseURL+"/redoc").ExpectStatusOK().ExpectBodyContains("redoc")
			okapitest.GET(t, ts.BaseURL+"/scalar").ExpectStatusOK().ExpectBodyContains("@scalar/api-reference")
		})
	}
}

// TestWithDocUIAfterDocs verifies the UI is resolved at request time, so
// WithDocUI takes effect even when called after WithOpenAPIDocs.
func TestWithDocUIAfterDocs(t *testing.T) {
	ts := NewTestServer(t)
	ts.WithOpenAPIDocs()
	ts.WithDocUI(ScalarUI)

	okapitest.GET(t, ts.BaseURL+"/docs").
		ExpectStatusOK().
		ExpectBodyContains("@scalar/api-reference")
}
