/*
 *  MIT License
 *
 * Copyright (c) 2024 Jonas Kaninda
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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

type fakeSpec struct{ title string }

func (f fakeSpec) OpenAPISpec() *openapi3.T {
	return &openapi3.T{OpenAPI: "3.1.0", Info: &openapi3.Info{Title: f.title, Version: "1"}, Paths: openapi3.NewPaths()}
}

func TestAssertOpenAPIGolden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testdata", "openapi.json")

	t.Setenv(UpdateGoldenEnv, "1")
	AssertOpenAPIGolden(t, fakeSpec{"Books"}, path)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("golden file not written: %v", err)
	}

	t.Setenv(UpdateGoldenEnv, "")
	AssertOpenAPIGolden(t, fakeSpec{"Books"}, path)

	// A changed document fails the comparison
	inner := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		AssertOpenAPIGolden(inner, fakeSpec{"Authors"}, path)
	}()
	<-done
	if !inner.Failed() {
		t.Error("a changed document should fail the comparison")
	}
}

func TestFirstDiff(t *testing.T) {
	if d := firstDiff("a\nb\n", "a\nb\n"); d != "" {
		t.Errorf("equal documents: got %q", d)
	}
	d := firstDiff("a\nb\nc\n", "a\nx\nc\n")
	if !strings.Contains(d, "-    2   b") || !strings.Contains(d, "+    2   x") {
		t.Errorf("unexpected diff:\n%s", d)
	}
}
