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
	input := "/users/:id"
	result := normalizeRoutePath(input)
	assert.Equal(t, "/users/{id}", result)

	input = "/*"
	result = normalizeRoutePath(input)
	assert.Equal(t, "/{any:.*}", result)

	input = "/*any"
	result = normalizeRoutePath(input)
	assert.Equal(t, "/{any:.*}", result)

	input = "/*any"
	result = normalizeRoutePath(input)
	assert.Equal(t, "/{any:.*}", result)

	input = "/*path"
	result = normalizeRoutePath(input)
	assert.Equal(t, "/{any:.*}", result)

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
