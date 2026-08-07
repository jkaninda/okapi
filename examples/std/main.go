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

// Standard library interop.
//
// Okapi speaks net/http: existing handlers and middleware drop in unchanged,
// so an existing application can be adopted one route at a time. This example
// registers the same server three ways — standard middleware, standard
// handlers, and native Okapi handlers — and shows them sharing one middleware
// chain.
//
// Run it, then try:
//
//	curl localhost:8080/greeting
//	curl localhost:8080/users/42          # r.PathValue inside a std handler
//	curl localhost:8080/api/legacy        # std handler behind group middleware
//	curl localhost:8080/assets/style.css  # http.FileServer on a catch-all
//	curl localhost:8080/users             # native handler, for contrast
//	curl -i localhost:8080/greeting       # note X-Powered-By from std middleware
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/jkaninda/okapi"
)

//go:embed all:static
var staticFS embed.FS

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func main() {
	o := okapi.Default()

	// 1. Standard middleware: func(http.Handler) http.Handler.
	//
	// This is the shape used by chi, rs/cors and most of the community
	// ecosystem. It applies to every route below, standard and native alike.
	o.UseMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Powered-By", "Okapi")
			next.ServeHTTP(w, r)
		})
	})

	// 2. A standard http.HandlerFunc, registered with HandleStd.
	//
	// It still gets routing, the middleware chain and CORS registration. The
	// documentation options are the same ones a native route accepts.
	o.HandleStd(http.MethodGet, "/greeting", greeting,
		okapi.DocSummary("Greet the caller"),
		okapi.DocTag("std"))

	// 3. Path parameters work as they do under http.ServeMux: Okapi copies what
	// the router captured onto the request, so r.PathValue reads it.
	o.HandleStd(http.MethodGet, "/users/{id}", getUser,
		okapi.DocSummary("Fetch a user by id"),
		okapi.DocTag("std"))

	// 4. Any http.Handler, registered with HandleHTTP. A catch-all segment
	// ({any...}) hands the prefix and everything beneath it to one handler,
	// which is how a standard file server is mounted.
	assets, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	o.HandleHTTP(http.MethodGet, "/assets/{any...}",
		http.StripPrefix("/assets/", http.FileServer(http.FS(assets))),
		okapi.DocHide())

	// 5. Groups expose the same two methods, so a standard handler inherits the
	// group's prefix and middleware.
	api := o.Group("/api")
	api.Use(func(c *okapi.Context) error {
		c.Response().Header().Set("X-Group", "api")
		return c.Next()
	})
	api.HandleStd(http.MethodGet, "/legacy", legacy, okapi.DocTag("std"))

	// 6. Native handlers sit alongside them, and are where binding, validation
	// and the error-returning signature become available.
	o.Get("/users", func(c *okapi.Context) error {
		return c.OK([]User{{ID: "42", Name: "Ada Lovelace"}})
	}, okapi.DocSummary("List users"), okapi.DocTag("native"))

	if err := o.Start(); err != nil {
		log.Fatal(err)
	}
}

func greeting(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, okapi.M{"message": "Hello from net/http"})
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, okapi.M{"error": "missing id"})
		return
	}
	writeJSON(w, http.StatusOK, User{ID: id, Name: "Ada Lovelace"})
}

// legacy stands in for a handler you have not migrated yet.
func legacy(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, okapi.M{"message": "legacy code, unchanged"})
}

// writeJSON is the boilerplate a standard handler carries. A native Okapi
// handler would write the same response with `return c.OK(v)`.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}
