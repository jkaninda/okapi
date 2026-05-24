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

package main

import (
	"net/http"

	"github.com/jkaninda/okapi"
)

func main() {
	// Load every .html and .tmpl template from the "views" directory.
	// Alternatives:
	//   okapi.NewTemplateFromFiles("views/*.html")          // glob pattern
	//   okapi.Default().WithRendererFromDirectory("views", ".html")
	tmpl, err := okapi.NewTemplateFromDirectory("views", ".html", ".tmpl")
	if err != nil {
		panic(err)
	}

	// Create a new Okapi instance and register the renderer.
	o := okapi.Default().WithRenderer(tmpl)

	o.Get("/", func(c *okapi.Context) error {
		return c.Render(http.StatusOK, "hello", okapi.M{
			"title":   "Greeting Page",
			"message": "Hello, World!",
		})
	})

	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
