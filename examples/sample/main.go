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
	"net/http"
	"time"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
type ErrorResponse struct {
	Success bool        `json:"success"`
	Status  int         `json:"status"`
	Message interface{} `json:"message"`
}

func main() {
	// Create a new Okapi instance with default config
	o := okapi.Default()

	o.Get("/", func(c okapi.Context) error {
		resp := Response{
			Success: true,
			Message: "Welcome to Okapi!",
			Data: okapi.M{
				"name":    "Okapi Web Framework",
				"Licence": "MIT",
				"date":    time.Now(),
			},
		}
		return c.OK(resp)
	},
		// OpenAPI Documentation
		okapi.DocSummary("Welcome page"),
		okapi.DocResponse(Response{}),                                  // Success Response body
		okapi.DocErrorResponse(http.StatusBadRequest, ErrorResponse{}), // Error response body
	)
	o.Get("/greeting/:name", greetingHandler)

	// Start the server
	if err := o.Start(); err != nil {
		panic(err)
	}
}
func greetingHandler(c okapi.Context) error {
	name := c.Param("name")
	if name == "" {
		errorResponse := ErrorResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "name is empty",
		}
		return c.ErrorBadRequest(errorResponse)
	}
	return c.OK(okapi.M{"message": "Hello " + name})
}
