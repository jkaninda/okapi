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
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/client"
)

const serverAddr = ":8089"
const baseURL = "http://localhost:8089"

// User is shared between the server and the client.
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var users = []User{
	{ID: 1, Name: "Ada"},
	{ID: 2, Name: "Linus"},
}

func runServer() *okapi.Okapi {
	app := okapi.New(okapi.WithAddr(serverAddr))

	app.Get("/users", func(c *okapi.Context) error {
		return c.OK(users)
	})

	app.Get("/users/:id", func(c *okapi.Context) error {
		id := c.Param("id")
		for _, u := range users {
			if fmt.Sprintf("%d", u.ID) == id {
				return c.OK(u)
			}
		}
		return c.ErrorNotFound(okapi.M{"error": "user not found"})
	})

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	time.Sleep(150 * time.Millisecond) // wait for the listener to be ready
	return app
}

func main() {
	app := runServer()
	defer func() { _ = app.Stop() }()

	c := client.New(baseURL,
		client.WithUserAgent("okapi-client-example/1.0"),
		client.WithTimeout(5*time.Second),
		client.WithMiddleware(
			client.RequestIDMiddleware(),
			client.LoggingMiddleware(os.Stdout),
		),
		client.WithRetry(client.RetryPolicy{
			MaxAttempts: 3,
			BaseDelay:   100 * time.Millisecond,
			MaxDelay:    1 * time.Second,
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var list []User
	if err := c.Get("/users").WithContext(ctx).Decode(&list); err != nil {
		log.Fatalf("list users: %v", err)
	}
	fmt.Printf("got %d users\n", len(list))

	var ada User
	if err := c.Get("/users/1").WithContext(ctx).Decode(&ada); err != nil {
		log.Fatalf("get user 1: %v", err)
	}
	fmt.Printf("user 1 = %+v\n", ada)

	resp, err := c.Get("/users/9999").WithContext(ctx).Do()
	if err != nil {
		log.Fatalf("get user 9999: %v", err)
	}
	var hErr *client.HTTPError
	if errors.As(resp.Error(), &hErr) {
		fmt.Printf("missing user returned %d (body: %s)\n", hErr.StatusCode, hErr.Body)
	}
}
