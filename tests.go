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
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type TestServer struct {
	*Okapi
	BaseURL     string
	t           TestingT
	httptestSrv *httptest.Server
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
	o.applyCommon()
	o.context.okapi = o
	srv := httptest.NewServer(o)
	t.Cleanup(srv.Close)

	return &TestServer{
		Okapi:       o,
		BaseURL:     srv.URL,
		t:           t,
		httptestSrv: srv,
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

	if o == nil {
		t.Fatalf("Okapi instance is nil")
	}

	errCh := make(chan error, 1)

	// Start server
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
				t.Errorf("Server failed to start: %v", err)
			}
		}
	}()

	// Cleanup Stop
	t.Cleanup(func() {
		if err := o.Stop(); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	})

	// Wait for server startup or startup error
	addr, err := o.waitForServerOrError(2*time.Second, errCh)
	if err != nil {
		t.Fatalf("Server failed to start: %v", err)
	}
	if addr == "" {
		t.Fatalf("Server did not start within timeout")
	}

	// Build base URL
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	return "http://" + strings.TrimPrefix(addr, ":")
}

func (o *Okapi) waitForServerOrError(timeout time.Duration, errCh <-chan error) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case err := <-errCh:
			if err != nil {
				return "", err
			}
		default:
		}

		if o.server != nil && o.server.Addr != "" {
			conn, err := net.DialTimeout("tcp", o.server.Addr, 50*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				select {
				case startErr := <-errCh:
					if startErr != nil {
						return "", startErr
					}
				default:
				}
				return o.server.Addr, nil
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case err := <-errCh:
		return "", err
	default:
	}
	return "", nil
}

// WaitForServer waits until the server is ready and returns the address
func (o *Okapi) WaitForServer(timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if o.server != nil && o.server.Addr != "" {
			conn, err := net.DialTimeout("tcp", o.server.Addr, 50*time.Millisecond)
			if err == nil {
				err = conn.Close()
				if err != nil {
					return ""
				}
				return o.server.Addr
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return ""
}
