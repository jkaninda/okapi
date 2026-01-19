/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type TestServer struct {
	*Okapi
	BaseURL string
	t       TestingT
}

type TestingT interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// NewTestContext creates a Context with its own in-memory request and recorder.
// Unlike NewContext, it does not initialize a default Okapi engine.
func NewTestContext(method, url string, body io.Reader) (*Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, url, body)
	w := httptest.NewRecorder()

	ctx := &Context{
		request:  req,
		okapi:    nil,
		response: &responseWriter{writer: w},
		store:    newStoreData(),
	}

	return ctx, w
}

// NewTestServer creates and starts a new Okapi test server.
//
// Example:
//
// testServer := okapi.NewTestServer(t)
//
// testServer.Get("/books", GetBooksHandler)
//
// okapitest.GET(t, testServer.BaseURL+"/books").ExpectStatusOK().ExpectBodyContains("The Go Programming Language")
func NewTestServer(t TestingT) *TestServer {
	t.Helper()
	o := New()
	baseURL := o.StartForTest(t)

	return &TestServer{
		Okapi:   o,
		BaseURL: baseURL,
		t:       t,
	}
}

// NewTestServerOn creates and starts a new Okapi test server.
//
// Example:
//
// testServer := okapi.NewTestServerOn(t,80801)
//
// testServer.Get("/books", GetBooksHandler)
//
// okapitest.GET(t, testServer.BaseURL+"/books").ExpectStatusOK().ExpectBodyContains("The Go Programming Language")
func NewTestServerOn(t TestingT, port int) *TestServer {
	t.Helper()
	o := New(WithPort(port))
	baseURL := o.StartForTest(t)

	return &TestServer{
		Okapi:   o,
		BaseURL: baseURL,
		t:       t,
	}
}

// StartForTest starts the Okapi server for testing and returns the base URL.
func (o *Okapi) StartForTest(t TestingT) string {
	t.Helper()
	ready := make(chan struct{})
	if o == nil || o.server == nil {
		t.Fatalf("Okapi instance or server is nil")
		return ""
	}
	go func() {
		if o == nil || o.server == nil {
			close(ready)
		}
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()

	t.Cleanup(func() {
		if err := o.Stop(); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	})

	// Wait for server
	addr := o.WaitForServer(100 * time.Millisecond)

	// Build base URL
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	return "http://" + addr
}

// WaitForServer waits until the server is ready and returns the address
func (o *Okapi) WaitForServer(timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	if o == nil || o.server == nil {
		return ""
	}
	for time.Now().Before(deadline) {
		if o.server != nil && o.server.Addr != "" {
			return o.server.Addr
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ""
}
