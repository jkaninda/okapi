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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jkaninda/okapi/client"
)

const testName = "Ada"

func newServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestGET_HappyPath(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/users/42" {
			t.Errorf("path = %s, want /users/42", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"id":42,"name":"Ada"}`)
	})

	c := client.New(srv.URL)
	resp, err := c.Get("/users/42").Do()
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("status = %d, want 2xx", resp.StatusCode)
	}
	var user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := resp.JSON(&user); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if user.ID != 42 || user.Name != testName {
		t.Errorf("user = %+v, want id=42 name=%s", user, testName)
	}
}

func TestClient_DefaultHeadersAndUserAgent(t *testing.T) {
	var gotUA, gotAuth, gotCustom string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotAuth = r.Header.Get("Authorization")
		gotCustom = r.Header.Get("X-Tenant")
	})

	c := client.New(srv.URL,
		client.WithUserAgent("test-agent/1"),
		client.WithBearerToken("tok"),
		client.WithHeader("X-Tenant", "acme"),
	)
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotUA != "test-agent/1" {
		t.Errorf("UA = %q, want test-agent/1", gotUA)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Auth = %q, want Bearer tok", gotAuth)
	}
	if gotCustom != "acme" {
		t.Errorf("X-Tenant = %q, want acme", gotCustom)
	}
}

func TestQueryParams(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("a") != "1" || q.Get("b") != "2" {
			t.Errorf("query = %v, want a=1 b=2", q)
		}
	})
	c := client.New(srv.URL)
	if _, err := c.Get("/q").QueryParam("a", "1").QueryParams(map[string]string{"b": "2"}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestPOST_JSONRoundTrip(t *testing.T) {
	type In struct {
		Title string `json:"title"`
	}
	type Out struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var got In
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Out{ID: 1, Title: got.Title})
	})

	c := client.New(srv.URL)
	var out Out
	err := c.Post("/items").
		JSONBody(In{Title: "hello"}).
		Decode(&out)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.ID != 1 || out.Title != "hello" {
		t.Errorf("out = %+v, want id=1 title=hello", out)
	}
}

func TestNon2xx_ReturnsHTTPError(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "missing")
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/none").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.IsSuccess() {
		t.Fatalf("expected non-success status")
	}
	herr, ok := resp.Error().(*client.HTTPError)
	if !ok || herr == nil {
		t.Fatalf("Error type = %T, want *HTTPError", resp.Error())
	}
	if herr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", herr.StatusCode)
	}
	if string(herr.Body) != "missing" {
		t.Errorf("body = %q, want %q", string(herr.Body), "missing")
	}
}

func TestContextCancellation(t *testing.T) {
	block := make(chan struct{})
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		<-block
	})
	defer close(block)

	c := client.New(srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.Get("/slow").WithContext(ctx).Send()
		if err == nil {
			t.Errorf("expected error from cancelled context")
		}
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()
}

func TestPerRequestTimeout(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	})
	c := client.New(srv.URL, client.WithTimeout(time.Second))
	start := time.Now()
	_, err := c.Get("/").Timeout(30 * time.Millisecond).Send()
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Errorf("Send took too long, per-request timeout ignored")
	}
}

func TestDo_ArbitraryRequest(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		_, _ = io.WriteString(w, "ok")
	})
	c := client.New(srv.URL)
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/x", nil)
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if resp.String() != "ok" {
		t.Errorf("body = %q", resp.String())
	}
}

func TestClient_ConcurrentUse(t *testing.T) {
	var hits int64
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		_, _ = io.WriteString(w, "ok")
	})
	c := client.New(srv.URL)
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := c.Get("/").Send(); err != nil {
				t.Errorf("Send: %v", err)
			}
		}()
	}
	wg.Wait()
	if atomic.LoadInt64(&hits) != n {
		t.Errorf("hits = %d, want %d", hits, n)
	}
}
