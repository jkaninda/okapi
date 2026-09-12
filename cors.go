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
	//
	// Credentials are never granted to the bare "*" wildcard: a request
	// matched by it receives the literal "*" and no credentials header.
	// Browsers reject that pairing precisely because it would let any site
	// read authenticated responses, and echoing the request's origin back
	// instead would sidestep the guard rather than honour it. Configurations
	// that need credentials must name their origins, exactly or by pattern.
	AllowCredentials bool
}

// CORSHandler applies CORS headers and short-circuits real preflight
// requests (OPTIONS with Access-Control-Request-Method) with 204.
// Plain OPTIONS requests fall through to the next handler.
func (cors Cors) CORSHandler(c *Context) error {
	isPreflight := c.request.Method == http.MethodOptions &&
		c.request.Header.Get("Access-Control-Request-Method") != ""

	addVary(c.response.Header(), "Origin")

	allowed, ok := cors.resolveOrigin(c.request.Header.Get("Origin"))
	if !ok {
		if isPreflight {
			c.response.WriteHeader(http.StatusNoContent)
			return nil
		}
		return c.Next()
	}

	cors.writeHeaders(c.response.Header(), c.request, allowed, isPreflight)

	if isPreflight {
		c.response.WriteHeader(http.StatusNoContent)
		return nil
	}
	return c.Next()
}

// allowedOrigin is the outcome of matching a request's Origin against the
// configuration: the value to send in Access-Control-Allow-Origin, and whether
// credentials may be granted alongside it.
type allowedOrigin struct {
	value       string
	credentials bool
}

// resolveOrigin matches origin against AllowedOrigins.
//
// The bare "*" wildcard resolves to the literal "*" and never carries
// credentials, so the unsafe wildcard-plus-credentials configuration degrades
// to the safe one instead of quietly working.
//
// The "null" origin never matches. It is what sandboxed iframes, data:
// documents and some redirect chains send, so reflecting it grants any of them
// the access the allow-list was meant to restrict.
func (cors Cors) resolveOrigin(origin string) (allowedOrigin, bool) {
	if origin == "" || strings.EqualFold(origin, "null") {
		return allowedOrigin{}, false
	}

	loweredOrigin := strings.ToLower(origin)
	for _, entry := range cors.AllowedOrigins {
		switch {
		case entry == "*":
			return allowedOrigin{value: "*"}, true
		case strings.EqualFold(entry, origin):
			return allowedOrigin{value: origin, credentials: cors.AllowCredentials}, true
		case strings.Contains(entry, "*") && matchWildcardOrigin(entry, loweredOrigin):
			return allowedOrigin{value: origin, credentials: cors.AllowCredentials}, true
		}
	}
	return allowedOrigin{}, false
}

func (cors Cors) writeHeaders(h http.Header, r *http.Request, allowed allowedOrigin, isPreflight bool) {
	h.Set(constAccessControlAllowOrigin, allowed.value)
	addVary(h, "Origin")

	if allowed.credentials {
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

// originAllowed reports whether origin matches the allow-list. It answers only
// the membership question; use Cors.resolveOrigin when the header values
// matter, since the wildcard and credentials interact.
func originAllowed(allowed []string, origin string) bool {
	_, ok := Cors{AllowedOrigins: allowed}.resolveOrigin(origin)
	return ok
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
