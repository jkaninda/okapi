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
	"github.com/stretchr/testify/assert"
	"reflect"
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
func TestGetFieldTypeName(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string type",
			input:    "",
			expected: "string",
		},
		{
			name:     "int type",
			input:    int(0),
			expected: "int",
		},
		{
			name:     "int8 type",
			input:    int8(0),
			expected: "int",
		},
		{
			name:     "int16 type",
			input:    int16(0),
			expected: "int",
		},
		{
			name:     "int32 type",
			input:    int32(0),
			expected: "int",
		},
		{
			name:     "int64 type",
			input:    int64(0),
			expected: "int",
		},
		{
			name:     "uint type",
			input:    uint(0),
			expected: "uint",
		},
		{
			name:     "uint8 type",
			input:    uint8(0),
			expected: "uint",
		},
		{
			name:     "uint16 type",
			input:    uint16(0),
			expected: "uint",
		},
		{
			name:     "uint32 type",
			input:    uint32(0),
			expected: "uint",
		},
		{
			name:     "uint64 type",
			input:    uint64(0),
			expected: "uint",
		},
		{
			name:     "float32 type",
			input:    float32(0),
			expected: "float",
		},
		{
			name:     "float64 type",
			input:    float64(0),
			expected: "float",
		},
		{
			name:     "bool type",
			input:    false,
			expected: "bool",
		},
		{
			name:     "pointer to string",
			input:    new(string),
			expected: "string",
		},
		{
			name:     "pointer to int",
			input:    new(int),
			expected: "int",
		},
		{
			name:     "slice type",
			input:    []string{},
			expected: "[]string",
		},
		{
			name:     "map type",
			input:    map[string]int{},
			expected: "map[string]int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := reflect.TypeOf(tt.input)
			result := getFieldTypeName(typ)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractParams(t *testing.T) {
	type SimpleStruct struct {
		ID   string `path:"id"`
		Name string `json:"name"`
	}

	type MultipleParams struct {
		UserID   string `path:"user_id"`
		TenantID int    `param:"tenant_id"`
		OrgID    int64  `path:"org_id"`
	}

	type MixedTags struct {
		PathParam string `path:"path_param"`
		ParamTag  int    `param:"param_tag"`
		JsonOnly  string `json:"json_only"`
		NoTag     string
		BothTags  string `path:"both_path" json:"both_json"`
	}

	type WithOptions struct {
		ID     string `path:"id,required"`
		Status string `param:"status,omitempty"`
	}

	type IgnoredTags struct {
		Ignored string `path:"-"`
		Valid   string `path:"valid"`
	}

	type EmptyTagValues struct {
		Empty string `path:""`
		Valid string `path:"valid"`
	}

	type VariousTypes struct {
		StringField  string  `path:"string_field"`
		IntField     int     `path:"int_field"`
		Int64Field   int64   `path:"int64_field"`
		UintField    uint    `path:"uint_field"`
		FloatField   float64 `path:"float_field"`
		BoolField    bool    `path:"bool_field"`
		PointerField *string `path:"pointer_field"`
	}

	tests := []struct {
		name          string
		input         any
		expectedCount int
		expectedNames []string
		expectedTypes []string
	}{
		{
			name:          "nil input",
			input:         nil,
			expectedCount: 0,
			expectedNames: nil,
			expectedTypes: nil,
		},
		{
			name:          "non-struct input",
			input:         "string",
			expectedCount: 0,
			expectedNames: nil,
			expectedTypes: nil,
		},
		{
			name:          "simple struct with path tag",
			input:         SimpleStruct{},
			expectedCount: 1,
			expectedNames: []string{"id"},
			expectedTypes: []string{"string"},
		},
		{
			name:          "pointer to struct",
			input:         &SimpleStruct{},
			expectedCount: 1,
			expectedNames: []string{"id"},
			expectedTypes: []string{"string"},
		},
		{
			name:          "multiple path and param tags",
			input:         MultipleParams{},
			expectedCount: 3,
			expectedNames: []string{"user_id", "tenant_id", "org_id"},
			expectedTypes: []string{"string", "int", "int"},
		},
		{
			name:          "mixed tags",
			input:         MixedTags{},
			expectedCount: 3,
			expectedNames: []string{"path_param", "param_tag", "both_path"},
			expectedTypes: []string{"string", "int", "string"},
		},
		{
			name:          "tags with options",
			input:         WithOptions{},
			expectedCount: 2,
			expectedNames: []string{"id", "status"},
			expectedTypes: []string{"string", "string"},
		},
		{
			name:          "ignored tag with dash",
			input:         IgnoredTags{},
			expectedCount: 1,
			expectedNames: []string{"valid"},
			expectedTypes: []string{"string"},
		},
		{
			name:          "empty tag values",
			input:         EmptyTagValues{},
			expectedCount: 1,
			expectedNames: []string{"valid"},
			expectedTypes: []string{"string"},
		},
		{
			name:          "various field types",
			input:         VariousTypes{},
			expectedCount: 7,
			expectedNames: []string{"string_field", "int_field", "int64_field", "uint_field", "float_field", "bool_field", "pointer_field"},
			expectedTypes: []string{"string", "int", "int", "uint", "float", "bool", "string"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := extractPathParamsFromStruct(tt.input)

			assert.Len(t, params, tt.expectedCount)

			if tt.expectedCount > 0 {
				for i, param := range params {
					assert.NotNil(t, param.Value)
					assert.Equal(t, tt.expectedNames[i], param.Value.Name)
					assert.Equal(t, "path", param.Value.In)
					assert.True(t, param.Value.Required)
				}
			}
		})
	}
}

func TestExtractPathParams_NilPointer(t *testing.T) {
	type TestStruct struct {
		ID string `path:"id"`
	}

	var nilPtr *TestStruct = nil
	params := extractPathParamsFromStruct(nilPtr)

	assert.Nil(t, params)
}

func TestExtractPathParams_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	params := extractPathParamsFromStruct(EmptyStruct{})

	assert.Empty(t, params)
}

func TestExtractPathParams_NestedStruct(t *testing.T) {
	type Inner struct {
		InnerID string `path:"inner_id"`
	}

	type Outer struct {
		OuterID string `path:"outer_id"`
		Inner   Inner
	}

	params := extractPathParamsFromStruct(Outer{})

	// Should only extract top-level fields
	assert.Len(t, params, 1)
	assert.Equal(t, "outer_id", params[0].Value.Name)
}

func TestParseTagName(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "simple name",
			tag:      "id",
			expected: "id",
		},
		{
			name:     "name with option",
			tag:      "id,required",
			expected: "id",
		},
		{
			name:     "name with multiple options",
			tag:      "id,required,omitempty",
			expected: "id",
		},
		{
			name:     "dash ignored",
			tag:      "-",
			expected: "",
		},
		{
			name:     "empty string",
			tag:      "",
			expected: "",
		},
		{
			name:     "only comma",
			tag:      ",required",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTagName(tt.tag)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPathParamName(t *testing.T) {
	type TestStruct struct {
		PathOnly    string `path:"path_name"`
		ParamOnly   string `param:"param_name"`
		BothTags    string `path:"path_first" param:"param_second"`
		JsonOnly    string `json:"json_name"`
		NoTags      string
		PathIgnored string `path:"-"`
	}

	typ := reflect.TypeOf(TestStruct{})

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"PathOnly", "path_name"},
		{"ParamOnly", "param_name"},
		{"BothTags", "path_first"},
		{"JsonOnly", ""},
		{"NoTags", ""},
		{"PathIgnored", ""},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			field, found := typ.FieldByName(tt.fieldName)
			assert.True(t, found)

			result := getPathParamName(field)
			assert.Equal(t, tt.expected, result)
		})
	}
}
