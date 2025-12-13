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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNormalizeRoutePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "colon param",
			input:    "/users/:id",
			expected: "/users/{id}",
		},
		{
			name:     "colon param with type",
			input:    "/users/:id:int",
			expected: "/users/{id}",
		},
		{
			name:     "colon param with type and trailing slash",
			input:    "/users/:id:int/",
			expected: "/users/{id}/",
		},
		{
			name:     "brace param with type",
			input:    "/users/{id:int}",
			expected: "/users/{id}",
		},
		{
			name:     "wildcard only",
			input:    "/*",
			expected: "/{any:.*}",
		},
		{
			name:     "named wildcard",
			input:    "/*any",
			expected: "/{any:.*}",
		},
		{
			name:     "custom wildcard name ignored",
			input:    "/*path",
			expected: "/{any:.*}",
		},
		{
			name:     "mixed params and wildcard",
			input:    "/users/:id/books/*",
			expected: "/users/{id}/books/{any:.*}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRoutePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllowOrigin(t *testing.T) {
	origin := "http://localhost"
	origins := []string{"https://test/com", "https:example.com", "http://localhost"}

	result := allowedOrigin(origins, origin)
	assert.Equal(t, true, result)

	origins = append(origins, "*")
	result = allowedOrigin(origins, origin)
	assert.Equal(t, true, result)
}

func TestValidateAddr(t *testing.T) {
	addr := "0.0.0.0:8080"
	if !ValidateAddr(addr) {
		t.Errorf("Invalid addr: %s", addr)
	}

}

func TestLoadJWKSFromFile(t *testing.T) {
	_, err := LoadJWKSFromFile("testdata/jwks.json")
	if err == nil {
		t.Errorf("LoadJWKSFromFile should have returned an error")
	}

}
