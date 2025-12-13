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
	"github.com/getkin/kin-openapi/openapi3"
	"testing"
)

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []expectedParam
	}{
		{
			name: "brace style id",
			path: "/users/{id}",
			expected: []expectedParam{
				{name: "id", schemaType: "string", format: constUUID},
			},
		},
		{
			name: "colon style id",
			path: "/users/:id",
			expected: []expectedParam{
				{name: "id", schemaType: "string", format: constUUID},
			},
		},
		{
			name: "brace style typed int",
			path: "/books/{id:int}",
			expected: []expectedParam{
				{name: "id", schemaType: "integer"},
			},
		},
		{
			name: "colon style typed int",
			path: "/books/:id:int",
			expected: []expectedParam{
				{name: "id", schemaType: "integer"},
			},
		},
		{
			name: "uuid user id",
			path: "/users/{user_id:uuid}",
			expected: []expectedParam{
				{name: "user_id", schemaType: "string", format: constUUID},
			},
		},
		{
			name: "mixed styles",
			path: "/books/{id:int}/chapters/:chapter",
			expected: []expectedParam{
				{name: "id", schemaType: "integer"},
				{name: "chapter", schemaType: "string"},
			},
		},
		{
			name: "no duplicate when mixed",
			path: "/users/{id:int}/details/:id",
			expected: []expectedParam{
				{name: "id", schemaType: "integer"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := extractPathParams(tt.path)

			if len(params) != len(tt.expected) {
				t.Fatalf("expected %d params, got %d", len(tt.expected), len(params))
			}

			for i, exp := range tt.expected {
				param := params[i].Value
				if param == nil {
					t.Fatalf("param %d is nil", i)
				}

				if param.In != "path" {
					t.Errorf("param %q: expected In=path, got %q", param.Name, param.In)
				}

				if !param.Required {
					t.Errorf("param %q should be required", param.Name)
				}

				assertSchema(t, param.Schema, exp)
			}
		})
	}
}

type expectedParam struct {
	name       string
	schemaType string
	format     string
}

func assertSchema(t *testing.T, schema *openapi3.SchemaRef, exp expectedParam) {
	t.Helper()

	if schema == nil || schema.Value == nil {
		t.Fatalf("schema is nil")
	}
	if !schema.Value.Type.Is(exp.schemaType) {
		t.Errorf(
			"param %q: expected schema type %q, got %q",
			exp.name,
			exp.schemaType,
			schema.Value.Type,
		)
	}

	if exp.format != "" && schema.Value.Format != exp.format {
		t.Errorf(
			"param %q: expected format %q, got %q",
			exp.name,
			exp.format,
			schema.Value.Format,
		)
	}
}
