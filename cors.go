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

type Cors struct {
	// AllowedOrigins specifies which origins are allowed.
	AllowedOrigins []string

	// AllowedHeaders defines which request headers are permitted.
	AllowedHeaders []string

	// ExposeHeaders indicates which response headers are exposed to the client.
	ExposeHeaders []string
	//
	Headers map[string]string

	// MaxAge defines how long the results of a preflight request can be cached (in seconds).
	MaxAge int

	// AllowMethods lists the HTTP methods permitted for cross-origin requests.
	AllowMethods     []string
	AllowCredentials bool
}

// CORSHandler applies CORS headers and handles preflight (OPTIONS) requests.
func (cors Cors) CORSHandler(next HandleFunc) HandleFunc {
	return func(c Context) error {
		origin := c.request.Header.Get("Origin")
		if !allowedOrigin(cors.AllowedOrigins, origin) {
			return next(c)
		}

		h := c.response.Header()

		// Always set origin
		h.Set(constAccessControlAllowOrigin, origin)

		// Allow credentials
		if cors.AllowCredentials {
			h.Set(constAccessControlAllowCredentials, "true")
		}

		// Allow headers
		if len(cors.AllowedHeaders) > 0 {
			h.Set(constAccessControlAllowHeaders, strings.Join(cors.AllowedHeaders, ", "))
		} else if reqHeaders := c.request.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			h.Set(constAccessControlAllowHeaders, reqHeaders)
		}

		// Allow methods
		if len(cors.AllowMethods) > 0 {
			h.Set(constAccessControlAllowMethods, strings.Join(cors.AllowMethods, ", "))
		} else if reqMethod := c.request.Header.Get("Access-Control-Request-Method"); reqMethod != "" {
			h.Set(constAccessControlAllowMethods, reqMethod)
		}

		// Expose headers
		if len(cors.ExposeHeaders) > 0 {
			h.Set(constAccessControlExposeHeaders, strings.Join(cors.ExposeHeaders, ", "))
		}

		// Max age
		if cors.MaxAge > 0 {
			h.Set(constAccessControlMaxAge, strconv.Itoa(cors.MaxAge))
		}

		// Preflight response
		if c.request.Method == http.MethodOptions {
			c.response.WriteHeader(http.StatusNoContent)
			return nil
		}

		return next(c)
	}
}
