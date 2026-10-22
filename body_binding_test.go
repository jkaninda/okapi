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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func bindInput[T any](t *testing.T, contentType, body string) (T, error) {
	t.Helper()
	ctx, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(body))
	ctx.Request().Header.Set("Content-Type", contentType)
	var in T
	err := ctx.Bind(&in)
	return in, err
}

func TestBind_BodyWithOwnBodyField(t *testing.T) {
	type message struct {
		Subject string `json:"subject" required:"true"`
		Body    string `json:"body" required:"true"`
	}
	type input struct {
		ID   string `query:"id"`
		Body message
	}
	in, err := bindInput[input](t, constJSON, `{"subject":"Hi","body":"Hello"}`)
	if err != nil {
		t.Fatalf("Bind() error: %v", err)
	}
	if in.Body.Subject != "Hi" || in.Body.Body != "Hello" {
		t.Errorf("got %+v", in.Body)
	}
	if _, err := bindInput[input](t, constJSON, `{"subject":"Hi"}`); err == nil || !strings.Contains(err.Error(), "field Body.Body is required") {
		t.Errorf("missing inner body: err = %v", err)
	}
}

func TestBind_NonStructBodies(t *testing.T) {
	type sliceIn struct{ Body []User }
	s, err := bindInput[sliceIn](t, constJSON, `[{"name":"Jane"},{"name":"John"}]`)
	if err != nil || len(s.Body) != 2 || s.Body[1].Name != "John" {
		t.Errorf("slice: %+v %v", s.Body, err)
	}

	type mapIn struct{ Body map[string]int }
	m, err := bindInput[mapIn](t, constJSON, `{"a":1,"b":2}`)
	if err != nil || m.Body["b"] != 2 {
		t.Errorf("map: %+v %v", m.Body, err)
	}

	type stringIn struct{ Body string }
	str, err := bindInput[stringIn](t, constJSON, `"hello"`)
	if err != nil || str.Body != "hello" {
		t.Errorf("string: %q %v", str.Body, err)
	}

	y, err := bindInput[sliceIn](t, constYAML, "- name: Jane\n")
	if err != nil || len(y.Body) != 1 || y.Body[0].Name != nameJane {
		t.Errorf("yaml slice: %+v %v", y.Body, err)
	}

	if _, err := bindInput[sliceIn](t, constJSON, `{"name":"Jane"}`); err == nil {
		t.Error("object into a slice body: expected an error")
	}
	if _, err := bindInput[sliceIn](t, constFormURLEncoded, "name=Jane"); err == nil || !strings.Contains(err.Error(), "needs a struct body") {
		t.Errorf("form into a slice body: err = %v", err)
	}
	if e, err := bindInput[sliceIn](t, constJSON, ""); err != nil || e.Body != nil {
		t.Errorf("empty slice body: %+v %v", e.Body, err)
	}
}

func TestBind_StructBodies(t *testing.T) {
	type Base struct {
		Name string `json:"name" form:"name" required:"true"`
	}
	type payload struct {
		Base
		Age int `json:"age" form:"age"`
	}
	type input struct{ Body payload }

	in, err := bindInput[input](t, constJSON, `{"name":"Jane","age":30}`)
	if err != nil || in.Body.Name != nameJane || in.Body.Age != 30 {
		t.Errorf("embedded struct: %+v %v", in.Body, err)
	}
	f, err := bindInput[struct{ Body User }](t, constFormURLEncoded, "name=Jane")
	if err != nil || f.Body.Name != nameJane {
		t.Errorf("form into a struct: %+v %v", f.Body, err)
	}

	type ptrIn struct{ Body *payload }
	p, err := bindInput[ptrIn](t, constJSON, `{"name":"Jane"}`)
	if err != nil || p.Body == nil || p.Body.Name != nameJane {
		t.Errorf("pointer to struct: %+v %v", p.Body, err)
	}
}

func TestBind_ErrorsNameTheFieldPath(t *testing.T) {
	type line struct {
		SKU string `json:"sku" required:"true"`
	}
	type order struct {
		Lines []line `json:"lines"`
		Qty   int    `json:"qty" max:"5"`
	}
	type input struct {
		Page int `query:"page"`
		Body order
	}
	tests := []struct {
		name, query, body, want string
	}{
		{"nested slice element", "", `{"lines":[{"sku":"a"},{}]}`, "field Body.Lines[1].SKU is required"},
		{"body constraint", "", `{"qty":9}`, "field Body.Qty: value 9 must be <= 5"},
		{"decode error", "", `{"qty":"x"}`, "field Body: invalid JSON body"},
		{"query value", "?page=x", `{}`, "field Page: invalid integer value 'x'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := NewTestContext(http.MethodPost, "/test"+tt.query, strings.NewReader(tt.body))
			ctx.Request().Header.Set("Content-Type", constJSON)
			var in input
			err := ctx.Bind(&in)
			if err == nil || !strings.HasPrefix(err.Error(), tt.want) {
				t.Errorf("err = %v, want prefix %q", err, tt.want)
			}
		})
	}
}

func TestHandle_BindErrorNamesInputType(t *testing.T) {
	type CreateInput struct {
		Body struct {
			Name string `json:"name" required:"true"`
		}
	}
	var got error
	o := New(WithErrorHandler(func(c *Context, code int, msg string, err error) error {
		got = err
		return DefaultErrorHandler(c, code, msg, err)
	}))
	o.Post("/items", H(func(c *Context, in *CreateInput) error { return c.OK(in) }))

	req := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", constJSON)
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var be *BindError
	if !errors.As(got, &be) || !strings.HasSuffix(be.Input, ".CreateInput") {
		t.Fatalf("error handler got %#v, want a BindError naming CreateInput", got)
	}
	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Details != "field Body.Name is required" {
		t.Errorf("details = %q", resp.Details)
	}
}
