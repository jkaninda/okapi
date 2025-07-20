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
	"fmt"
	goutils "github.com/jkaninda/go-utils"
	"html/template"
	"io"
	"net/http"
	"os"
	"testing"
)

var content = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Hello</title>
</head>
<body>
{{define "hello"}}
<h1>{{.title}}</h1>
<p>{{.message}}</p>
{{end}}
</body>
</html>
`

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
func TestWithRenderer(t *testing.T) {
	createTemplate(t)
	temp := &Template{
		templates: template.Must(template.ParseGlob("public/*.html")),
	}
	o := New().WithRenderer(temp)
	o.Get("/", func(c Context) error {

		title := "Greeting Page"
		message := "Hello, World!"
		return c.Render(http.StatusOK, "hello", M{
			"title":   title,
			"message": message})
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

	assertStatus(t, "GET", fmt.Sprintf("%s/", testBaseURL), nil, nil, "", http.StatusOK)

}

func createTemplate(t *testing.T) {
	err := os.MkdirAll("public", 0777)
	if err != nil {
		t.Errorf("Failed to create public directory: %v", err)
	}
	err = goutils.WriteToFile("public/hello.html", content)
	if err != nil {
		t.Errorf("Failed to create file: %v", err)
	}

}
