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
	"log/slog"
	"net/http"
	"testing"
)

func TestGroup(t *testing.T) {
	o := Default()
	// create api group
	api := o.Group("/api").SetDisabled(false)
	// Okapi's Group Middleware
	api.Use(func(next HandleFunc) HandleFunc {
		return func(c Context) (err error) {
			slog.Info("Okapi's Group middleware")
			return next(c)
		}
	})
	// Go's standard HTTP middleware function
	api.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			handler.ServeHTTP(w, r)
		})

	})
	// Go's standard http.HandlerFunc
	api.HandleStd("GET", "/standard", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Calling route", "path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("standard standard http.HandlerFunc response"))
		if err != nil {
			return
		}
	})
	api.HandleHTTP("GET", "/standard-http", http.FileServer(http.Dir("static")))
	// Okapi Group HandleFun
	api.Get("hello", helloHandler)
	api.Post("hello", helloHandler)
	api.Put("hello", helloHandler)
	api.Patch("hello", helloHandler)
	api.Delete("hello", helloHandler)
	api.Options("hello", helloHandler)
	api.Head("hello", helloHandler)
	api.Trace("hello", helloHandler)
	api.Connect("hello", helloHandler)

	api.Get("/group", func(c Context) error {
		slog.Info("Calling route", "path", c.Request.URL.Path)
		return c.OK(M{"message": "Welcome to Okapi!"})
	})

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()

	waitForServer()

	assertStatus(t, "GET", "http://localhost:8080/api/group", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/standard", nil, nil, "", http.StatusOK)

	assertStatus(t, "GET", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "POST", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "PUT", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "PATCH", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "DELETE", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "OPTIONS", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "TRACE", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "CONNECT", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "HEAD", "http://localhost:8080/api/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/standard-http", nil, nil, "", http.StatusNotFound)
}
func helloHandler(c Context) error {
	slog.Info("Calling route", "path", c.Request.URL.Path, "method", c.Request.Method)
	return c.OK(M{"message": "Hello from Okapi!"})

}
