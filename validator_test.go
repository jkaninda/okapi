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

import "testing"

func TestBindStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name" form:"name" query:"name" header:"X-Name" required:"true" minLength:"1" maxLength:"100"`
		Age   int    `json:"age" form:"age" query:"age" header:"X-Age" required:"true" min:"1" max:"120"`
		Email string `json:"email" form:"email" query:"email" header:"X-Email" required:"true" format:"email" minLength:"1" maxLength:"100"`
	}

	tests := []struct {
		name       string
		input      map[string]string
		source     string
		expected   TestStruct
		shouldFail bool
	}{
		{
			name:   "Valid JSON Input",
			input:  map[string]string{"name": "John", "age": "30", "email": "john@example.com"},
			source: "json",
			expected: TestStruct{
				Name:  "John",
				Age:   30,
				Email: "john@example.com",
			},
			shouldFail: false,
		},
		{
			name:   "Missing Required Field",
			input:  map[string]string{"age": "30", "email": ""},
			source: "json",
			expected: TestStruct{
				Name:  "",
				Age:   30,
				Email: "",
			},
			shouldFail: true,
		},
	}

	for _, _ = range tests {

	}

}
