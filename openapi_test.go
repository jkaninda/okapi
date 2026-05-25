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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jkaninda/okapi/okapitest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type input struct {
	SessionId string `header:"Session-Id"`
	Body      string `json:"body"`
}
type output struct {
	Status   int    `json:"status"`
	ClientId string `header:"Client-Id"`
	Body     Book   `json:"body"`
}

func TestOpenAPI(t *testing.T) {
	o := Default()

	o.Get("/", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	},
		DocOperationId("getBook"),
		DocSummary("Root Endpoint"),
		DocDescription("This is the root endpoint of the API."),
		DocTags("Root"),
		DocResponse(200, M{"message": "Hello World!"}),
		DocResponse(http.StatusInternalServerError, M{"error": "Internal Server Error"}),
		DocBasicAuth(),
	)

	// create api group
	api := o.Group("api").WithBearerAuth()
	v1 := api.Group("v1")
	v2 := api.Group("v2")
	v1.Post("/books", anyHandler,
		DocSummary("Book Summary"),
		docAutoPathParams(),
		DocQueryParamWithDefault("auth", "string", "auth name", true, "defaultAuth"),
		DocHeaderWithDefault("X-Custom-Header", "string", "A custom header", false, "defaultHeaderValue"),
		DocBearerAuth(),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
	)
	v1.Put("/books", anyHandler,
		DocSummary("Book Summary"),
		docAutoPathParams(),
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
		DocQueryParamWithDefault("id", "int", "book id", true, 1),
		DocResponse(Book{}),
		DocRequestBody(Book{}),
		DocTags("Book Tags"),
		DocErrorResponse(http.StatusBadRequest, M{"": ""}),
		DocDeprecated(),
	)

	v1.Get("/books/{id}/comments", anyHandler,
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
		docAutoPathParams(),
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
			Response(http.StatusBadRequest, M{"": ""}).AsOption(),
	)
	v2.Get("/books", anyHandler,
		Doc().Summary("Book Summary").
			BearerAuth().
			QueryParam("auth", "string", "auth name", true).
			ResponseHeader("X-RateLimit-Limit", "int", "The number of requests allowed per minute").
			Response(200, Book{}).
			Tags("Book Tags").
			ErrorResponse(http.StatusBadRequest, M{"": ""}).Build(),
	).Hide()
	v2.Delete("/books/:id", anyHandler,
		Doc().Summary("Delete Book").
			Description("Delete a book by ID").
			BearerAuth().
			PathParam("id", "int", "book id").
			Response(Book{}).
			Tags("Book Tags").
			Response(http.StatusBadRequest, M{"": ""}).Build(),
	)
	v2.Post("/products", anyHandler,
		Doc().Summary("Create a product").
			BearerAuth().
			Response(&TestProduct{}).
			RequestBody(&TestProduct{}).
			Tags("Product").
			Response(http.StatusBadRequest, M{"": ""}).AsOption(),
	)
	// New Style
	apiV3 := api.Group("v3")
	apiV3.Post("/books", anyHandler).WithIO(&input{}, &output{})
	apiV3.Put("/books", anyHandler).WithInput(&input{})
	apiV3.Get("/books/:id", anyHandler).WithOutput(&output{})

	o.StartForTest(t)

	okapitest.GET(t, fmt.Sprintf("%s/docs", testBaseURL)).ExpectStatusOK()
	okapitest.GET(t, fmt.Sprintf("%s/openapi.json", testBaseURL)).ExpectStatusOK()

}
func TestNew(t *testing.T) {
	o := New()
	o.WithContext(context.Background())
	o.WithOpenAPIDocs(OpenAPI{
		Title:       "Okapi Web Framework Example",
		Version:     "1.0.0",
		Description: "Okapi Web Framework Example",
		Summary:     "Okapi Web Framework Example",
		License: License{
			Name: "MIT",
		},
		SecuritySchemes: SecuritySchemes{
			{
				Name:   "basicAuth",
				Type:   "http",
				Scheme: "basic",
			},
			{
				Name:         "bearerAuth",
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
			{
				Name: "X-API-KEY",
				Type: "apiKey",
				In:   "header",
			},
			{
				Name: "OAuth2",
				Type: "oauth2",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
		},
		Servers: Servers{
			{
				URL: "http://localhost:8080",
			},
		},
		ExternalDocs: &ExternalDocs{
			URL:         "http://localhost:8080/openapi.json",
			Description: "OpenAPI 2",
			Extensions:  map[string]interface{}{},
			Origin: &Origin{
				Key: &Location{
					Line:   1,
					Column: 1,
				},
				Fields: map[string]Location{
					"url": {
						Line: 2,
					},
					"description": {
						Line: 3,
					},
				},
			},
		},
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
	okapitest.GET(t, "http://localhost:8080/docs").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/openapi.json").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/openapi.yaml").ExpectStatusOK()
}
func TestWithOpenAPIDisabled(t *testing.T) {
	o := Default().WithOpenAPIDisabled().WithDebug()
	o.Get("/", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	}).Hide()
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
	okapitest.GET(t, "http://localhost:8080/docs").ExpectStatusNotFound()
	okapitest.GET(t, "http://localhost:8080/openapi.json").ExpectStatusNotFound()
	okapitest.GET(t, "http://localhost:8080/openapi.yaml").ExpectStatusNotFound()

}

func anyHandler(c *Context) error {
	slog.Info("Calling route", "path", c.Request().URL.Path, "method", c.request.Method)
	c.SetHeader("X-RateLimit-Limit", "100")
	return c.OK(M{"message": "Hello from Okapi!"})

}

// nullable31Model exercises 3.1-specific schema generation:
//   - Nickname is a pointer -> nullable
//   - Status carries a const tag
type nullable31Model struct {
	Name     string  `json:"name"`
	Nickname *string `json:"nickname"`
	Status   string  `json:"status" const:"active"`
}

// validateOpenAPIDoc round-trips the spec through the loader (resolving refs)
// and validates it. kin-openapi validates against the document's own version.
func validateOpenAPIDoc(t *testing.T, spec *openapi3.T) {
	t.Helper()
	data, err := spec.MarshalJSON()
	require.NoError(t, err)
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	require.NoError(t, err)
	require.NoError(t, doc.Validate(loader.Context))
}

func TestOpenAPI31Derivation(t *testing.T) {
	o := New()
	o.WithOpenAPIDocs(OpenAPI{
		Title:   "Okapi 3.1",
		Version: "1.0.0",
		License: License{Name: "MIT", Identifier: "MIT"},
		Servers: Servers{{URL: "http://localhost:8080"}},
	})
	o.Post("/things", anyHandler,
		DocSummary("Create thing"),
		DocRequestBody(&nullable31Model{}),
		DocResponse(200, &nullable31Model{}),
	)
	o.Webhook("newThing", http.MethodPost,
		DocSummary("Notify about a new thing"),
		DocRequestBody(&nullable31Model{}),
		DocResponse(200, M{"received": true}),
	)
	// Rebuild so the routes and webhook registered above are included.
	o.buildOpenAPISpec()

	spec30 := o.openapiSpec
	spec31 := o.openapiSpec31

	// Version + dialect.
	assert.Equal(t, "3.0.3", spec30.OpenAPI)
	assert.Empty(t, spec30.JSONSchemaDialect)
	assert.Equal(t, "3.1.0", spec31.OpenAPI)
	assert.Equal(t, jsonSchemaDialect, spec31.JSONSchemaDialect)

	// Webhooks only on 3.1.
	assert.Empty(t, spec30.Webhooks)
	require.Contains(t, spec31.Webhooks, "newThing")
	assert.NotNil(t, spec31.Webhooks["newThing"].Post)

	// SPDX license identifier only on 3.1; url cleared there.
	assert.Empty(t, spec30.Info.License.Identifier)
	require.NotNil(t, spec31.Info.License)
	assert.Equal(t, "MIT", spec31.Info.License.Identifier)

	// Nullable + const representation.
	model30 := spec30.Components.Schemas["nullable31Model"].Value
	model31 := spec31.Components.Schemas["nullable31Model"].Value
	require.NotNil(t, model30)
	require.NotNil(t, model31)

	// 3.0: nullable flag, no const, no marker leakage.
	nick30 := model30.Properties["nickname"].Value
	assert.True(t, nick30.Nullable)
	assert.False(t, nick30.Type.Includes("null"))
	status30 := model30.Properties["status"].Value
	assert.Nil(t, status30.Const)
	assert.NotContains(t, status30.Extensions, extOkapiConst)

	// 3.1: type array, no nullable flag, const promoted, marker removed.
	nick31 := model31.Properties["nickname"].Value
	assert.False(t, nick31.Nullable)
	assert.True(t, nick31.Type.Includes("null"))
	assert.True(t, nick31.Type.Includes("string"))
	status31 := model31.Properties["status"].Value
	assert.Equal(t, "active", status31.Const)
	assert.NotContains(t, status31.Extensions, extOkapiConst)

	// Both documents must be valid under their own version.
	validateOpenAPIDoc(t, spec30)
	validateOpenAPIDoc(t, spec31)
}

func TestOpenAPI31Endpoints(t *testing.T) {
	app := Default().WithOpenAPIDocs(OpenAPI{
		Title:   "Okapi 3.1 Endpoints",
		Version: "1.0.0",
		License: License{Name: "MIT", Identifier: "MIT"},
	})
	o := NewTestServerWithOkapi(t, app)
	o.Post("/things", anyHandler).WithIO(&nullable31Model{}, &nullable31Model{})

	// Default endpoints serve OpenAPI 3.1.
	okapitest.GET(t, fmt.Sprintf("%s/openapi.json", o.BaseURL)).
		ExpectStatusOK().ExpectJSONPath("openapi", "3.1.0")
	okapitest.GET(t, fmt.Sprintf("%s/openapi.yaml", o.BaseURL)).
		ExpectStatusOK().ExpectBodyContains("openapi: 3.1.0")

	// Version-pinned 3.0 endpoints (preserved).
	okapitest.GET(t, fmt.Sprintf("%s/openapi-3.0.json", o.BaseURL)).
		ExpectStatusOK().ExpectJSONPath("openapi", "3.0.3")
	okapitest.GET(t, fmt.Sprintf("%s/openapi-3.0.yaml", o.BaseURL)).
		ExpectStatusOK().ExpectBodyContains("openapi: 3.0.3")

	okapitest.GET(t, fmt.Sprintf("%s/docs/favicon.png", o.BaseURL)).
		ExpectStatusOK().ExpectContentType("image/png")
	okapitest.GET(t, fmt.Sprintf("%s/scalar", o.BaseURL)).
		ExpectStatusOK().ExpectBodyContains(`href="/docs/favicon.png"`)
}

func TestOpenAPICustomFavicon(t *testing.T) {
	app := Default().WithOpenAPIDocs(OpenAPI{
		Title:   "Custom Favicon",
		Favicon: "https://example.com/icon.png",
	})
	o := NewTestServerWithOkapi(t, app)
	o.Get("/", anyHandler)

	// Custom favicon is referenced; the embedded route is not registered.
	okapitest.GET(t, fmt.Sprintf("%s/swagger", o.BaseURL)).
		ExpectStatusOK().ExpectBodyContains(`href="https://example.com/icon.png"`)
	okapitest.GET(t, fmt.Sprintf("%s/docs/favicon.png", o.BaseURL)).ExpectStatusNotFound()
}

func TestWithOpenAPIDocs(t *testing.T) {
	app := Default().
		WithOpenAPIDocs(OpenAPI{
			Title:   "Okapi Web Framework Example",
			Version: "1.0.0",
			License: License{
				Name: "MIT",
			},
			Servers: Servers{
				{
					URL: "http://localhost:8080",
				},
			},
			ExternalDocs: &ExternalDocs{
				URL: "http://localhost:8080/openapi.json",
			},
		})
	o := NewTestServerWithOkapi(t, app)

	o.Get("/", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	}).WithIO(&SliceRequest{}, &SliceRequest{})
	okapitest.GET(t, fmt.Sprintf("%s/", o.BaseURL)).ExpectStatusOK()
	okapitest.GET(t, fmt.Sprintf("%s/docs", o.BaseURL)).ExpectStatusOK()
	okapitest.GET(t, fmt.Sprintf("%s/openapi.json", o.BaseURL)).ExpectStatusOK()
	okapitest.GET(t, fmt.Sprintf("%s/redoc", o.BaseURL)).ExpectStatusOK()
}

// validationKeywordsModel exercises the OpenAPI emission of the new validation
// keywords: exclusive numeric bounds, string format, and object property counts.
type validationKeywordsModel struct {
	Age     int               `json:"age" exclusiveMin:"0" exclusiveMax:"150"`
	Website string            `json:"website" format:"url"`
	Labels  map[string]string `json:"labels" minProperties:"1" maxProperties:"5"`
}

func TestOpenAPIValidationKeywords(t *testing.T) {
	o := New()
	o.WithOpenAPIDocs(OpenAPI{
		Title:   "Validation Keywords",
		Version: "1.0.0",
		License: License{Name: "MIT"},
		Servers: Servers{{URL: "http://localhost:8080"}},
	})
	o.Post("/things", anyHandler,
		DocRequestBody(&validationKeywordsModel{}),
		DocResponse(200, &validationKeywordsModel{}),
	)
	o.buildOpenAPISpec()

	spec30 := o.openapiSpec
	spec31 := o.openapiSpec31

	m30 := spec30.Components.Schemas["validationKeywordsModel"].Value
	m31 := spec31.Components.Schemas["validationKeywordsModel"].Value
	require.NotNil(t, m30)
	require.NotNil(t, m31)

	// 3.0: exclusive bounds are minimum/maximum + boolean exclusive flags.
	age30 := m30.Properties["age"].Value
	require.NotNil(t, age30.Min)
	assert.Equal(t, 0.0, *age30.Min)
	assert.True(t, age30.ExclusiveMin.IsTrue())
	require.NotNil(t, age30.Max)
	assert.Equal(t, 150.0, *age30.Max)
	assert.True(t, age30.ExclusiveMax.IsTrue())

	// 3.1: exclusive bounds are numeric; minimum/maximum cleared.
	age31 := m31.Properties["age"].Value
	assert.Nil(t, age31.Min)
	require.NotNil(t, age31.ExclusiveMin.Value)
	assert.Equal(t, 0.0, *age31.ExclusiveMin.Value)
	assert.Nil(t, age31.Max)
	require.NotNil(t, age31.ExclusiveMax.Value)
	assert.Equal(t, 150.0, *age31.ExclusiveMax.Value)

	// format flows through from the format tag (previously a no-op).
	assert.Equal(t, "url", m30.Properties["website"].Value.Format)
	assert.Equal(t, "url", m31.Properties["website"].Value.Format)

	// object property counts.
	labels30 := m30.Properties["labels"].Value
	assert.Equal(t, uint64(1), labels30.MinProps)
	require.NotNil(t, labels30.MaxProps)
	assert.Equal(t, uint64(5), *labels30.MaxProps)

	// Both documents must validate under their own version.
	validateOpenAPIDoc(t, spec30)
	validateOpenAPIDoc(t, spec31)
}
