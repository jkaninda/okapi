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
	"strconv"
	"strings"
)

// Cors configures Cross-Origin Resource Sharing behavior.
type Cors struct {
	// AllowedOrigins is the set of origins permitted to access the resource.
	// Supports exact matches ("https://app.example.com"), a single wildcard
	// ("*" — allows any origin), or scheme+subdomain patterns
	// ("https://*.example.com").
	AllowedOrigins []string

	// AllowedHeaders lists request headers permitted on cross-origin requests.
	// When empty, the value of Access-Control-Request-Headers is echoed back
	// on preflight responses.
	AllowedHeaders []string

	// ExposeHeaders lists response headers the browser is allowed to expose
	// to client-side scripts (Access-Control-Expose-Headers).
	ExposeHeaders []string

	// Headers contains additional response headers to set on every
	// CORS-matched response (e.g. "X-Frame-Options": "DENY").
	Headers map[string]string

	// MaxAge is how long (in seconds) the browser may cache a preflight
	// response. Values ≤ 0 are omitted.
	MaxAge int

	// AllowMethods lists HTTP methods permitted for cross-origin requests.
	// When empty, the value of Access-Control-Request-Method is echoed back
	// on preflight responses.
	AllowMethods []string

	// AllowCredentials enables Access-Control-Allow-Credentials: true.
	// When set, AllowedOrigins should not rely on the bare "*" wildcard —
	// the origin is always echoed verbatim so credentialed requests work.
	AllowCredentials bool
}

// CORSHandler applies CORS headers and short-circuits real preflight
// requests (OPTIONS with Access-Control-Request-Method) with 204.
// Plain OPTIONS requests fall through to the next handler.
func (cors Cors) CORSHandler(c *Context) error {
	origin := c.request.Header.Get("Origin")
	isPreflight := c.request.Method == http.MethodOptions &&
		c.request.Header.Get("Access-Control-Request-Method") != ""

	if origin == "" || !originAllowed(cors.AllowedOrigins, origin) {
		if isPreflight {
			c.response.WriteHeader(http.StatusNoContent)
			return nil
		}
		return c.Next()
	}

	cors.writeHeaders(c.response.Header(), c.request, isPreflight)

	if isPreflight {
		c.response.WriteHeader(http.StatusNoContent)
		return nil
	}
	return c.Next()
}

func (cors Cors) writeHeaders(h http.Header, r *http.Request, isPreflight bool) {
	origin := r.Header.Get("Origin")

	h.Set(constAccessControlAllowOrigin, origin)
	addVary(h, "Origin")

	if cors.AllowCredentials {
		h.Set(constAccessControlAllowCredentials, "true")
	}

	if len(cors.ExposeHeaders) > 0 {
		h.Set(constAccessControlExposeHeaders, strings.Join(cors.ExposeHeaders, ", "))
	}

	if isPreflight {
		if len(cors.AllowedHeaders) > 0 {
			h.Set(constAccessControlAllowHeaders, strings.Join(cors.AllowedHeaders, ", "))
		} else if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			h.Set(constAccessControlAllowHeaders, reqHeaders)
			addVary(h, "Access-Control-Request-Headers")
		}

		if len(cors.AllowMethods) > 0 {
			h.Set(constAccessControlAllowMethods, strings.Join(cors.AllowMethods, ", "))
		} else if reqMethod := r.Header.Get("Access-Control-Request-Method"); reqMethod != "" {
			h.Set(constAccessControlAllowMethods, reqMethod)
			addVary(h, "Access-Control-Request-Method")
		}

		if cors.MaxAge > 0 {
			h.Set(constAccessControlMaxAge, strconv.Itoa(cors.MaxAge))
		}
	}

	for k, v := range cors.Headers {
		h.Set(k, v)
	}
}

func originAllowed(allowed []string, origin string) bool {
	if origin == "" {
		return false
	}
	loweredOrigin := strings.ToLower(origin)
	for _, entry := range allowed {
		if entry == "*" {
			return true
		}
		if strings.EqualFold(entry, origin) {
			return true
		}
		if strings.Contains(entry, "*") && matchWildcardOrigin(entry, loweredOrigin) {
			return true
		}
	}
	return false
}

func matchWildcardOrigin(pattern, origin string) bool {
	pattern = strings.ToLower(pattern)
	star := strings.Index(pattern, "*")
	if star < 0 {
		return false
	}
	prefix := pattern[:star]
	suffix := pattern[star+1:]
	if !strings.HasPrefix(origin, prefix) || !strings.HasSuffix(origin, suffix) {
		return false
	}
	middle := origin[len(prefix) : len(origin)-len(suffix)]
	return middle != "" && !strings.Contains(middle, "/")
}

// addVary appends value to the Vary header if not already present.
func addVary(h http.Header, value string) {
	existing := h.Values("Vary")
	for _, v := range existing {
		if strings.EqualFold(v, value) {
			return
		}
	}
	h.Add("Vary", value)
}
