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
	"embed"
	"net/http"
	"strconv"
	"time"

	"github.com/jkaninda/okapi"
)

//go:embed all:web
var webFS embed.FS

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var users = []User{
	{ID: 1, Name: "Jonas", Email: "jonas@example.com"},
	{ID: 2, Name: "Sam", Email: "sam@example.com"},
	{ID: 3, Name: "Josh", Email: "josh@example.com"},
}

func main() {
	o := okapi.Default()

	api := o.Group("/api/v1")
	api.Get("/health", func(c *okapi.Context) error {
		return c.OK(okapi.M{"status": "ok"})
	})

	// List all users.
	api.Get("/users", func(c *okapi.Context) error {
		return c.OK(okapi.M{"users": users})
	})

	// Get a single user by ID
	api.Get("/users/:id", func(c *okapi.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, okapi.M{"error": "invalid user id"})
		}
		for _, u := range users {
			if u.ID == id {
				return c.OK(u)
			}
		}
		return c.JSON(http.StatusNotFound, okapi.M{"error": "user not found"})
	})

	// Serve the SPA. Real files (index.html, assets/*) are served directly

	// any other path falls back to index.html so the client-side router takes
	// over. Try "/", "/dashboard", "/users/2".
	// Embedded (single binary) — recommended for deployments:
	o.WebFS("/", webFS, okapi.WebConfig{
		Root:   "web",     // sub-directory inside the embed.FS
		MaxAge: time.Hour, // Cache-Control for assets; index.html stays no-cache
	})

	// From disk — during front-end development:
	//
	//	o.Web("/", "./web", okapi.WebConfig{MaxAge: time.Hour})

	// Visit http://localhost:8080
	if err := o.Start(); err != nil {
		panic(err)
	}
}
