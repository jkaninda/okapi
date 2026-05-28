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

// Package okapi is a modern, minimalist HTTP web framework for Go, designed for simplicity, performance,
// and developer happiness, it helps you build fast, scalable, and well-documented APIs
// with minimal boilerplate.
//
// The framework is named after the okapi (/oʊˈkɑːpiː/), a rare and graceful mammal
// native to the rainforests of the northeastern Democratic Republic of the Congo.
// Just like its namesake — which resembles a blend of giraffe and zebra — Okapi blends
// simplicity and strength in a unique, powerful package.
//
// Key Features:
//
//   - Intuitive & Expressive API:
//     Clean, declarative syntax for effortless route and middleware definition.
//
//   - Automatic Request Binding:
//     Seamlessly parse JSON, XML, form data, query params, headers, and path variables into structs.
//
//   - Built-in Auth & Security:
//     Native support for JWT, OAuth2, Basic Auth, and custom middleware.
//
//   - First-Class Documentation:
//     OpenAPI 3.1 (default) and 3.0 with Swagger UI, ReDoc, and Scalar integrated out of the box—auto-generate
//     API docs with minimal effort and pick the UI rendered at /docs. The default spec at /openapi.json and
//     /openapi.yaml is OpenAPI 3.1; the 3.0 spec is also served at /openapi-3.0.{json,yaml}.
//
//   - Modern Tooling:
//     Route grouping, middleware chaining, static file serving, templating engine support,
//     CORS management, fine-grained timeout controls.
//
//   - Developer Experience:
//     Minimal boilerplate, clear error handling, structured logging, and easy testing.
//
// Okapi is built for speed, simplicity, and real-world use—whether you're prototyping or running in production.
//
// For more information and documentation, visit: https://github.com/jkaninda/okapi
package okapi

import (
	_ "embed"
	"html/template"
	"net/http"
)

// okapiFavicon is the default favicon served for the documentation UIs at
// docFaviconPath: a 32x32 Okapi logo, embedded so the docs render the same
// branding offline and without any external (CDN) dependency.
//
//go:embed favicon.png
var okapiFavicon []byte

const (
	redoc = `

<!DOCTYPE html>
<html>
  <head>
    <title>{{.Title}} | ReDoc</title>
    <link rel="icon" type="image/png" href="{{.Favicon}}">
    <!-- needed for adaptive design -->
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <!--
    Redoc doesn't change outer page styles
    -->
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <redoc spec-URL='/openapi.json'></redoc>
    <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"> </script>
  </body>
</html>
`
	swagger = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="SwaggerUI" />
    <title>{{.Title}} | Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.27.1/swagger-ui.css" />
<link rel="icon" type="image/png" href="{{.Favicon}}">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5.27.1/swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="https://unpkg.com/swagger-ui-dist@5.27.1/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script src="https://unpkg.com/swagger-ui-dist@5.27.1/swagger-initializer.js" charset="UTF-8"></script>
<script>
  window.onload = () => {
    window.ui = SwaggerUIBundle({
      url: '/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
    });
  };
</script>
</body>
</html>
`
	scalar = `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content="Scalar API Reference" />
    <title>{{.Title}} | Scalar</title>
    <link rel="icon" type="image/png" href="{{.Favicon}}">
  </head>
  <body>
    <div id="app"></div>
    <!-- Load the Script -->
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    
    <!-- Initialize the Scalar API Reference -->
    <script>
      Scalar.createApiReference('#app', {
        url: '/openapi.json',
      });
    </script>
  </body>
</html>
`
)

// DocUI identifies which interactive documentation UI is rendered at /docs.
type DocUI string

const (
	// SwaggerUI renders Swagger UI. This is the default.
	SwaggerUI DocUI = "swagger"
	// RedocUI renders ReDoc.
	RedocUI DocUI = "redoc"
	// ScalarUI renders Scalar API Reference.
	ScalarUI DocUI = "scalar"
)

var (
	redocTemplate   = template.Must(template.New("redoc").Parse(redoc))
	swaggerTemplate = template.Must(template.New("swagger").Parse(swagger))
	scalarTemplate  = template.Must(template.New("scalar").Parse(scalar))
)

// selectedDocUI returns the UI rendered at /docs, normalized to a known value.
// It falls back to Swagger UI when unset or unrecognized.
func (o *Okapi) selectedDocUI() DocUI {
	switch o.openAPI.UI {
	case RedocUI, ScalarUI, SwaggerUI:
		return o.openAPI.UI
	default:
		return SwaggerUI
	}
}

// docsTemplate returns the template for the UI selected via OpenAPI.UI
// (or WithDocUI). It falls back to Swagger UI when unset or unrecognized.
func (o *Okapi) docsTemplate() *template.Template {
	switch o.selectedDocUI() {
	case RedocUI:
		return redocTemplate
	case ScalarUI:
		return scalarTemplate
	default:
		return swaggerTemplate
	}
}

// registerDocRoutes registers the OpenAPI documentation routes for the Okapi instance.
func (o *Okapi) registerDocRoutes(title string) {
	favicon := o.openAPI.Favicon
	if favicon == "" {
		favicon = docFaviconPath
		o.Get(docFaviconPath, func(c *Context) error {
			return c.Data(http.StatusOK, "image/png", okapiFavicon)
		}).internalRoute().Hide()
	}
	docData := M{"Title": title, "Favicon": favicon}
	// Default OpenAPI routes serve the latest version (3.1).
	o.Get(openApiDocPath, func(c *Context) error {
		return c.JSON(http.StatusOK, o.openapiSpec31)
	}).internalRoute().Hide() // Hide the route from the OpenAPI documentation
	o.Get(openApiYamlPath, func(c *Context) error {
		return c.YAML(http.StatusOK, o.openapiSpec31)
	}).internalRoute().Hide()
	// Version-pinned OpenAPI 3.0 routes
	o.Get(openApiDocPath30, func(c *Context) error {
		return c.JSON(http.StatusOK, o.openapiSpec)
	}).internalRoute().Hide()
	o.Get(openApiYamlPath30, func(c *Context) error {
		return c.YAML(http.StatusOK, o.openapiSpec)
	}).internalRoute().Hide()
	// Register the main docs route.
	o.Get(openApiDocPrefix, func(c *Context) error {
		return c.renderHTML(http.StatusOK, o.docsTemplate(), docData)
	},
	).internalRoute().Hide() // Hide the route from the OpenAPI documentation
	// TODO: remove this route in the next major release
	o.Get("/docs/index.html", func(c *Context) error {
		return c.renderHTML(http.StatusOK, o.docsTemplate(), docData)
	},
	).internalRoute().Hide() // Hide the route from the OpenAPI documentation

	if o.openAPI.StrictDocUI {
		return
	}
	// Register the Swagger UI route
	o.Get(docSwaggerPath, func(c *Context) error {
		return c.renderHTML(http.StatusOK, swaggerTemplate, docData)
	},
	).internalRoute().Hide() // Hide the route from the OpenAPI documentation
	// Register the Redoc route
	o.Get(docRedocPath, func(c *Context) error {
		return c.renderHTML(http.StatusOK, redocTemplate, docData)
	},
	).internalRoute().Hide() // Hide the route from the OpenAPI documentation
	// Register the Scalar route
	o.Get(docScalarPath, func(c *Context) error {
		return c.renderHTML(http.StatusOK, scalarTemplate, docData)
	},
	).internalRoute().Hide() // Hide the route from the OpenAPI documentation
}
