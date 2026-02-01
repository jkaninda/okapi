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
	"github.com/jkaninda/okapi"
	"html/template"
	"io"
	"net/http"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c *okapi.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	tmpl, _ := okapi.NewTemplateFromDirectory("public/views", ".html", ".tmpl")
	// Or
	// tmpl, _ := okapi.NewTemplateFromFiles("public/views/*.html")

	// Example usage of the Okapi framework
	// Create a new Okapi instance and set renderer
	o := okapi.Default().WithRenderer(tmpl)
	// or
	// tmpl := &Template{
	//	templates: template.Must(template.ParseGlob("templates/*.html")),
	// }
	// o.With().WithRenderer(tmpl)
	// or you can use a custom renderer function

	/*
		o.renderer = okapi.RendererFunc(func(w io.Writer, name string, data interface{}, c *okapi.Context) error {
			// Render the template with the provided data
			tmpl, err := template.ParseFiles("public/views/" + name + ".html")
			if err != nil {
				return err
			}
			return tmpl.ExecuteTemplate(w, name, data)
		})
	*/
	o.Get("/", func(c *okapi.Context) error {

		title := "Greeting Page"
		message := "Hello, World!"
		return c.Render(http.StatusOK, "hello", okapi.M{
			"title":   title,
			"message": message})
	})

	// Start the server
	err := o.Start()
	if err != nil {
		return
	}
}
