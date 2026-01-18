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

type User struct {
	Name string `json:"name" required:"true" xml:"name" form:"name" query:"name" yaml:"name"`
}

func TestContext_Bind(t *testing.T) {
	o := Default()

	o.Get("/", func(c *Context) error {
		return c.XML(http.StatusOK, books)
	})
	o.Get("/hello", func(c *Context) error {
		return c.Text(http.StatusOK, "Hello World!")
	})
	o.Post("/hello", func(c *Context) error {
		user := User{}
		if err := c.Bind(&user); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		if ok, err := c.ShouldBind(&user); !ok {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Put("/hello", func(c *Context) error {
		user := User{}
		if err := c.B(&user); err != nil {
			return c.AbortBadRequest("Bad requests")
		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Get("/hello", func(c *Context) error {
		return c.JSON(http.StatusOK, books)
	})

	o.Post("/bind", func(c *Context) error {
		user := User{}
		if ok, err := c.ShouldBind(&user); !ok {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/multipart", func(c *Context) error {
		user := User{}
		if err := c.BindMultipart(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/xml", func(c *Context) error {
		user := User{}
		if err := c.BindXML(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/form", func(c *Context) error {
		user := User{}
		if err := c.BindForm(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Get("/query", func(c *Context) error {
		user := User{}
		if err := c.BindQuery(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/json", func(c *Context) error {
		user := User{}
		if err := c.BindJSON(&user); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/yaml", func(c *Context) error {
		user := User{}
		if err := c.BindYAML(&user); err != nil {
			return c.ErrorBadRequest("Bad requests")

		}
		return c.JSON(http.StatusCreated, user)
	})
	o.Post("/protobuf", func(c *Context) error {
		if err := c.BindProtoBuf(nil); err != nil {
			return c.AbortBadRequest("Bad requests", err)

		}
		return c.OK(http.StatusOK)
	})
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
	okapitest.AssertHTTPStatus(t, "GET", "http://localhost:8080", nil, nil, "", http.StatusOK)
	okapitest.AssertHTTPStatus(t, "POST", "http://localhost:8080/hello", nil, nil, "", http.StatusBadRequest)
	okapitest.AssertHTTPStatus(t, "POST", "http://localhost:8080/json", nil, nil, "", http.StatusBadRequest)

}
