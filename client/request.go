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
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// RequestBuilder builds a single HTTP request. Most builder methods return
// the same builder for fluent chaining; Send is the terminal call that issues
// the request and returns the response.
type RequestBuilder struct {
	client      *Client
	ctx         context.Context
	method      string
	url         string
	query       url.Values
	headers     map[string]string
	body        io.Reader
	contentType string
	timeout     time.Duration
	retry       RetryPolicy
	extraMW     []Middleware
	buildErr    error
}

// WithContext attaches a context to the request. Defaults to context.Background.
func (rb *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	rb.ctx = ctx
	return rb
}

// Path appends a path segment to the request URL.
func (rb *RequestBuilder) Path(path string) *RequestBuilder {
	rb.url = strings.TrimRight(rb.url, "/") + "/" + strings.TrimLeft(path, "/")
	return rb
}

// Header sets a single request header, overwriting any existing value.
func (rb *RequestBuilder) Header(key, value string) *RequestBuilder {
	rb.headers[key] = value
	return rb
}

// Headers merges multiple headers into the request, overwriting on conflicts.
func (rb *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.headers[k] = v
	}
	return rb
}

// QueryParam appends a single query string parameter. Multiple calls with the
// same key produce multi-value query parameters.
func (rb *RequestBuilder) QueryParam(key, value string) *RequestBuilder {
	if rb.query == nil {
		rb.query = url.Values{}
	}
	rb.query.Add(key, value)
	return rb
}

// QueryParams adds multiple query parameters in one call.
func (rb *RequestBuilder) QueryParams(params map[string]string) *RequestBuilder {
	for k, v := range params {
		rb.QueryParam(k, v)
	}
	return rb
}

// BearerToken sets the Authorization header to "Bearer <token>".
func (rb *RequestBuilder) BearerToken(token string) *RequestBuilder {
	rb.headers["Authorization"] = "Bearer " + token
	return rb
}

// BasicAuth sets the Authorization header using HTTP Basic auth.
func (rb *RequestBuilder) BasicAuth(username, password string) *RequestBuilder {
	rb.headers["Authorization"] = "Basic " + basicAuth(username, password)
	return rb
}

// Timeout overrides the per-request timeout for this call.
func (rb *RequestBuilder) Timeout(d time.Duration) *RequestBuilder {
	rb.timeout = d
	return rb
}

// Retry overrides the retry policy for this call.
func (rb *RequestBuilder) Retry(p RetryPolicy) *RequestBuilder {
	rb.retry = p
	return rb
}

// Middleware appends middleware applied only to this request, after the
// client's middlewares.
func (rb *RequestBuilder) Middleware(mw ...Middleware) *RequestBuilder {
	rb.extraMW = append(rb.extraMW, mw...)
	return rb
}

// Body sets a raw body reader. The caller controls Content-Type via Header.
func (rb *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

// RawBody sets a raw byte slice as the request body.
func (rb *RequestBuilder) RawBody(b []byte) *RequestBuilder {
	rb.body = bytes.NewReader(b)
	return rb
}

// JSONBody marshals v as JSON and sets the Content-Type to application/json.
// If v is a string or []byte it is used verbatim.
func (rb *RequestBuilder) JSONBody(v any) *RequestBuilder {
	rb.contentType = "application/json"
	switch x := v.(type) {
	case string:
		rb.body = strings.NewReader(x)
	case []byte:
		rb.body = bytes.NewReader(x)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			rb.buildErr = fmt.Errorf("marshal json body: %w", err)
			return rb
		}
		rb.body = bytes.NewReader(b)
	}
	return rb
}

// XMLBody marshals v as XML and sets the Content-Type to application/xml.
func (rb *RequestBuilder) XMLBody(v any) *RequestBuilder {
	rb.contentType = "application/xml"
	b, err := xml.Marshal(v)
	if err != nil {
		rb.buildErr = fmt.Errorf("marshal xml body: %w", err)
		return rb
	}
	rb.body = bytes.NewReader(b)
	return rb
}

// YAMLBody marshals v as YAML and sets the Content-Type to application/yaml.
func (rb *RequestBuilder) YAMLBody(v any) *RequestBuilder {
	rb.contentType = "application/yaml"
	b, err := yaml.Marshal(v)
	if err != nil {
		rb.buildErr = fmt.Errorf("marshal yaml body: %w", err)
		return rb
	}
	rb.body = bytes.NewReader(b)
	return rb
}

// FormBody encodes values as application/x-www-form-urlencoded.
func (rb *RequestBuilder) FormBody(values map[string]string) *RequestBuilder {
	form := url.Values{}
	for k, v := range values {
		form.Set(k, v)
	}
	rb.contentType = "application/x-www-form-urlencoded"
	rb.body = strings.NewReader(form.Encode())
	return rb
}

// Multipart builds a multipart/form-data body using the supplied callback. The
// callback receives a *multipart.Writer that callers use to add fields and
// files. The writer is closed automatically once the callback returns.
func (rb *RequestBuilder) Multipart(build func(*multipart.Writer) error) *RequestBuilder {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := build(w); err != nil {
		rb.buildErr = fmt.Errorf("build multipart: %w", err)
		return rb
	}
	if err := w.Close(); err != nil {
		rb.buildErr = fmt.Errorf("close multipart: %w", err)
		return rb
	}
	rb.contentType = w.FormDataContentType()
	rb.body = &buf
	return rb
}

// Do issues the request and returns the response. The response body is fully
// read into memory before Do returns. Send is an alias for Do.
func (rb *RequestBuilder) Do() (*Response, error) {
	if rb.buildErr != nil {
		return nil, rb.buildErr
	}

	ctx := rb.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	finalURL := rb.url
	if len(rb.query) > 0 {
		sep := "?"
		if strings.Contains(finalURL, "?") {
			sep = "&"
		}
		finalURL = finalURL + sep + rb.query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, rb.method, finalURL, rb.body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}
	if rb.contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", rb.contentType)
	}

	rt := rb.client.roundTripper(rb.retry, rb.timeout, rb.extraMW)
	resp, err := rt(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return &Response{
		Response: resp,
		Body:     body,
		Method:   rb.method,
		URL:      finalURL,
	}, nil
}

// Send issues the request and returns the response. It is an alias for Do,
// provided for callers who prefer the verb.
func (rb *RequestBuilder) Send() (*Response, error) {
	return rb.Do()
}

// Decode is a shortcut for Do followed by Response.Decode into target, which
// selects the format from the response Content-Type. When the response is not
// 2xx it returns the *HTTPError without attempting to decode the body.
func (rb *RequestBuilder) Decode(target any) error {
	resp, err := rb.Do()
	if err != nil {
		return err
	}
	if err := resp.Error(); err != nil {
		return err
	}
	return resp.Decode(target)
}
