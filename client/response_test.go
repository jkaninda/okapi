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

package client_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/jkaninda/okapi/client"
)

func TestResponse_JSONPath(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"user":{"profile":{"name":"Ada"}}}`)
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	got, ok := resp.JSONPath("user.profile.name")
	if !ok {
		t.Fatal("path not found")
	}
	if got != testName {
		t.Errorf("got = %v, want %s", got, testName)
	}
	if _, ok := resp.JSONPath("user.missing"); ok {
		t.Error("expected missing path to return ok=false")
	}
}

func TestResponse_Cookie(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "xyz"})
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	cookie := resp.Cookie("sid")
	if cookie == nil || cookie.Value != "xyz" {
		t.Errorf("Cookie = %+v, want value xyz", cookie)
	}
	if resp.Cookie("missing") != nil {
		t.Error("expected missing cookie nil")
	}
}

func TestResponse_YAMLDecode(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "name: Ada\nage: 36\n")
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	var out struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}
	if err := resp.YAML(&out); err != nil {
		t.Fatalf("YAML: %v", err)
	}
	if out.Name != testName || out.Age != 36 {
		t.Errorf("out = %+v", out)
	}
}

func TestResponse_XMLDecode(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<u><name>Ada</name></u>`)
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	var out struct {
		XMLName struct{} `xml:"u"`
		Name    string   `xml:"name"`
	}
	if err := resp.XML(&out); err != nil {
		t.Fatalf("XML: %v", err)
	}
	if out.Name != testName {
		t.Errorf("name = %s", out.Name)
	}
}

func TestResponse_StringAndIsSuccess(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, "queued")
	})
	c := client.New(srv.URL)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !resp.IsSuccess() {
		t.Error("202 should be a success")
	}
	if resp.String() != "queued" {
		t.Errorf("string = %q", resp.String())
	}
	if resp.Error() != nil {
		t.Errorf("Error = %v", resp.Error())
	}
}

func TestResponse_Decode_ByContentType(t *testing.T) {
	type payload struct {
		XMLName struct{} `xml:"u" json:"-" yaml:"-"`
		Name    string   `json:"name" xml:"name" yaml:"name"`
	}
	cases := []struct {
		name        string
		contentType string
		body        string
	}{
		{"json", "application/json", `{"name":"Ada"}`},
		{"xml", "application/xml", `<u><name>Ada</name></u>`},
		{"yaml", "application/yaml", "name: Ada\n"},
		{"unknown defaults to json", "text/plain", `{"name":"Ada"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				_, _ = io.WriteString(w, tc.body)
			})
			c := client.New(srv.URL)
			resp, err := c.Get("/").Do()
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			var out payload
			if err := resp.Decode(&out); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if out.Name != testName {
				t.Errorf("name = %q, want %s", out.Name, testName)
			}
		})
	}
}

func TestRequestBuilder_Decode_UsesContentType(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = io.WriteString(w, `<u><name>Ada</name></u>`)
	})
	c := client.New(srv.URL)
	var out struct {
		XMLName struct{} `xml:"u"`
		Name    string   `xml:"name"`
	}
	if err := c.Get("/").Decode(&out); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if out.Name != testName {
		t.Errorf("name = %q, want %s", out.Name, testName)
	}
}
