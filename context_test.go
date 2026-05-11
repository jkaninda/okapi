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
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jkaninda/okapi/okapitest"
)

// HelloHandler is referenced by helper_test.go (handleName name extraction).
func HelloHandler(c *Context) error {
	return c.Text(http.StatusOK, "Hello world!")
}

func TestNewTestContext(t *testing.T) {
	t.Parallel()

	ctx, rec := NewTestContext(http.MethodGet, "/test", nil)

	if got := ctx.Request().Method; got != http.MethodGet {
		t.Errorf("Method = %q, want %q", got, http.MethodGet)
	}
	if got := ctx.Request().URL.Path; got != "/test" {
		t.Errorf("URL.Path = %q, want %q", got, "/test")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("recorder.Code = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewContext_NilEngineFallsBackToDefault(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	ctx := NewContext(nil, rec, req)
	if ctx == nil {
		t.Fatal("NewContext returned nil")
	}
	if ctx.Request() != req {
		t.Error("request not wired through")
	}
	if ctx.Response() == nil {
		t.Error("response not wired through")
	}
}

// Store: Set / Get / typed accessors

func TestContext_Get_MissingKey(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)

	if v, ok := ctx.Get("missing"); ok || v != nil {
		t.Errorf("Get(missing) = (%v, %v), want (nil, false)", v, ok)
	}
	if got := ctx.GetString("missing"); got != "" {
		t.Errorf("GetString(missing) = %q, want \"\"", got)
	}
	if got := ctx.GetInt("missing"); got != 0 {
		t.Errorf("GetInt(missing) = %d, want 0", got)
	}
	if got := ctx.GetInt64("missing"); got != 0 {
		t.Errorf("GetInt64(missing) = %d, want 0", got)
	}
	if got := ctx.GetBool("missing"); got {
		t.Errorf("GetBool(missing) = true, want false")
	}
	if got, ok := ctx.GetTime("missing"); ok || !got.IsZero() {
		t.Errorf("GetTime(missing) = (%v, %v), want (zero, false)", got, ok)
	}
}

func TestContext_GetTypedAccessors(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name string
		key  string
		set  any
		want any                              // expected value of getter
		got  func(c *Context, key string) any // typed getter to invoke
	}{
		{"string direct", "s", "hello", "hello", func(c *Context, k string) any { return c.GetString(k) }},
		{"bool direct", "b", true, true, func(c *Context, k string) any { return c.GetBool(k) }},
		{"bool from \"true\"", "bs", "true", true, func(c *Context, k string) any { return c.GetBool(k) }},
		{"bool from non-true string", "bsf", "yes", false, func(c *Context, k string) any { return c.GetBool(k) }},
		{"int direct", "i", 42, 42, func(c *Context, k string) any { return c.GetInt(k) }},
		{"int from string", "is", "42", 42, func(c *Context, k string) any { return c.GetInt(k) }},
		{"int from float64", "if", float64(7), 7, func(c *Context, k string) any { return c.GetInt(k) }},
		{"int from non-numeric string", "ix", "abc", 0, func(c *Context, k string) any { return c.GetInt(k) }},
		{"int64 direct", "i64", int64(100), int64(100), func(c *Context, k string) any { return c.GetInt64(k) }},
		{"int64 from string", "i64s", "100", int64(100), func(c *Context, k string) any { return c.GetInt64(k) }},
		{"int64 from float64", "i64f", float64(100), int64(100), func(c *Context, k string) any { return c.GetInt64(k) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, _ := NewTestContext(http.MethodGet, "/", nil)
			ctx.Set(tt.key, tt.set)
			if got := tt.got(ctx, tt.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}

	t.Run("time direct", func(t *testing.T) {
		t.Parallel()
		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Set("t", now)
		got, ok := ctx.GetTime("t")
		if !ok {
			t.Fatal("GetTime ok = false")
		}
		if !got.Equal(now) {
			t.Errorf("GetTime = %v, want %v", got, now)
		}
	})

	t.Run("time wrong type returns zero", func(t *testing.T) {
		t.Parallel()
		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Set("t", "not-a-time")
		got, ok := ctx.GetTime("t")
		if ok || !got.IsZero() {
			t.Errorf("GetTime = (%v, %v), want (zero, false)", got, ok)
		}
	})
}

func TestContext_Set_IsConcurrencySafe(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			ctx.Set("k", i)
			_ = ctx.GetInt("k")
		}(i)
	}
	wg.Wait()

	if _, ok := ctx.Get("k"); !ok {
		t.Error("expected key to be set after concurrent writes")
	}
}

func TestContext_Copy(t *testing.T) {
	t.Parallel()

	src, _ := NewTestContext(http.MethodGet, "/", nil)
	src.Set("name", nameJane)
	src.Set("count", 3)

	cp := src.Copy()

	if got := cp.GetString("name"); got != nameJane {
		t.Errorf("copy.GetString(name) = %q, want %q", got, nameJane)
	}

	// Mutating the copy must not leak into the source.
	cp.Set("name", "John")
	if got := src.GetString("name"); got != nameJane {
		t.Errorf("source.GetString(name) = %q after copy mutation, want %q", got, nameJane)
	}
	if got := cp.GetString("name"); got != "John" {
		t.Errorf("copy.GetString(name) = %q, want %q", got, "John")
	}
}

// Request data accessors: query, headers, cookies, content-type

func TestContext_Query(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/?name=Jane&tags=a&tags=b&tags=c,d&empty=", nil)

	if got := ctx.Query("name"); got != "Jane" {
		t.Errorf("Query(name) = %q, want %q", got, "Jane")
	}
	if got := ctx.Query("missing"); got != "" {
		t.Errorf("Query(missing) = %q, want \"\"", got)
	}

	wantTags := []string{"a", "b", "c", "d"}
	if got := ctx.QueryArray("tags"); !reflect.DeepEqual(got, wantTags) {
		t.Errorf("QueryArray(tags) = %v, want %v", got, wantTags)
	}

	if got := ctx.QueryArray("missing"); got != nil {
		t.Errorf("QueryArray(missing) = %v, want nil", got)
	}

	gotMap := ctx.QueryMap()
	if gotMap["name"] != "Jane" {
		t.Errorf("QueryMap[name] = %q, want %q", gotMap["name"], "Jane")
	}
}

func TestContext_HeaderAndContentType(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)
	ctx.Request().Header.Set("X-Custom", "v1")
	ctx.Request().Header.Add("X-Multi", "a")
	ctx.Request().Header.Add("X-Multi", "b")
	ctx.Request().Header.Set("Content-Type", "application/json; charset=utf-8")
	ctx.Request().Header.Set("Accept", "application/json, text/plain")
	ctx.Request().Header.Set("Accept-Language", "en-US, fr ;q=0.8")

	if got := ctx.Header("X-Custom"); got != "v1" {
		t.Errorf("Header(X-Custom) = %q, want %q", got, "v1")
	}

	gotHeaders := ctx.Headers()
	if v := gotHeaders["X-Multi"]; !reflect.DeepEqual(v, []string{"a", "b"}) {
		t.Errorf("Headers[X-Multi] = %v, want [a b]", v)
	}

	if ct := ctx.ContentType(); ct != "application/json; charset=utf-8" {
		t.Errorf("ContentType() = %q", ct)
	}

	wantAccept := []string{"application/json", " text/plain"}
	if got := ctx.Accept(); !reflect.DeepEqual(got, wantAccept) {
		t.Errorf("Accept() = %v, want %v", got, wantAccept)
	}

	wantLangs := []string{"en-US", "fr ;q=0.8"}
	if got := ctx.AcceptLanguage(); !reflect.DeepEqual(got, wantLangs) {
		t.Errorf("AcceptLanguage() = %v, want %v", got, wantLangs)
	}
}

func TestContext_AcceptHeaders_Empty(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)

	if got := ctx.Accept(); got != nil {
		t.Errorf("Accept() = %v, want nil", got)
	}
	if got := ctx.AcceptLanguage(); got != nil {
		t.Errorf("AcceptLanguage() = %v, want nil", got)
	}
}

func TestContext_Cookie(t *testing.T) {
	t.Parallel()

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)
	ctx.Request().AddCookie(&http.Cookie{Name: "session", Value: "abc123"})

	val, err := ctx.Cookie("session")
	if err != nil {
		t.Fatalf("Cookie(session): %v", err)
	}
	if val != "abc123" {
		t.Errorf("Cookie(session) = %q, want %q", val, "abc123")
	}

	if _, err := ctx.Cookie("missing"); err == nil {
		t.Error("Cookie(missing): expected error, got nil")
	}
}

// IsWebSocketUpgrade

func TestContext_IsWebSocketUpgrade(t *testing.T) {
	t.Parallel()

	// IsWebSocketUpgrade requires Upgrade: websocket on a GET request.
	tests := []struct {
		name    string
		method  string
		upgrade string
		want    bool
	}{
		{"plain GET", http.MethodGet, "", false},
		{"GET with websocket upgrade", http.MethodGet, "websocket", true},
		{"GET with non-websocket upgrade", http.MethodGet, "h2c", false},
		{"POST with websocket upgrade rejected", http.MethodPost, "websocket", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, _ := NewTestContext(tt.method, "/", nil)
			if tt.upgrade != "" {
				ctx.Request().Header.Set("Upgrade", tt.upgrade)
			}
			if got := ctx.IsWebSocketUpgrade(); got != tt.want {
				t.Errorf("IsWebSocketUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContext_IsSSE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		accept string
		want   bool
	}{
		{"plain GET", http.MethodGet, "", false},
		{"GET event-stream", http.MethodGet, "text/event-stream", true},
		{"GET other accept", http.MethodGet, "application/json", false},
		{"POST event-stream rejected", http.MethodPost, "text/event-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, _ := NewTestContext(tt.method, "/", nil)
			if tt.accept != "" {
				ctx.Request().Header.Set("Accept", tt.accept)
			}
			if got := ctx.IsSSE(); got != tt.want {
				t.Errorf("IsSSE() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Response writers: JSON / Text / Data / NoContent

func TestContext_ResponseWriters(t *testing.T) {
	ts := NewTestServer(t)

	ts.Get("/json", func(c *Context) error {
		return c.JSON(http.StatusOK, map[string]string{"hello": "world"})
	})
	ts.Get("/created", func(c *Context) error {
		return c.Created(map[string]string{"hello": "world"})
	})
	ts.Get("/text", func(c *Context) error {
		return c.Text(http.StatusAccepted, "Hello world!")
	})
	ts.Get("/data", func(c *Context) error {
		return c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("raw bytes"))
	})
	ts.Get("/no-content", func(c *Context) error { return c.NoContent() })

	t.Run("JSON", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/json").
			ExpectStatusOK().
			ExpectBodyContains(`"hello":"world"`)
	})
	t.Run("Created", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/created").
			ExpectStatusCreated().
			ExpectBodyContains(`"hello":"world"`)
	})
	t.Run("Text", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/text").
			ExpectStatus(http.StatusAccepted).
			ExpectBodyContains("Hello world!")
	})
	t.Run("Data", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/data").
			ExpectStatusOK().
			ExpectBody("raw bytes")
	})
	t.Run("NoContent", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/no-content").
			ExpectStatus(http.StatusNoContent)
	})
}

// TestContext_Redirect exercises Redirect via a recorder so the default
// http.Client redirect-following doesn't swallow the 302.
func TestContext_Redirect(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/from", nil)
	rec := httptest.NewRecorder()
	ctx := NewContext(nil, rec, req)

	ctx.Redirect(http.StatusFound, "/elsewhere")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/elsewhere" {
		t.Errorf("Location = %q, want %q", loc, "/elsewhere")
	}
	if body := rec.Body.String(); body != "Redirecting to /elsewhere" {
		t.Errorf("body = %q, want %q", body, "Redirecting to /elsewhere")
	}
}

// File serving

func TestContext_ServeFileAttachment(t *testing.T) {
	createTemplate(t)

	ts := NewTestServer(t)
	ts.Get("/", func(c *Context) error {
		c.ServeFileAttachment("public/hello.html", "hello.html")
		return nil
	})

	resp, _ := okapitest.GET(t, ts.BaseURL+"/").
		ExpectStatusOK().
		Execute()

	if cd := resp.Header.Get("Content-Disposition"); cd == "" || cd[:len("attachment;")] != "attachment;" {
		t.Errorf("Content-Disposition = %q, want attachment; ...", cd)
	}
}

// Path / Param helpers via real routing

func TestContext_PathParam(t *testing.T) {
	ts := NewTestServer(t)
	ts.Get("/books/:id", func(c *Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"id":   c.Param("id"),
			"path": c.Path(),
		})
	})

	okapitest.GET(t, ts.BaseURL+"/books/42").
		ExpectStatusOK().
		ExpectBodyContains(`"id":"42"`).
		ExpectBodyContains(`"path":"/books/42"`)
}
