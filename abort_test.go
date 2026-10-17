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
	"net/http"
	"net/http/httptest"
	"testing"
)

// A middleware that aborts and then calls Next anyway must not reach the handler it refused.
func TestNextAfterAbortSkipsTheHandler(t *testing.T) {
	app := New(WithAccessLogDisabled())
	reached := false
	guard := func(c *Context) error {
		_ = c.AbortForbidden("denied")
		return c.Next()
	}
	app.Get("/x", func(c *Context) error {
		reached = true
		return c.OK("secret")
	}, UseMiddleware(guard))

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if reached {
		t.Fatal("the handler ran after the middleware aborted")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// Every Abort helper marks the request, and keeps returning the write result.
func TestAbortMarksTheRequest(t *testing.T) {
	aborts := map[string]func(c *Context) error{
		"AbortForbidden":  func(c *Context) error { return c.AbortForbidden("denied") },
		"AbortWithError":  func(c *Context) error { return c.AbortWithError(http.StatusConflict, nil) },
		"AbortWithJSON":   func(c *Context) error { return c.AbortWithJSON(http.StatusTeapot, map[string]string{}) },
		"AbortWithStatus": func(c *Context) error { return c.AbortWithStatus(http.StatusGone, "gone") },
		"AbortValidationErrors": func(c *Context) error {
			return c.AbortValidationErrors([]ValidationError{{Field: "name", Message: "required"}})
		},
		"AbortWithProblemDetail": func(c *Context) error {
			return c.AbortWithProblemDetail(NewProblemDetail(http.StatusBadRequest, "about:blank", "bad"))
		},
		"AbortNotModified": func(c *Context) error { return c.AbortNotModified() },
		"Abort":            func(c *Context) error { return c.Abort(nil) },
	}
	for name, abort := range aborts {
		ctx, _ := NewTestContext(http.MethodGet, "/x", nil)
		if ctx.IsAborted() {
			t.Fatalf("%s: a fresh context reports aborted", name)
		}
		if err := abort(ctx); err != nil {
			t.Errorf("%s = %v, want nil", name, err)
		}
		if !ctx.IsAborted() {
			t.Errorf("%s: IsAborted = false after the abort", name)
		}
		if err := ctx.Next(); err != nil {
			t.Errorf("%s: Next after the abort = %v, want nil", name, err)
		}
	}
}

// Writing a response without aborting leaves the chain running.
func TestPlainResponseDoesNotAbort(t *testing.T) {
	ctx, _ := NewTestContext(http.MethodGet, "/x", nil)
	_ = ctx.ErrorForbidden("denied")
	if ctx.IsAborted() {
		t.Fatal("ErrorForbidden marked the request aborted")
	}
}
