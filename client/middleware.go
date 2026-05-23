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

package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RoundTripFunc is the function form of an http.RoundTripper used to compose
// the client middleware chain.
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements http.RoundTripper.
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// Middleware wraps a RoundTripFunc to observe or modify requests/responses.
// Middlewares should call next exactly once unless they short-circuit on
// purpose (e.g. caching).
type Middleware func(next RoundTripFunc) RoundTripFunc

// chain composes the middlewares around base. The first middleware in the
// slice ends up as the outermost wrapper.
func chain(base RoundTripFunc, mw []Middleware) RoundTripFunc {
	if base == nil {
		base = http.DefaultTransport.RoundTrip
	}
	for i := len(mw) - 1; i >= 0; i-- {
		base = mw[i](base)
	}
	return base
}

// LoggingMiddleware logs one line per request to w with the method, URL,
// response status, and duration. It uses a fixed format suitable for
// development; production users typically supply their own.
func LoggingMiddleware(w io.Writer) Middleware {
	return func(next RoundTripFunc) RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := next(req)
			dur := time.Since(start)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s %s -> error %v (%s)\n", req.Method, req.URL.String(), err, dur)
				return resp, err
			}
			_, _ = fmt.Fprintf(w, "%s %s -> %d (%s)\n", req.Method, req.URL.String(), resp.StatusCode, dur)
			return resp, err
		}
	}
}

// UserAgentMiddleware sets the User-Agent header on every outgoing request,
// overwriting any value previously set.
func UserAgentMiddleware(ua string) Middleware {
	return func(next RoundTripFunc) RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			req.Header.Set("User-Agent", ua)
			return next(req)
		}
	}
}

// RequestIDMiddleware ensures an X-Request-Id header is set on every request,
// generating a random 16-byte hex value when the caller did not supply one.
func RequestIDMiddleware() Middleware {
	return func(next RoundTripFunc) RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Request-Id") == "" {
				req.Header.Set("X-Request-Id", newRequestID())
			}
			return next(req)
		}
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("rid-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
