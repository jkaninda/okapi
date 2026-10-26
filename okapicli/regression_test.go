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
	"testing"
	"time"

	"github.com/jkaninda/okapi"
)

func TestCLIGetReturnsFlagValues(t *testing.T) {
	defer setOSArgs("--name", "books", "--port", "9000", "--debug", "--ratio", "0.5", "--timeout", "2s")()
	c := New(okapi.New()).
		String("name", "n", "", "").
		Int("port", "p", 8080, "").
		Bool("debug", "d", false, "").
		Float("ratio", "r", 0, "").
		Duration("timeout", "t", time.Second, "")
	if err := c.ParseFlags(); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "books", "port": 9000, "debug": true, "ratio": 0.5, "timeout": 2 * time.Second}
	for name, v := range want {
		if got := c.Get(name); got != v {
			t.Errorf("Get(%q) = %#v, want %#v", name, got, v)
		}
	}
	if got := c.Get("missing"); got != nil {
		t.Errorf("Get(missing) = %#v, want nil", got)
	}
}

func TestCLIFromStructWritesBackDuration(t *testing.T) {
	type config struct {
		Timeout time.Duration `cli:"timeout" default:"5s"`
		Port    int           `cli:"port" default:"8080"`
	}
	defer setOSArgs("--timeout", "3m", "--port", "9000")()
	cfg := &config{}
	if err := New(okapi.New()).FromStruct(cfg).ParseFlags(); err != nil {
		t.Fatal(err)
	}
	if cfg.Timeout != 3*time.Minute || cfg.Port != 9000 {
		t.Errorf("got %+v, want Timeout 3m and Port 9000", *cfg)
	}
}
