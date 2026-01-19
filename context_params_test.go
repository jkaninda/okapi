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
	"fmt"
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestParam(t *testing.T) {

	o := Default()
	o.Get("/api/:version/users/:id", func(c *Context) error {
		version := c.Param("version")
		q := c.Query("q")
		tags := c.QueryArray("tags")
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.ErrorBadRequest(errors.New("invalid id"))
		}

		body := fmt.Sprintf(`{"version":"%s","user_id":%d,"q":"%s","tags":"%v"}`, version, id, q, strings.Join(tags, ","))
		return c.String(http.StatusOK, body)
	})

	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	body := `{"version":"v1","user_id":1}`
	res := `{"version":"v1","user_id":1,"q":"Hello","tags":"hp,pc,mini"}`

	okapitest.GET(t, "http://localhost:8080/api/v1/users/1?q=Hello&tags=hp,pc&tags=mini").ExpectStatusOK().Body(strings.NewReader(body)).ExpectBody(res)

}
