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
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecondToDuration(t *testing.T) {
	duration := secondsToDuration(30)
	slog.Info(duration.String())

}
func TestFPrintError(t *testing.T) {
	fPrintError("Error occurred ", "code", 400, "message", "Invalid input")
	slog.Info("Check the logs for formatted error message")
}
func TestFPrint(t *testing.T) {
	fPrint("Hello World")
	fPrint("Hello World", "key1", "value1", "key2", "value2")
}

func TestSanitizeHeaders(t *testing.T) {
	headers := http.Header{
		"Authorization":    []string{"Bearer token"},
		"Content-Type":     []string{"application/json"},
		"cookie":           []string{"cookie-value"},
		"X-Another-Header": []string{"AnotherValue"},
	}
	sanitizedHeaders := sanitizeHeaders(headers)
	slog.Info("Sanitized Headers:", "headers", sanitizedHeaders)
	if authorization, exists := sanitizedHeaders["Authorization"]; exists {
		if strings.Contains(authorization, "Bearer token") {
			t.Error("Authorization header should be sanitized")
		}

	} else {
		t.Error("Authorization header not found in sanitized headers")
	}

	if cookie, exists := sanitizedHeaders["cookie"]; exists {
		if strings.Contains(cookie, "cookie-value") {
			t.Error("Cookie header should be sanitized")
		}
	} else {
		t.Error("Cookie header not found in sanitized headers")
	}
}
func TestBuildDebugFields(t *testing.T) {
	o := New().WithDebug()
	o.Get("/debug", func(c *Context) error {
		fields := buildDebugFields(c)
		slog.Info("Debug Fields", "fields", fields)
		return c.JSON(http.StatusOK, fields)
	})

	req, _ := http.NewRequest(http.MethodGet, "/debug", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Length", "1234")
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "request_headers") {
		t.Error("Expected 'request_headers' in response body")
	}
}
