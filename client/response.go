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
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// Response wraps *http.Response with an already-read body and decode helpers.
// The original response body is closed by the time Response is returned, so
// callers should use Body or the typed decoders.
type Response struct {
	*http.Response

	// Body is the fully read response payload.
	Body []byte
	// Method is the HTTP method used by the originating request.
	Method string
	// URL is the final request URL.
	URL string
}

// String returns the body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// IsSuccess reports whether the status code is in [200, 300).
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// Error returns an *HTTPError describing the response when the status is not
// 2xx, and nil otherwise.
func (r *Response) Error() error {
	if r.IsSuccess() {
		return nil
	}
	return &HTTPError{
		StatusCode: r.StatusCode,
		Status:     r.Status,
		Method:     r.Method,
		URL:        r.URL,
		Body:       r.Body,
		Header:     r.Header,
	}
}

// JSON decodes the body into target as JSON.
func (r *Response) JSON(target any) error {
	return json.Unmarshal(r.Body, target)
}

// XML decodes the body into target as XML.
func (r *Response) XML(target any) error {
	return xml.Unmarshal(r.Body, target)
}

// YAML decodes the body into target as YAML.
func (r *Response) YAML(target any) error {
	return yaml.Unmarshal(r.Body, target)
}

// Decode unmarshals the body into target, choosing the format from the
// response Content-Type header: an "xml" content type decodes as XML, a
// "yaml" content type decodes as YAML, and anything else decodes as JSON.
// Use JSON, XML, or YAML directly when explicit control is needed.
func (r *Response) Decode(target any) error {
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "xml"):
		return r.XML(target)
	case strings.Contains(ct, "yaml"):
		return r.YAML(target)
	default:
		return r.JSON(target)
	}
}

// JSONPath returns the value at the dot-separated path within a JSON object
// body. It returns ok=false if the body is not a JSON object or the path does
// not resolve.
func (r *Response) JSONPath(path string) (any, bool) {
	var data map[string]any
	if err := json.Unmarshal(r.Body, &data); err != nil {
		return nil, false
	}
	var current any = data
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// Cookie returns the named cookie set by the server, or nil if not present.
func (r *Response) Cookie(name string) *http.Cookie {
	for _, c := range r.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}
