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
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jkaninda/okapi"
)

// A simple HTML template for the multipart form with Name, Age, and avatar fields
const (
	template = `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>OKAPI Multipart Example</title>
</head>
<body>
<h1>{{.name}}</h1>
<p>{{.message}}</p>
<form action="{{.upload}}" method="post" enctype="multipart/form-data">
  <label for="name">Name:</label>
  <input type="text" id="name" name="name" required><br>
  <label for="age">Age:</label>
  <input type="number" id="age" name="age" required><br>
  <label for="avatar">Avatar:</label>
  <input type="file" id="avatar" name="avatar" accept="image/*" required><br>
  <label for="tags">Tags (comma-separated):</label>
  <input type="text" id="tags" name="tags"><br>
  <input type="submit" value="Upload">
</form>
</body>
</html>
`
	successTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Upload Success</title>
</head>
<body>
<h1>Upload Successful</h1>
<p>Your file has been uploaded successfully.</p>
</body>
</html>`
)

type MultipartBody struct {
	Name string `form:"name" json:"name" required:"true"`
	Age  int    `form:"age" json:"age" required:"true"`
	// Content-Type header for the multipart request
	Content string `header:"Content-Type" json:"content-type" required:"true"`
	// FileHeader for the uploaded file
	Avatar *multipart.FileHeader `form:"avatar"  required:"true"`
	// supports ?tags=a&tags=b or ?tags=a,b
	Tags []string `form:"tags" default:"a,b"`
}

func main() {
	// Example usage of multipart handling in Okapi
	// Create a new Okapi instance
	o := okapi.Default()
	// Define a route for the root path
	o.Get("/", func(c *okapi.Context) error {
		// Render the HTML template with dynamic data
		return c.HTMLView(http.StatusOK, template, okapi.M{
			"name":    "OKAPI Multipart Example",
			"message": "This is a multipart example using Okapi.",
			"upload":  "/upload",
		})
	})
	// Define a route for handling multipart form data
	o.Post("/upload", func(c *okapi.Context) error {
		// Bind the multipart form data into the struct
		form := &MultipartBody{}
		if err := c.Bind(form); err != nil {
			return c.ErrorBadRequest(okapi.M{
				"error":   "Failed to parse form data",
				"details": err.Error(),
			})
		}

		// Open the uploaded file
		src, err := form.Avatar.Open()
		if err != nil {
			return c.ErrorBadRequest(okapi.M{
				"error":   "Failed to read uploaded file",
				"details": err.Error(),
			})
		}
		defer func(src multipart.File) {
			if err := src.Close(); err != nil {
				slog.Error("Failed to close uploaded file", "error", err)
			}
		}(src)

		// Create the uploads directory if it doesn't exist
		if err := os.MkdirAll("uploads", 0o755); err != nil {
			return c.ErrorInternalServerError(okapi.M{"error": "Failed to create uploads directory", "details": err.Error()})
		}

		// Create the destination file and stream the upload into it
		dst, err := os.Create(filepath.Join("uploads", form.Avatar.Filename))
		if err != nil {
			return c.ErrorInternalServerError(okapi.M{"error": "Failed to create file", "details": err.Error()})
		}
		defer func(dst *os.File) {
			if err := dst.Close(); err != nil {
				slog.Error("Failed to close destination file", "error", err)
			}
		}(dst)

		if _, err := io.Copy(dst, src); err != nil {
			return c.ErrorInternalServerError(okapi.M{"error": "Failed to save file", "details": err.Error()})
		}

		slog.Info("Upload received",
			"name", form.Name,
			"age", form.Age,
			"tags", form.Tags,
			"file", form.Avatar.Filename,
		)
		return c.HTMLView(http.StatusOK, successTemplate, okapi.M{})
	},
		okapi.DocRequestBody(MultipartBody{}),
	)

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}
