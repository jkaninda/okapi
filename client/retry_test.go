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
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jkaninda/okapi/client"
)

func TestRetry_On5xxSucceedsEventually(t *testing.T) {
	var hits int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   5 * time.Millisecond,
	}))
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !resp.IsSuccess() {
		t.Errorf("status = %d, want success", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3", got)
	}
}

func TestRetry_DoesNotRetry4xx(t *testing.T) {
	var hits int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
	}))
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("hits = %d, want 1 (no retry on 400)", got)
	}
}

func TestRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	var hits int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
	}))
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3", got)
	}
}

func TestRetry_CustomShouldRetry(t *testing.T) {
	var hits int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadRequest) // not normally retried
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond,
		ShouldRetry: func(resp *http.Response, err error) bool {
			return resp != nil && resp.StatusCode == http.StatusBadRequest
		},
	}))
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Errorf("hits = %d, want 3", got)
	}
}

func TestRetry_ContextCancelStopsBackoff(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   500 * time.Millisecond,
	}))
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	start := time.Now()
	_, err := c.Get("/").WithContext(ctx).Send()
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if time.Since(start) > 400*time.Millisecond {
		t.Errorf("Send blocked longer than expected: %v", time.Since(start))
	}
}

func TestRetry_PerRequestOverride(t *testing.T) {
	var hits int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	c := client.New(srv.URL) // no client-level retry
	if _, err := c.Get("/").Retry(client.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
	}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Errorf("hits = %d, want 2 (per-request retry honored)", got)
	}
}

func TestRetry_BodyRewound(t *testing.T) {
	var bodies []string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	c := client.New(srv.URL, client.WithRetry(client.RetryPolicy{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
	}))
	if _, err := c.Post("/").JSONBody(map[string]string{"k": "v"}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Errorf("bodies = %v, want two identical entries", bodies)
	}
}
