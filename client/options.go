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
	"net/http"
	"time"
)

// Option configures a Client. Options are applied in order, so later options
// can override earlier ones.
type Option func(*Client)

// WithHTTPClient sets the underlying *http.Client used to dispatch requests.
// The client's Transport is wrapped with the configured middleware chain when
// the Client is used.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithTimeout sets the default per-request timeout. It is applied to every
// request unless the request builder overrides it via RequestBuilder.Timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithHeader sets a default header sent on every request.
func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithHeaders sets multiple default headers in one call.
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		for k, v := range headers {
			c.headers[k] = v
		}
	}
}

// WithBearerToken sets the default Authorization header to "Bearer <token>".
func WithBearerToken(token string) Option {
	return func(c *Client) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// WithBasicAuth sets the default Authorization header using HTTP Basic auth.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		c.headers["Authorization"] = "Basic " + basicAuth(username, password)
	}
}

// WithUserAgent sets the default User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.headers["User-Agent"] = ua
	}
}

// WithMiddleware appends middleware to the client's chain. Middleware runs in
// registration order, with the first registered being the outermost wrapper.
func WithMiddleware(mw ...Middleware) Option {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, mw...)
	}
}

// WithRetry sets the default retry policy for every request.
func WithRetry(p RetryPolicy) Option {
	return func(c *Client) {
		c.retry = p
	}
}
