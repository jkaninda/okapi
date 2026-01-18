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
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"testing"
)

func TestRegisterDocRoutes(t *testing.T) {
	o := New()
	o.Get("/", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	})

	o.registerDocRoutes(o.openAPI.Title)

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.AssertHTTPStatus(t, "GET", "http://localhost:8080/openapi.json", nil, nil, "", http.StatusOK)
	okapitest.AssertHTTPStatus(t, "GET", "http://localhost:8080/docs", nil, nil, "", http.StatusOK)
	okapitest.AssertHTTPStatus(t, "GET", "http://localhost:8080/redoc", nil, nil, "", http.StatusOK)

}
