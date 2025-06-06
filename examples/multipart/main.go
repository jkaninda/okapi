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
	"fmt"
	"github.com/jkaninda/okapi"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
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
	Name string `form:"name" json:"name" xml:"name" required:"true"`
	Age  int    `form:"age" json:"age" xml:"age" required:"true"`
	// Content-Type header for the multipart request
	Content string `header:"Content-Type" json:"content-type" xml:"content-type" required:"true"`
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
	o.Get("/", func(c okapi.Context) error {
		// Render the HTML template with dynamic data
		return c.HTMLView(200, template, okapi.M{
			"name":    "OKAPI Multipart Example",
			"message": "This is a multipart example using Okapi.",
			"upload":  "/upload",
		})
	})
	// Define a route for handling multipart form data
	o.Post("/upload", func(c okapi.Context) error {
		// Create an instance of MultipartBody to hold the form data
		multipartBody := &MultipartBody{}
		// Parse the multipart form data
		if err := c.Bind(multipartBody); err != nil {
			return c.ErrorBadRequest(okapi.M{
				"error":   "Failed to parse form data",
				"details": err.Error(),
			})
		}
		// Access the uploaded file
		file := *multipartBody.Avatar

		// Read the file content (for example, you can save it or process it)
		fileContent, err := file.Open()
		if err != nil {
			return c.JSON(400, okapi.M{"error": "Failed to open file"})
		}
		defer func(fileContent multipart.File) {
			err = fileContent.Close()
			if err != nil {
				slog.Error("Failed to close file content", "details", err)
			}
		}(fileContent)

		fileBytes := make([]byte, file.Size)
		_, err = fileContent.Read(fileBytes)
		if err != nil {
			return c.JSON(400, okapi.M{"error": "Failed to read file content"})
		}
		// Create a directory to save the uploaded file if it doesn't exist
		err = os.MkdirAll("uploads", 0755)
		if err != nil {
			return c.JSON(500, okapi.M{"error": "Failed to create uploads directory", "details": err.Error()})
		}

		f, err := os.Create(filepath.Join("uploads", file.Filename))
		if err != nil {
			return c.JSON(400, okapi.M{"error": "Failed to create file", "details": err.Error()})
		}
		defer func(f *os.File) {
			err = f.Close()
			if err != nil {
				slog.Error("Failed to close file", "error", err)
			}
		}(f)

		fmt.Printf("Tags: %v\n", multipartBody.Tags)
		// Write the file content to the created file
		_, err = f.Write(fileBytes)

		return c.HTMLView(200, successTemplate, okapi.M{})
	},
		okapi.DocRequestBody(MultipartBody{}),
	)

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}
