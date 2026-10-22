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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

// WithOpenAPICommand registers an "openapi" command that writes the
// application's OpenAPI document, so it can be checked in or fed to a client
// generator without starting the server:
//
//	app openapi                         # OpenAPI 3.1 JSON on stdout
//	app openapi -o openapi.yaml         # format taken from the extension
//	app openapi --version 3.0 -f yaml
//
// Routes must be registered before Execute is called. As with any command,
// a CLI that also serves needs a command for that, set as DefaultCommand to
// keep running the server without arguments.
func (c *CLI) WithOpenAPICommand() *CLI {
	c.Command("openapi", "Write the OpenAPI document", func(cmd *Command) error {
		return c.writeOpenAPI(cmd.GetString("output"), cmd.GetString("format"), cmd.GetString("version"))
	}).
		String("output", "o", "", "File to write the document to (default: stdout)").
		String("format", "f", "", "Document format, json or yaml (default: from the output extension, else json)").
		String("version", "v", "3.1", "OpenAPI version, 3.1 or 3.0")
	return c
}

// writeOpenAPI writes the OpenAPI document of the given version and format to
// output, or to stdout when output is empty.
func (c *CLI) writeOpenAPI(output, format, version string) error {
	var spec *openapi3.T
	switch version {
	case "3.1", "3.1.0":
		spec = c.o.OpenAPISpec()
	case "3.0", "3.0.3":
		spec = c.o.OpenAPISpec30()
	default:
		return fmt.Errorf("unsupported OpenAPI version %q: use 3.1 or 3.0", version)
	}

	if format == "" {
		format = "json"
		if ext := strings.ToLower(filepath.Ext(output)); ext == ".yaml" || ext == ".yml" {
			format = "yaml"
		}
	}
	data, err := marshalSpec(spec, format)
	if err != nil {
		return err
	}

	var w io.Writer = os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", output, err)
		}
		defer f.Close()
		w = f
	}
	_, err = w.Write(data)
	return err
}

// marshalSpec encodes spec as indented JSON or as YAML.
func marshalSpec(spec *openapi3.T, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case "yaml", "yml":
		return yaml.Marshal(spec)
	default:
		return nil, fmt.Errorf("unsupported format %q: use json or yaml", format)
	}
}
