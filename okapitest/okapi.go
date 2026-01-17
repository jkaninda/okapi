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
	"fmt"
	"io"
	"net/http"
	"testing"
)

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
