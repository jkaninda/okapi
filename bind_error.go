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
	"errors"
	"strings"
)

// BindError reports a request that could not be bound into a handler's input
// type. Error returns only the field-level reason, which is safe to send to
// clients; Input names the Go type for logs and custom error handlers.
type BindError struct {
	// Input is the handler's input type, e.g. "main.CreateBookInput".
	Input string
	Err   error
}

func (e *BindError) Error() string { return e.Err.Error() }

func (e *BindError) Unwrap() error { return e.Err }

var errRequired = errors.New("is required")

// fieldError is a bind or validation failure at a field path such as Body.Items[2].Name.
type fieldError struct {
	path string
	err  error
}

func (e *fieldError) Error() string {
	if e.err == errRequired {
		return "field " + e.path + " is required"
	}
	return "field " + e.path + ": " + e.err.Error()
}

func (e *fieldError) Unwrap() error { return e.err }

// withField prefixes err's field path with name, so nested failures read as one path.
func withField(name string, err error) error {
	fe, ok := err.(*fieldError)
	if !ok {
		return &fieldError{path: name, err: err}
	}
	if strings.HasPrefix(fe.path, "[") {
		return &fieldError{path: name + fe.path, err: fe.err}
	}
	return &fieldError{path: name + "." + fe.path, err: fe.err}
}

// abortBind answers a request that failed to bind into input with 400 and logs
// the input type, which the response leaves out.
func (c *Context) abortBind(input string, err error) error {
	if c.okapi != nil && c.okapi.logger != nil {
		c.okapi.logger.Debug("request binding failed",
			"input", input,
			"error", err.Error(),
			"method", c.request.Method,
			"path", c.request.URL.Path,
			"component", "okapi",
		)
	}
	return c.AbortBadRequest("Bad Request", &BindError{Input: input, Err: err})
}
