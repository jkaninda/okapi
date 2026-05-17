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
	"net/http"
	"strings"
	"testing"

	"github.com/jkaninda/okapi/client"
)

func TestVerbShortcuts(t *testing.T) {
	var gotMethod string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
	})
	c := client.New(srv.URL)

	cases := []struct {
		name string
		fn   func() error
		want string
	}{
		{"PUT", func() error { _, err := c.Put("/").Send(); return err }, http.MethodPut},
		{"PATCH", func() error { _, err := c.Patch("/").Send(); return err }, http.MethodPatch},
		{"DELETE", func() error { _, err := c.Delete("/").Send(); return err }, http.MethodDelete},
		{"HEAD", func() error { _, err := c.Head("/").Send(); return err }, http.MethodHead},
		{"OPTIONS", func() error { _, err := c.Options("/").Send(); return err }, http.MethodOptions},
		{"Request", func() error { _, err := c.Request("PROPFIND", "/").Send(); return err }, "PROPFIND"},
	}
	for _, tc := range cases {
		if err := tc.fn(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if gotMethod != tc.want {
			t.Errorf("%s: method = %s, want %s", tc.name, gotMethod, tc.want)
		}
	}
}

func TestClient_BaseURL(t *testing.T) {
	c := client.New("https://example.com/api/")
	if got := c.BaseURL(); got != "https://example.com/api" {
		t.Errorf("BaseURL = %q", got)
	}
}

func TestOptions_HTTPClientHeadersBasicAuth(t *testing.T) {
	var gotAuth, gotXTrace string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotXTrace = r.Header.Get("X-Trace")
	})
	custom := &http.Client{}
	c := client.New(srv.URL,
		client.WithHTTPClient(custom),
		client.WithHeaders(map[string]string{"X-Trace": "abc"}),
		client.WithBasicAuth("u", "p"),
	)
	if _, err := c.Get("/").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("Auth = %q", gotAuth)
	}
	if gotXTrace != "abc" {
		t.Errorf("X-Trace = %q", gotXTrace)
	}
}

func TestRawAndReaderBody(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 16)
		n, _ := r.Body.Read(buf)
		got = string(buf[:n])
	})
	c := client.New(srv.URL)
	if _, err := c.Post("/").RawBody([]byte("raw-bytes")).Send(); err != nil {
		t.Fatalf("Send raw: %v", err)
	}
	if got != "raw-bytes" {
		t.Errorf("raw body = %q", got)
	}

	if _, err := c.Post("/").Body(strings.NewReader("reader-body")).Send(); err != nil {
		t.Fatalf("Send body: %v", err)
	}
	if got != "reader-body" {
		t.Errorf("reader body = %q", got)
	}
}

func TestRequestHeadersMap(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Tenant")
	})
	c := client.New(srv.URL)
	if _, err := c.Get("/").Headers(map[string]string{"X-Tenant": "acme"}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != "acme" {
		t.Errorf("X-Tenant = %q", got)
	}
}

func TestRoundTripFunc_RoundTrip(t *testing.T) {
	called := false
	var fn client.RoundTripFunc = func(*http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 204, Body: http.NoBody}, nil
	}
	resp, err := fn.RoundTrip(&http.Request{})
	if err != nil || resp.StatusCode != 204 || !called {
		t.Errorf("RoundTrip didn't dispatch: err=%v status=%d called=%v", err, resp.StatusCode, called)
	}
}

func TestHTTPError_Message(t *testing.T) {
	e := &client.HTTPError{StatusCode: 404, Status: "404 Not Found", Method: "GET", URL: "/x"}
	if !strings.Contains(e.Error(), "404") {
		t.Errorf("Error() = %q", e.Error())
	}
	e.Body = []byte("missing")
	if !strings.Contains(e.Error(), "missing") {
		t.Errorf("Error() with body = %q", e.Error())
	}
}

func TestDecode_NonSuccessReturnsHTTPError(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	c := client.New(srv.URL)
	var out struct{}
	err := c.Get("/").Decode(&out)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*client.HTTPError); !ok {
		t.Errorf("err type = %T, want *HTTPError", err)
	}
}

func TestSendIsAliasForDo(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.String() != "ok" {
		t.Errorf("body = %q", resp.String())
	}
}

func TestJoinURL_AbsoluteAndEmptyPath(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	// path empty -> hits base
	c := client.New(srv.URL)
	if _, err := c.Get("").Send(); err != nil {
		t.Fatalf("empty path: %v", err)
	}

	// absolute path -> bypasses base
	c2 := client.New("http://will-not-be-used.invalid")
	if _, err := c2.Get(srv.URL + "/abs").Send(); err != nil {
		t.Fatalf("absolute path: %v", err)
	}
}
