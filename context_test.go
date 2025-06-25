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

package okapi

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	rec := httptest.NewRecorder()

	resp := NewFakeResponse(rec)
	store := &Store{
		data: make(map[string]any),
	}
	ctx := &Context{
		okapi:    nil,
		request:  req,
		response: &resp,
		store:    store,
	}
	err := HelloHandler(*ctx)
	if err != nil {
		t.Errorf("Handler returned an error: %v", err)
	}

	result := rec.Result()
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			slog.Error(err.Error())
		}
	}(result.Body)

	if result.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

}
func NewFakeResponse(w http.ResponseWriter) response {
	return response{writer: w}
}

func NewFakeContext(method, target string) *Context {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	resp := &fakeResponse{ResponseWriter: rec}

	store := &Store{data: make(map[string]any)}

	return &Context{
		okapi:    nil,
		request:  req,
		response: resp,
		store:    store,
	}
}
func TestServeFile(t *testing.T) {
	createTemplate(t)

	o := New()
	o.Get("/", func(c Context) error {
		c.ServeFileAttachment("public", "hello.html")
		return nil
	})

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer o.Stop()

	waitForServer()
	assertStatus(t, "GET", "http://localhost:8080", nil, nil, "", http.StatusOK)
}

type fakeResponse struct {
	http.ResponseWriter
	status        int
	headerWritten bool
	bodyBytesSent int64
}

func (f *fakeResponse) Write(b []byte) (int, error) {
	if !f.headerWritten {
		f.WriteHeader(http.StatusOK)
	}
	n, err := f.ResponseWriter.Write(b)
	f.bodyBytesSent += int64(n)
	return n, err
}

func (f *fakeResponse) WriteHeader(statusCode int) {
	if f.headerWritten {
		return
	}
	f.status = statusCode
	f.headerWritten = true
	f.ResponseWriter.WriteHeader(statusCode)
}

func (f *fakeResponse) BodyBytesSent() int64 {
	return f.bodyBytesSent
}

func (f *fakeResponse) StatusCode() int {
	return f.status
}

func (f *fakeResponse) Close() {
}

func (f *fakeResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := f.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("not hijackable")
	}
	return h.Hijack()
}

func HelloHandler(c Context) error {
	if c.IsWebSocketUpgrade() {
		fmt.Println("WebSocket upgrade detected")
	}
	c.Set("hello", "Hello world!")
	c.Set("isAdmin", true)
	c.Set("id", 3)

	id := c.GetInt("id")
	if id != 3 {
		return errors.New("id is not 3")
	}
	isAdmin := c.GetBool("isAdmin")
	if !isAdmin {
		return errors.New("isAdmin is not true")
	}
	hello := c.GetString("hello")
	return c.Data(http.StatusOK, PLAINTEXT, []byte(hello))
}
