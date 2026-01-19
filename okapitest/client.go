/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
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

package okapitest

import (
	"testing"
)

type TestClient struct {
	// BaseURL is the base URL for the test client.
	BaseURL string
	// Headers are the default headers to include in each request.
	Headers map[string]string
	t       *testing.T
}

// NewClient creates a new TestClient with the specified base URL.
func NewClient(t *testing.T, baseURL string) *TestClient {
	return &TestClient{BaseURL: baseURL, t: t, Headers: make(map[string]string)}
}

func (tc *TestClient) POST(path string) *RequestBuilder {
	return POST(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) GET(path string) *RequestBuilder {
	return GET(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) PUT(path string) *RequestBuilder {
	return PUT(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) DELETE(path string) *RequestBuilder {
	return DELETE(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) PATCH(path string) *RequestBuilder {
	return PATCH(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) HEAD(path string) *RequestBuilder {
	return HEAD(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
func (tc *TestClient) OPTIONS(path string) *RequestBuilder {
	return OPTIONS(tc.t, tc.BaseURL+path).Headers(tc.Headers)
}
