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
	"errors"
	"fmt"
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	rec := httptest.NewRecorder()

	ctx := NewContext(nil, rec, req)
	err := HelloHandler(ctx)
	if err != nil {
		t.Errorf("Handler returned an error: %v", err)
	}
	okapitest.FromRecorder(t, rec).
		ExpectStatus(http.StatusOK).
		ExpectBody("Hello world!")

}

func TestServeFile(t *testing.T) {
	createTemplate(t)

	o := New()
	o.Get("/", func(c *Context) error {
		c.ServeFileAttachment("public", "hello.html")
		return nil
	})

	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8080").ExpectStatusOK()

}
func TestNewTestContext(t *testing.T) {
	ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
	if ctx.request.Method != http.MethodGet {
		t.Errorf("Expected method %s, got %s", http.MethodGet, ctx.request.Method)
	}
	if ctx.request.URL.Path != "/test" {
		t.Errorf("Expected URL path /test, got %s", ctx.request.URL.Path)
	}
	if rec.Code != 200 {
		t.Errorf("Expected initial response code 200, got %d", rec.Code)
	}
}
func HelloHandler(c *Context) error {
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
