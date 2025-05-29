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
	"log"
	"net/http"
	"time"
)

func main() {
	// Initialize TLS configuration for secure HTTPS connections
	tls, err := okapi.LoadTLSConfig("path/to/cert.pem", "path/to/key.pem", "", false)
	if err != nil {
		panic(fmt.Sprintf("Failed to load TLS configuration: %v", err))
	}

	// Create a new Okapi instance with default config
	// Configured to listen on port 8080 for HTTP connections
	o := okapi.Default(okapi.WithAddr(":8080"))

	// Configure a secondary HTTPS server listening on port 8443
	// This creates both HTTP (8080) and HTTPS (8443) endpoints
	o.With(okapi.WithTLSServer(":8443", tls))

	// Register application routes and handlers

	o.Get("/", func(c okapi.Context) error {
		return c.JSON(http.StatusOK, okapi.M{
			"message": "Welcome to Okapi!",
			"status":  "operational",
		})
	})

	// Example parameterized route demonstrating path variables
	o.Get("/greeting/:name", greetingHandler)

	// Start the server(s)
	// This will launch both HTTP and HTTPS listeners in separate goroutines
	log.Println("Starting server on :8080 (HTTP) and :8443 (HTTPS)")
	if err := o.Start(); err != nil {
		panic(fmt.Sprintf("Server failed to start: %v", err))
	}
}

// greetingHandler handles personalized greeting requests
func greetingHandler(c okapi.Context) error {
	name := c.Param("name") // Extract name from URL path

	// Return personalized greeting as JSON
	return c.JSON(http.StatusOK, okapi.M{
		"message":    fmt.Sprintf("Hello %s!", name),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"user_agent": c.Request.UserAgent(),
	})
}
