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

package client_test

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/jkaninda/okapi/client"
)

func TestMiddlewareOrdering(t *testing.T) {
	var mu sync.Mutex
	var order []string
	mark := func(name string) client.Middleware {
		return func(next client.RoundTripFunc) client.RoundTripFunc {
			return func(req *http.Request) (*http.Response, error) {
				mu.Lock()
				order = append(order, name+":before")
				mu.Unlock()
				resp, err := next(req)
				mu.Lock()
				order = append(order, name+":after")
				mu.Unlock()
				return resp, err
			}
		}
	}

	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {})
	c := client.New(srv.URL,
		client.WithMiddleware(mark("a"), mark("b")),
	)
	if _, err := c.Get("/").Middleware(mark("c")).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := []string{
		"a:before", "b:before", "c:before",
		"c:after", "b:after", "a:after",
	}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-Id")
	})
	c := client.New(srv.URL, client.WithMiddleware(client.RequestIDMiddleware()))
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(got) == 0 {
		t.Error("X-Request-Id not set by middleware")
	}
}

func TestRequestIDMiddleware_PreservesExistingID(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Request-Id")
	})
	c := client.New(srv.URL, client.WithMiddleware(client.RequestIDMiddleware()))
	if _, err := c.Get("/").Header("X-Request-Id", "caller-supplied").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != "caller-supplied" {
		t.Errorf("X-Request-Id = %q, want caller-supplied", got)
	}
}

func TestUserAgentMiddleware_Overrides(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	})
	c := client.New(srv.URL,
		client.WithUserAgent("default"),
		client.WithMiddleware(client.UserAgentMiddleware("override")),
	)
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != "override" {
		t.Errorf("UA = %q, want override", got)
	}
}

func TestLoggingMiddleware_WritesLine(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	var buf bytes.Buffer
	c := client.New(srv.URL, client.WithMiddleware(client.LoggingMiddleware(&buf)))
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(buf.String(), "418") {
		t.Errorf("log output = %q", buf.String())
	}
}
