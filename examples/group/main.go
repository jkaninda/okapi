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
	"net/http"
)

type User struct {
	ID       int    `param:"id" query:"id" form:"id" json:"id" xml:"id" max:"10" `
	Name     string `json:"name" form:"name"  max:"15"`
	IsActive bool   `json:"is_active" query:"is_active" yaml:"isActive"`
}

var (
	users = []*User{
		{ID: 1, Name: "John Doe", IsActive: true},
		{ID: 2, Name: "Jonas Kaninda", IsActive: false},
		{ID: 3, Name: "Alice Johnson", IsActive: true},
	}
)

func main() {
	// Example usage of Group handling in Okapi
	// Create a new Okapi instance
	o := okapi.New(okapi.WithDebug())
	o.Get("/", func(c okapi.Context) error {
		// Handler logic for the root route
		return c.JSON(http.StatusOK, okapi.M{"message": "Welcome to Okapi!"})
	})
	// Create a new group with a base path
	api := o.Group("/api")

	v1 := api.Group("/v1")

	// Define a route with a handler
	v1.Get("/users", func(c okapi.Context) error {
		// Handler logic for the route
		return c.JSON(http.StatusOK, users)
	})
	// Get user
	v1.Get("/users/:id", show)
	// Update user
	v1.Put("/users/:id", update)
	// Create user
	v1.Post("/users", store)

	// Create a new group with a base path v2
	v2 := api.Group("/v2")
	// Define a route with a handler
	v2.Get("/users", func(c okapi.Context) error {
		c.SetHeader("Version", "v2")
		// Handler logic for the route
		return c.JSON(http.StatusOK, users)
	})
	v2.Get("/users/:id", func(c okapi.Context) error {
		c.SetHeader("Version", "v2")
		id := c.Param("id")
		for _, user := range users {
			if fmt.Sprintf("%d", user.ID) == id {
				return c.JSON(http.StatusOK, user)
			}
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	})

	// Start the server
	err := o.Start()
	if err != nil {
		panic(err)
	}
}
func store(c okapi.Context) error {
	var newUser User
	if ok, err := c.ShouldBind(&newUser); !ok {
		errMessage := fmt.Sprintf("Failed to bind user data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
	}
	// Add the new user to the list
	newUser.ID = len(users) + 1 // Simple ID assignment
	users = append(users, &newUser)
	// Respond with the created user
	return c.JSON(http.StatusCreated, newUser)
}
func show(c okapi.Context) error {
	var newUser User
	if ok, err := c.ShouldBind(&newUser); !ok {
		errMessage := fmt.Sprintf("Failed to bind user data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
	}
	for _, user := range users {
		if user.ID == newUser.ID {
			return c.JSON(http.StatusOK, user)
		}
	}
	return c.JSON(http.StatusNotFound, okapi.M{"error": "User not found"})
}
func update(c okapi.Context) error {
	var newUser User
	if ok, err := c.ShouldBind(&newUser); !ok {
		errMessage := fmt.Sprintf("Failed to bind user data: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input " + errMessage})
	}
	for _, user := range users {
		if user.ID == newUser.ID {
			user.Name = newUser.Name
			user.IsActive = newUser.IsActive
			return c.JSON(http.StatusOK, user)
		}
	}
	return c.JSON(http.StatusNotFound, okapi.M{"error": "User not found"})
}
