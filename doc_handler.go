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
	"html/template"
	"net/http"
)

const (
	redoc = `

<!DOCTYPE html>
<html>
  <head>
    <title> {{.Title }}</title>
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
    <title> {{.Title }}</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.27.1/swagger-ui.css" />
<link rel="icon" type="image/png" sizes="32x32" href="https://unpkg.com/swagger-ui-dist@5.27.1/favicon-32x32.png">
<link rel="icon" type="image/png" sizes="16x16" href="https://unpkg.com/swagger-ui-dist@5.27.1/favicon-16x16.png">
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
)

var (
	redocTemplate   = template.Must(template.New("redoc").Parse(redoc))
	swaggerTemplate = template.Must(template.New("swagger").Parse(swagger))
)

func (o *Okapi) registerDocUIHandler(title string) {
	// Register the swagger route
	o.Get(openApiDocPrefix, func(c Context) error {
		return c.renderHTML(http.StatusOK, swaggerTemplate, M{"Title": title})
	},
	).internalRoute().Hide()
	// TODO: remove this route in the next major release
	o.Get("/docs/index.html", func(c Context) error {
		return c.renderHTML(http.StatusOK, swaggerTemplate, M{"Title": title})
	},
	).internalRoute().Hide()
	// Register the Redoc route
	o.Get("/redoc", func(c Context) error {
		return c.renderHTML(http.StatusOK, redocTemplate, M{"Title": title})
	},
	).internalRoute().Hide()
}
