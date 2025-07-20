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
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func secondsToDuration(sec int) time.Duration {
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}
func fPrintError(msg string, args ...interface{}) {
	b := strings.Builder{}
	b.WriteString(msg)

	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = fmt.Sprintf("invalid_key_%d", i)
		}
		b.WriteString(fmt.Sprintf(" %s=%v", key, args[i+1]))
	}

	b.WriteByte('\n')
	_, _ = fmt.Fprint(os.Stderr, b.String())
}
func fPrint(msg string, args ...interface{}) {
	b := strings.Builder{}
	b.WriteString(msg)

	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			key = fmt.Sprintf("invalid_key_%d", i)
		}
		b.WriteString(fmt.Sprintf(" %s=%v", key, args[i+1]))
	}

	b.WriteByte('\n')
	_, _ = fmt.Fprint(os.Stdout, b.String())
}

func buildDebugFields(c Context) []any {
	fields := []any{
		"request_content_length", c.request.ContentLength,
	}

	if len(c.request.Header) > 0 {
		fields = append(fields, "request_headers", sanitizeHeaders(c.request.Header))
	}
	if len(c.request.URL.Query()) > 0 {
		fields = append(fields, "query_params", c.request.URL.Query())
	}
	if len(c.response.Header()) > 0 {
		fields = append(fields, "response_headers", sanitizeHeaders(c.response.Header()))
	}
	return fields

}

// sanitizeHeaders removes sensitive headers from logging
func sanitizeHeaders(headers http.Header) map[string][]string {
	sanitized := make(map[string][]string)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
		"x-api-key":     true,
		"x-auth-token":  true,
	}

	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if sensitiveHeaders[lowerKey] {
			sanitized[key] = []string{"[REDACTED]"}
		} else {
			sanitized[key] = values
		}
	}
	return sanitized
}
