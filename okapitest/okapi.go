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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type RequestBuilder struct {
	t           *testing.T
	method      string
	url         string
	headers     map[string]string
	body        io.Reader
	contentType string
	timeout     time.Duration
	resp        *http.Response
	respBody    []byte
	executed    bool
}

func Request(t *testing.T) *RequestBuilder {
	t.Helper()
	return &RequestBuilder{
		t:       t,
		headers: make(map[string]string),
		timeout: 30 * time.Second,
	}
}

// FromRecorder creates a RequestBuilder from a httptest.ResponseRecorder
// This is useful for testing handlers directly without making HTTP requests
func FromRecorder(t *testing.T, recorder interface{ Result() *http.Response }) *RequestBuilder {
	t.Helper()
	result := recorder.Result()

	bodyBytes, err := io.ReadAll(result.Body)
	_ = result.Body.Close()
	if err != nil {
		t.Fatalf("failed to read recorder response body: %v", err)
	}

	return &RequestBuilder{
		t:        t,
		headers:  make(map[string]string),
		timeout:  30 * time.Second,
		resp:     result,
		respBody: bodyBytes,
		executed: true,
	}
}

// HTTP method constructors

func GET(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodGet).URL(url)
}

func POST(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodPost).URL(url)
}

func PUT(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodPut).URL(url)
}

func DELETE(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodDelete).URL(url)
}

func PATCH(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodPatch).URL(url)
}

func HEAD(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodHead).URL(url)
}

func OPTIONS(t *testing.T, url string) *RequestBuilder {
	return Request(t).Method(http.MethodOptions).URL(url)
}

// Builder methods

func (rb *RequestBuilder) Method(method string) *RequestBuilder {
	rb.method = method
	return rb
}

// URL sets the request URL
func (rb *RequestBuilder) URL(url string) *RequestBuilder {
	rb.url = url
	return rb
}

// Path appends a path segment to the existing URL
func (rb *RequestBuilder) Path(path string) *RequestBuilder {
	rb.url = strings.TrimRight(rb.url, "/") + "/" + strings.TrimLeft(path, "/")
	return rb
}

func (rb *RequestBuilder) Header(k, v string) *RequestBuilder {
	rb.headers[k] = v
	return rb
}

func (rb *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		rb.headers[k] = v
	}
	return rb
}
func (rb *RequestBuilder) SetBasicAuth(username, password string) *RequestBuilder {
	rb.Header("Authorization", "Basic "+basicAuth(username, password))
	return rb
}
func (rb *RequestBuilder) SetBearerAuth(token string) *RequestBuilder {
	rb.Header("Authorization", "Bearer "+token)
	return rb
}
func (rb *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	rb.body = body
	return rb
}

func (rb *RequestBuilder) JSONBody(v any) *RequestBuilder {
	rb.contentType = "application/json"
	switch x := v.(type) {
	case string:
		rb.body = strings.NewReader(x)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			rb.t.Fatalf("failed to marshal JSON body: %v", err)
		}
		rb.body = bytes.NewReader(b)
	}
	return rb
}

func (rb *RequestBuilder) FormBody(values map[string]string) *RequestBuilder {
	rb.contentType = "application/x-www-form-urlencoded"
	parts := make([]string, 0, len(values))

	for k, v := range values {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	rb.body = strings.NewReader(strings.Join(parts, "&"))
	return rb
}

func (rb *RequestBuilder) Timeout(timeout time.Duration) *RequestBuilder {
	rb.timeout = timeout
	return rb
}

// Execute the request
func (rb *RequestBuilder) do() (*http.Response, []byte) {
	rb.t.Helper()

	if rb.executed {
		return rb.resp, rb.respBody
	}

	req, err := http.NewRequest(rb.method, rb.url, rb.body)
	if err != nil {
		rb.t.Fatalf("failed to create request: %v", err)
	}

	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}
	if rb.contentType != "" {
		req.Header.Set("Content-Type", rb.contentType)
	}

	client := &http.Client{
		Timeout: rb.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		rb.t.Fatalf("failed to perform request: %v", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		rb.t.Fatalf("failed to read response body: %v", err)
	}

	rb.resp = resp
	rb.respBody = bodyBytes
	rb.executed = true

	return resp, bodyBytes
}

// Execute and return response for manual inspection
func (rb *RequestBuilder) Execute() (*http.Response, []byte) {
	rb.t.Helper()
	return rb.do()
}

// Status code assertions

func (rb *RequestBuilder) ExpectStatus(code int) *RequestBuilder {
	rb.t.Helper()
	resp, _ := rb.do()
	if resp.StatusCode != code {
		rb.t.Errorf("expected status %d, got %d\nResponse body: %s",
			code, resp.StatusCode, string(rb.respBody))
	}
	return rb
}

func (rb *RequestBuilder) ExpectStatusOK() *RequestBuilder {
	return rb.ExpectStatus(http.StatusOK)
}

func (rb *RequestBuilder) ExpectStatusCreated() *RequestBuilder {
	return rb.ExpectStatus(http.StatusCreated)
}

func (rb *RequestBuilder) ExpectStatusAccepted() *RequestBuilder {
	return rb.ExpectStatus(http.StatusAccepted)
}

func (rb *RequestBuilder) ExpectStatusNoContent() *RequestBuilder {
	return rb.ExpectStatus(http.StatusNoContent)
}

func (rb *RequestBuilder) ExpectStatusBadRequest() *RequestBuilder {
	return rb.ExpectStatus(http.StatusBadRequest)
}

func (rb *RequestBuilder) ExpectStatusUnauthorized() *RequestBuilder {
	return rb.ExpectStatus(http.StatusUnauthorized)
}

func (rb *RequestBuilder) ExpectStatusForbidden() *RequestBuilder {
	return rb.ExpectStatus(http.StatusForbidden)
}

func (rb *RequestBuilder) ExpectStatusNotFound() *RequestBuilder {
	return rb.ExpectStatus(http.StatusNotFound)
}

func (rb *RequestBuilder) ExpectStatusConflict() *RequestBuilder {
	return rb.ExpectStatus(http.StatusConflict)
}

func (rb *RequestBuilder) ExpectStatusInternalServerError() *RequestBuilder {
	return rb.ExpectStatus(http.StatusInternalServerError)
}

// Body assertions

func (rb *RequestBuilder) ExpectBody(expected string) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()
	if string(body) != expected {
		rb.t.Errorf("expected body %q, got %q", expected, string(body))
	}
	return rb
}

func (rb *RequestBuilder) ExpectBodyContains(substr string) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()
	if !strings.Contains(string(body), substr) {
		rb.t.Errorf("expected body to contain %q, got %q", substr, string(body))
	}
	return rb
}

func (rb *RequestBuilder) ExpectContains(substr string) *RequestBuilder {
	return rb.ExpectBodyContains(substr)
}

func (rb *RequestBuilder) ExpectBodyNotContains(substr string) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()
	if strings.Contains(string(body), substr) {
		rb.t.Errorf("expected body to not contain %q, got %q", substr, string(body))
	}
	return rb
}

func (rb *RequestBuilder) ExpectEmptyBody() *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()
	if len(body) != 0 {
		rb.t.Errorf("expected empty body, got %q", string(body))
	}
	return rb
}

// JSON assertions

func (rb *RequestBuilder) ExpectJSON(expected any) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()

	var actual any
	if err := json.Unmarshal(body, &actual); err != nil {
		rb.t.Fatalf("response is not valid JSON: %v\nbody=%s", err, string(body))
	}

	expBytes, _ := json.Marshal(expected)
	actBytes, _ := json.Marshal(actual)

	if !bytes.Equal(expBytes, actBytes) {
		rb.t.Errorf("expected JSON:\n%s\ngot:\n%s", expBytes, actBytes)
	}
	return rb
}

func (rb *RequestBuilder) ExpectJSONPath(path string, expected any) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		rb.t.Fatalf("response is not valid JSON: %v", err)
	}

	actual := extractJSONPath(data, path)
	if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
		rb.t.Errorf("expected JSON path %q to be %v, got %v", path, expected, actual)
	}
	return rb
}

func (rb *RequestBuilder) ParseJSON(target any) *RequestBuilder {
	rb.t.Helper()
	_, body := rb.do()

	if err := json.Unmarshal(body, target); err != nil {
		rb.t.Fatalf("failed to parse JSON response: %v\nbody=%s", err, string(body))
	}
	return rb
}

// Header assertions

func (rb *RequestBuilder) ExpectHeader(key, value string) *RequestBuilder {
	rb.t.Helper()
	resp, _ := rb.do()
	actual := resp.Header.Get(key)
	if actual != value {
		rb.t.Errorf("expected header %q to be %q, got %q", key, value, actual)
	}
	return rb
}

func (rb *RequestBuilder) ExpectHeaderContains(key, substr string) *RequestBuilder {
	rb.t.Helper()
	resp, _ := rb.do()
	actual := resp.Header.Get(key)
	if !strings.Contains(actual, substr) {
		rb.t.Errorf("expected header %q to contain %q, got %q", key, substr, actual)
	}
	return rb
}

func (rb *RequestBuilder) ExpectHeaderExists(key string) *RequestBuilder {
	rb.t.Helper()
	resp, _ := rb.do()
	if resp.Header.Get(key) == "" {
		rb.t.Errorf("expected header %q to exist", key)
	}
	return rb
}

func (rb *RequestBuilder) ExpectContentType(contentType string) *RequestBuilder {
	return rb.ExpectHeader("Content-Type", contentType)
}

// Helper functions
func extractJSONPath(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	var current any = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			current = v[part]
		default:
			return nil
		}
	}
	return current
}

// AssertHTTPStatus asserts that an HTTP request returns the expected status code.
// Deprecated: Use RequestBuilder instead.
func AssertHTTPStatus(
	t *testing.T,
	method, url string,
	headers map[string]string,
	body io.Reader,
	contentType string,
	expected int,
) {
	t.Helper()

	resp, _, err := doRequest(method, url, headers, contentType, body)
	if err != nil {
		t.Fatalf("HTTP %s %s failed: %v", method, url, err)
	}

	if resp.StatusCode != expected {
		t.Errorf("Expected status %d for %s %s, got %d",
			expected, method, url, resp.StatusCode)
	}
}

// AssertHTTPResponse asserts that an HTTP request returns the expected status code and body.
// // Deprecated: Use RequestBuilder instead.
func AssertHTTPResponse(
	t *testing.T,
	method, url string,
	headers map[string]string,
	body io.Reader,
	contentType string,
	expectedStatus int,
	expectedBody string,
) {
	t.Helper()

	resp, data, err := doRequest(method, url, headers, contentType, body)
	if err != nil {
		t.Fatalf("HTTP %s %s failed: %v", method, url, err)
	}

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d for %s %s, got %d",
			expectedStatus, method, url, resp.StatusCode)
	}

	if expectedBody != "" && string(data) != expectedBody {
		t.Errorf("Expected response body:\n%s\nGot:\n%s", expectedBody, string(data))
	}
}

func doRequest(method, url string, headers map[string]string, contentType string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("performing request: %w", err)
	}

	data, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if err != nil {
		return resp, nil, fmt.Errorf("reading body: %w", err)
	}

	return resp, data, nil
}
