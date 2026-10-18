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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

// UpdateGoldenEnv is the environment variable that, set to 1, makes
// AssertOpenAPIGolden write the golden file instead of comparing with it.
const UpdateGoldenEnv = "OKAPI_UPDATE_GOLDEN"

// OpenAPIProvider builds an OpenAPI document. It is implemented by *okapi.Okapi.
type OpenAPIProvider interface {
	OpenAPISpec() *openapi3.T
}

// AssertOpenAPIGolden compares the OpenAPI 3.1 document of app with the golden
// file at path, and fails the test with the first difference. It keeps the
// documented API from drifting unnoticed: a change to it shows up in the
// golden file's diff.
//
// Run the tests with OKAPI_UPDATE_GOLDEN=1 to write the golden file:
//
//	func TestOpenAPI(t *testing.T) {
//	    okapitest.AssertOpenAPIGolden(t, newApp(), "testdata/openapi.json")
//	}
func AssertOpenAPIGolden(t *testing.T, app OpenAPIProvider, path string) {
	t.Helper()
	got, err := json.MarshalIndent(app.OpenAPISpec(), "", "  ")
	if err != nil {
		t.Fatalf("failed to encode the OpenAPI document: %v", err)
	}
	got = append(got, '\n')

	if os.Getenv(UpdateGoldenEnv) == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create the golden file directory: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("failed to write the golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		t.Fatalf("golden file %s does not exist: run the test with %s=1 to create it", path, UpdateGoldenEnv)
	}
	if err != nil {
		t.Fatalf("failed to read the golden file: %v", err)
	}
	if diff := firstDiff(string(want), string(got)); diff != "" {
		t.Errorf("OpenAPI document differs from %s (run with %s=1 to update it):\n%s", path, UpdateGoldenEnv, diff)
	}
}

// firstDiff describes the first line where want and got differ, with the
// lines around it, or returns "" when they are equal.
func firstDiff(want, got string) string {
	if want == got {
		return ""
	}
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	i := 0
	for i < len(wantLines) && i < len(gotLines) && wantLines[i] == gotLines[i] {
		i++
	}
	const context = 3
	var b strings.Builder
	for j := max(0, i-context); j < i; j++ {
		fmt.Fprintf(&b, "  %4d   %s\n", j+1, wantLines[j])
	}
	for j := i; j < min(len(wantLines), i+context); j++ {
		fmt.Fprintf(&b, "- %4d   %s\n", j+1, wantLines[j])
	}
	for j := i; j < min(len(gotLines), i+context); j++ {
		fmt.Fprintf(&b, "+ %4d   %s\n", j+1, gotLines[j])
	}
	return b.String()
}
