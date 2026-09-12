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

	"github.com/stretchr/testify/assert"
)

func TestAllowOrigin(t *testing.T) {
	origin := "http://localhost"
	origins := make([]string, 0, 4)
	origins = append(origins,
		"https://test/com",
		"https:example.com",
		"http://localhost",
	)

	result := originAllowed(origins, origin)
	assert.Equal(t, true, result)

	origins = append(origins, "*")
	result = originAllowed(origins, origin)
	assert.Equal(t, true, result)

	origins = append(origins, "*")
	result = originAllowed(origins, origin)
	assert.Equal(t, true, result)
}

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		name    string
		allowed []string
		origin  string
		want    bool
	}{
		{"empty origin is rejected", []string{"*"}, "", false},
		{"empty allow list is rejected", nil, "https://a.example", false},
		{"bare wildcard matches", []string{"*"}, "https://a.example", true},
		{"exact match", []string{"https://a.example"}, "https://a.example", true},
		{"case-insensitive exact match", []string{"HTTPS://A.Example"}, "https://a.example", true},
		{"mismatched exact", []string{"https://a.example"}, "https://b.example", false},
		{"subdomain wildcard matches", []string{"https://*.example.com"}, "https://app.example.com", true},
		{"subdomain wildcard case-insensitive", []string{"https://*.Example.com"}, "https://APP.example.com", true},
		{"subdomain wildcard rejects parent", []string{"https://*.example.com"}, "https://example.com", false},
		{"subdomain wildcard rejects wrong scheme", []string{"https://*.example.com"}, "http://app.example.com", false},
		{"subdomain wildcard rejects other domain", []string{"https://*.example.com"}, "https://app.attacker.com", false},
		{"wildcard does not cross slash", []string{"https://*.example.com"}, "https://a.example.com/evil.attacker.com", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, originAllowed(tc.allowed, tc.origin))
		})
	}
}

func TestAddVaryDeduplicates(t *testing.T) {
	h := http.Header{}
	addVary(h, "Origin")
	addVary(h, "origin") // case-insensitive dup
	addVary(h, "Access-Control-Request-Method")

	values := h.Values("Vary")
	assert.Equal(t, []string{"Origin", "Access-Control-Request-Method"}, values)
}

// invokeCORS runs the Cors middleware with a minimal Context and returns the
// recorder so tests can inspect the response headers and status.
func invokeCORS(cors Cors, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	c := &Context{
		request:  r,
		response: newResponseWriter(rec),
		okapi:    Default(),
	}
	_ = cors.CORSHandler(c)
	return rec
}

func TestCORSHandler_DisallowedOriginFallsThrough(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"https://good.example"}}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("Origin", "https://bad.example")

	rec := invokeCORS(cors, r)

	assert.Empty(t, rec.Header().Get(constAccessControlAllowOrigin))
	// Vary is set even when the origin is refused: the response still depends
	// on Origin, and a cache that does not know that can serve this
	// header-less response to an allowed origin.
	assert.Contains(t, rec.Header().Values("Vary"), "Origin")
}

func TestCORSHandler_EmptyOriginFallsThrough(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"*"}}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)

	rec := invokeCORS(cors, r)

	assert.Empty(t, rec.Header().Get(constAccessControlAllowOrigin))
}

func TestCORSHandler_SimpleRequestSetsOriginAndVary(t *testing.T) {
	cors := Cors{
		AllowedOrigins:   []string{"https://app.example"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Total-Count"},
		Headers:          map[string]string{"X-Frame-Options": "DENY"},
	}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("Origin", "https://app.example")

	rec := invokeCORS(cors, r)

	h := rec.Header()
	assert.Equal(t, "https://app.example", h.Get(constAccessControlAllowOrigin))
	assert.Equal(t, "true", h.Get(constAccessControlAllowCredentials))
	assert.Equal(t, "X-Total-Count", h.Get(constAccessControlExposeHeaders))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Contains(t, h.Values("Vary"), "Origin")

	// Preflight-only headers must not leak onto simple responses.
	assert.Empty(t, h.Get(constAccessControlAllowHeaders))
	assert.Empty(t, h.Get(constAccessControlAllowMethods))
	assert.Empty(t, h.Get(constAccessControlMaxAge))
}

func TestCORSHandler_PreflightReturns204WithHeaders(t *testing.T) {
	cors := Cors{
		AllowedOrigins: []string{"https://app.example"},
		MaxAge:         600,
	}
	r := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.Header.Set("Origin", "https://app.example")
	r.Header.Set("Access-Control-Request-Method", "POST")
	r.Header.Set("Access-Control-Request-Headers", "X-Custom, Authorization")

	rec := invokeCORS(cors, r)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	h := rec.Header()
	assert.Equal(t, "https://app.example", h.Get(constAccessControlAllowOrigin))
	// Reflected values when server-side config is empty.
	assert.Equal(t, "POST", h.Get(constAccessControlAllowMethods))
	assert.Equal(t, "X-Custom, Authorization", h.Get(constAccessControlAllowHeaders))
	assert.Equal(t, "600", h.Get(constAccessControlMaxAge))

	vary := h.Values("Vary")
	assert.Contains(t, vary, "Origin")
	assert.Contains(t, vary, "Access-Control-Request-Method")
	assert.Contains(t, vary, "Access-Control-Request-Headers")
}

func TestCORSHandler_PreflightUsesConfiguredLists(t *testing.T) {
	cors := Cors{
		AllowedOrigins: []string{"https://app.example"},
		AllowMethods:   []string{"GET", "POST"},
		AllowedHeaders: []string{"X-Api-Key"},
	}
	r := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.Header.Set("Origin", "https://app.example")
	r.Header.Set("Access-Control-Request-Method", "DELETE")
	r.Header.Set("Access-Control-Request-Headers", "X-Attacker")

	rec := invokeCORS(cors, r)

	h := rec.Header()
	// Configured lists win over reflected request headers.
	assert.Equal(t, "GET, POST", h.Get(constAccessControlAllowMethods))
	assert.Equal(t, "X-Api-Key", h.Get(constAccessControlAllowHeaders))
	// No reflected Vary entries when we didn't echo from the request.
	vary := h.Values("Vary")
	assert.NotContains(t, vary, "Access-Control-Request-Method")
	assert.NotContains(t, vary, "Access-Control-Request-Headers")
}

func TestCORSHandler_PlainOptionsIsNotPreflight(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"https://app.example"}}
	r := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.Header.Set("Origin", "https://app.example")
	// No Access-Control-Request-Method → not a preflight.

	rec := invokeCORS(cors, r)

	h := rec.Header()
	assert.Equal(t, "https://app.example", h.Get(constAccessControlAllowOrigin))
	assert.Empty(t, h.Get(constAccessControlAllowMethods),
		"plain OPTIONS should not emit preflight-only headers")
	assert.Empty(t, h.Get(constAccessControlAllowHeaders))
}

func TestCORSHandler_PreflightWithDisallowedOriginReturns204(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"https://good.example"}}
	r := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.Header.Set("Origin", "https://bad.example")
	r.Header.Set("Access-Control-Request-Method", "POST")

	rec := invokeCORS(cors, r)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	// No CORS headers are emitted — browser will reject.
	assert.Empty(t, rec.Header().Get(constAccessControlAllowOrigin))
}

func TestCORSHandler_WildcardSubdomainMatches(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"https://*.example.com"}}
	r := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.Header.Set("Origin", "https://tenant-1.example.com")

	rec := invokeCORS(cors, r)

	assert.Equal(t, "https://tenant-1.example.com",
		rec.Header().Get(constAccessControlAllowOrigin))
}

// TestCORSWildcardNeverGrantsCredentials guards the combination the same-origin
// policy exists to prevent: reflecting the caller's origin alongside
// Access-Control-Allow-Credentials lets any site read authenticated responses.
func TestCORSWildcardNeverGrantsCredentials(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"*"}, AllowCredentials: true}

	t.Run("simple request", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("Origin", "https://evil.example")

		h := invokeCORS(cors, r).Header()

		assert.Equal(t, "*", h.Get(constAccessControlAllowOrigin))
		assert.Empty(t, h.Get(constAccessControlAllowCredentials))
	})

	t.Run("preflight", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodOptions, "/x", nil)
		r.Header.Set("Origin", "https://evil.example")
		r.Header.Set("Access-Control-Request-Method", http.MethodGet)

		h := invokeCORS(cors, r).Header()

		assert.Equal(t, "*", h.Get(constAccessControlAllowOrigin))
		assert.Empty(t, h.Get(constAccessControlAllowCredentials))
	})

	t.Run("named origins still receive credentials", func(t *testing.T) {
		named := Cors{AllowedOrigins: []string{"https://app.example"}, AllowCredentials: true}
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("Origin", "https://app.example")

		h := invokeCORS(named, r).Header()

		assert.Equal(t, "https://app.example", h.Get(constAccessControlAllowOrigin))
		assert.Equal(t, "true", h.Get(constAccessControlAllowCredentials))
	})

	t.Run("pattern origins still receive credentials", func(t *testing.T) {
		pattern := Cors{AllowedOrigins: []string{"https://*.example.com"}, AllowCredentials: true}
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("Origin", "https://app.example.com")

		h := invokeCORS(pattern, r).Header()

		assert.Equal(t, "https://app.example.com", h.Get(constAccessControlAllowOrigin))
		assert.Equal(t, "true", h.Get(constAccessControlAllowCredentials))
	})
}

// TestCORSNullOriginNeverReflected covers the origin sent by sandboxed iframes,
// data: documents and some redirect chains.
func TestCORSNullOriginNeverReflected(t *testing.T) {
	for _, allowed := range [][]string{{"*"}, {"null"}, {"https://app.example"}} {
		cors := Cors{AllowedOrigins: allowed, AllowCredentials: true}
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("Origin", "null")

		h := invokeCORS(cors, r).Header()

		assert.NotEqual(t, "null", h.Get(constAccessControlAllowOrigin),
			"null origin reflected for AllowedOrigins %v", allowed)
		assert.Empty(t, h.Get(constAccessControlAllowCredentials))
	}
}

// TestWithCorsInstallsHandler covers the setup the documentation shows:
// okapi.New(okapi.WithCors(...)) alone must put CORS headers on actual
// responses, not only answer the preflight.
func TestWithCorsInstallsHandler(t *testing.T) {
	cors := Cors{AllowedOrigins: []string{"https://app.example"}, AllowCredentials: true}
	app := New(WithCors(cors))
	app.Get("/probe", func(c *Context) error { return c.String(http.StatusOK, "data") })

	r := httptest.NewRequest(http.MethodGet, "/probe", nil)
	r.Header.Set("Origin", "https://app.example")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://app.example", rec.Header().Get(constAccessControlAllowOrigin))
	assert.Equal(t, "true", rec.Header().Get(constAccessControlAllowCredentials))
	assert.Contains(t, rec.Header().Values("Vary"), "Origin")
}

// TestWithCorsInstallsHandlerOnce ensures repeated configuration replaces the
// settings rather than stacking handlers.
func TestWithCorsInstallsHandlerOnce(t *testing.T) {
	app := New(WithCors(Cors{AllowedOrigins: []string{"https://first.example"}}))
	before := len(app.middlewares)
	app.WithCORS(Cors{AllowedOrigins: []string{"https://second.example"}})

	assert.Equal(t, before, len(app.middlewares))

	app.Get("/probe", func(c *Context) error { return c.String(http.StatusOK, "data") })
	r := httptest.NewRequest(http.MethodGet, "/probe", nil)
	r.Header.Set("Origin", "https://second.example")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, r)

	assert.Equal(t, "https://second.example", rec.Header().Get(constAccessControlAllowOrigin))
}
