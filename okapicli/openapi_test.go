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

package okapicli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jkaninda/okapi"
	"gopkg.in/yaml.v3"
)

func newOpenAPITestCLI() *CLI {
	o := okapi.New()
	o.Get("/books", func(c *okapi.Context) error { return nil }, okapi.Summary("List books"))
	return New(o).WithOpenAPICommand()
}

func TestOpenAPICommandWritesDocument(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		file, format, version, want string
		yaml                        bool
	}{
		{file: "openapi.json", version: "3.1", want: "3.1.0"},
		{file: "openapi.yaml", version: "3.1", want: "3.1.0", yaml: true},
		{file: "spec.txt", format: "yaml", version: "3.0", want: "3.0.3", yaml: true},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			c := newOpenAPITestCLI()
			path := filepath.Join(dir, tt.file)
			if err := c.writeOpenAPI(path, tt.format, tt.version); err != nil {
				t.Fatalf("writeOpenAPI: %v", err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if tt.yaml {
				err = yaml.Unmarshal(data, &doc)
			} else {
				err = json.Unmarshal(data, &doc)
			}
			if err != nil {
				t.Fatalf("decode: %v\n%s", err, data)
			}
			if doc["openapi"] != tt.want {
				t.Errorf("openapi = %v, want %s", doc["openapi"], tt.want)
			}
			paths, _ := doc["paths"].(map[string]any)
			if _, ok := paths["/books"]; !ok {
				t.Errorf("document has no /books path: %v", paths)
			}
		})
	}
}

func TestOpenAPICommandRejectsUnknownOptions(t *testing.T) {
	c := newOpenAPITestCLI()
	if err := c.writeOpenAPI("", "json", "2.0"); err == nil {
		t.Error("expected an error for an unsupported version")
	}
	if err := c.writeOpenAPI("", "toml", "3.1"); err == nil {
		t.Error("expected an error for an unsupported format")
	}
}

func TestOpenAPICommandIsRegistered(t *testing.T) {
	c := newOpenAPITestCLI()
	if _, ok := c.commands["openapi"]; !ok {
		t.Fatal("openapi command is not registered")
	}
}
