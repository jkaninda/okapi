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
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is a reusable HTTP client bound to a base URL with default headers,
// middleware, and retry policy. It is safe for concurrent use.
type Client struct {
	baseURL     string
	headers     map[string]string
	http        *http.Client
	timeout     time.Duration
	middlewares []Middleware
	retry       RetryPolicy
}

// New returns a Client rooted at baseURL with the supplied options applied.
// The baseURL may be empty when callers always pass absolute URLs.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: make(map[string]string),
		http:    &http.Client{},
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// BaseURL returns the client's base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// Get starts a GET request builder against baseURL+path.
func (c *Client) Get(path string) *RequestBuilder {
	return c.newRequest(http.MethodGet, path)
}

// Post starts a POST request builder.
func (c *Client) Post(path string) *RequestBuilder {
	return c.newRequest(http.MethodPost, path)
}

// Put starts a PUT request builder.
func (c *Client) Put(path string) *RequestBuilder {
	return c.newRequest(http.MethodPut, path)
}

// Patch starts a PATCH request builder.
func (c *Client) Patch(path string) *RequestBuilder {
	return c.newRequest(http.MethodPatch, path)
}

// Delete starts a DELETE request builder.
func (c *Client) Delete(path string) *RequestBuilder {
	return c.newRequest(http.MethodDelete, path)
}

// Head starts a HEAD request builder.
func (c *Client) Head(path string) *RequestBuilder {
	return c.newRequest(http.MethodHead, path)
}

// Options starts an OPTIONS request builder.
func (c *Client) Options(path string) *RequestBuilder {
	return c.newRequest(http.MethodOptions, path)
}

// Request starts a builder for an arbitrary HTTP method.
func (c *Client) Request(method, path string) *RequestBuilder {
	return c.newRequest(method, path)
}

// Do dispatches a fully prepared *http.Request through the client middleware
// chain and returns the read Response. Provided for callers that need full
// control over request construction.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rt := c.roundTripper(c.retry, c.timeout, nil)
	resp, err := rt(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return &Response{
		Response: resp,
		Body:     body,
		Method:   req.Method,
		URL:      req.URL.String(),
	}, nil
}

// newRequest constructs a builder pre-populated with the client's defaults.
func (c *Client) newRequest(method, path string) *RequestBuilder {
	rb := &RequestBuilder{
		client:  c,
		method:  method,
		url:     joinURL(c.baseURL, path),
		headers: make(map[string]string, len(c.headers)),
		timeout: c.timeout,
		retry:   c.retry,
	}
	for k, v := range c.headers {
		rb.headers[k] = v
	}
	return rb
}

// roundTripper returns the chained RoundTripFunc for this client and the
// supplied per-request middleware/policy. The composition order, from
// outermost to innermost, is: client middlewares, per-request middlewares,
// retry middleware (when enabled), base transport. A non-zero perReqTimeout
// overrides the underlying http.Client's timeout for the duration of the call.
func (c *Client) roundTripper(p RetryPolicy, perReqTimeout time.Duration, extra []Middleware) RoundTripFunc {
	httpClient := c.http
	if perReqTimeout > 0 && perReqTimeout != httpClient.Timeout {
		cp := *httpClient
		cp.Timeout = perReqTimeout
		httpClient = &cp
	}
	base := RoundTripFunc(httpClient.Do)
	mw := make([]Middleware, 0, len(c.middlewares)+len(extra)+1)
	mw = append(mw, c.middlewares...)
	mw = append(mw, extra...)
	if p.enabled() {
		mw = append(mw, retryMiddleware(p))
	}
	return chain(base, mw)
}

func joinURL(base, path string) string {
	if base == "" {
		return path
	}
	if path == "" {
		return base
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return base + path
	}
	return base + "/" + path
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
