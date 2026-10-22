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
	"bytes"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInputProblems(t *testing.T) {
	type ok struct {
		ID     int           `path:"id"`
		Tags   []string      `query:"tags"`
		Limit  *int          `query:"limit"`
		Trace  string        `header:"X-Trace"`
		Wait   time.Duration `query:"wait"`
		Body   []map[string]any
		hidden time.Time
	}
	type bad struct {
		Org   string    `param:"org" required:"true"`
		Team  string    `param:"team"`
		Since time.Time `query:"since"`
		Rates []float64 `query:"rates"`
		Body  func()
	}
	tests := []struct {
		name  string
		input reflect.Type
		path  string
		want  []string
	}{
		{"valid", reflect.TypeFor[ok](), "/items/{id}", nil},
		{"wildcard param", reflect.TypeFor[struct {
			Rest string `path:"rest"`
		}](), "/files/{rest...}", nil},
		{"invalid", reflect.TypeFor[bad](), "/items", []string{
			`field Org: required path parameter "org" is not in the route`,
			"field Since: time.Time cannot be parsed from a query value",
			"field Rates: []float64 cannot be parsed from a query value",
			"field Body: func() cannot be decoded from a request body",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inputProblems(tt.input, tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("inputProblems() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegisterLogsUnbindableInput(t *testing.T) {
	type GetInput struct {
		ID string `path:"id" required:"true"`
	}
	var buf bytes.Buffer
	o := New(WithLogger(slog.New(slog.NewTextHandler(&buf, nil))))
	o.Get("/items/:id", H(func(c *Context, in *GetInput) error { return nil }))
	if buf.Len() != 0 {
		t.Fatalf("valid route logged: %s", buf.String())
	}
	o.Get("/things/{key}", H(func(c *Context, in *GetInput) error { return nil }))
	if out := buf.String(); !strings.Contains(out, "route input can never bind") || !strings.Contains(out, `required path parameter \"id\" is not in the route`) {
		t.Errorf("log = %s", out)
	}
}
