/*
 *  MIT License
 *
 * Copyright (c) 2026 Jonas Kaninda
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
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/jkaninda/okapi/client"
)

func TestFormBody(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("ct = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.Form.Get("name") != testName {
			t.Errorf("name = %q", r.Form.Get("name"))
		}
	})
	c := client.New(srv.URL)
	if _, err := c.Post("/").FormBody(map[string]string{"name": testName}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestXMLBody(t *testing.T) {
	type In struct {
		XMLName struct{} `xml:"in"`
		Name    string   `xml:"name"`
	}
	var gotCT, gotBody string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	})
	c := client.New(srv.URL)
	if _, err := c.Post("/").XMLBody(In{Name: testName}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotCT != "application/xml" {
		t.Errorf("ct = %q", gotCT)
	}
	if !strings.Contains(gotBody, "<name>Ada</name>") {
		t.Errorf("body = %q", gotBody)
	}
}

func TestYAMLBody(t *testing.T) {
	type In struct {
		Name string `yaml:"name"`
	}
	var gotCT, gotBody string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	})
	c := client.New(srv.URL)
	if _, err := c.Post("/").YAMLBody(In{Name: testName}).Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotCT != "application/yaml" {
		t.Errorf("ct = %q", gotCT)
	}
	if !strings.Contains(gotBody, "name: Ada") {
		t.Errorf("body = %q", gotBody)
	}
}

func TestMultipart(t *testing.T) {
	var gotName string
	var gotFile string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		gotName = r.FormValue("name")
		f, _, err := r.FormFile("upload")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = f.Close() }()
		b, _ := io.ReadAll(f)
		gotFile = string(b)
	})

	c := client.New(srv.URL)
	_, err := c.Post("/upload").
		Multipart(func(w *multipart.Writer) error {
			if err := w.WriteField("name", testName); err != nil {
				return err
			}
			fw, err := w.CreateFormFile("upload", "hello.txt")
			if err != nil {
				return err
			}
			_, err = fw.Write([]byte("hello world"))
			return err
		}).
		Send()
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotName != testName {
		t.Errorf("name = %q", gotName)
	}
	if gotFile != "hello world" {
		t.Errorf("file = %q", gotFile)
	}
}

func TestRequest_BasicAndBearerAuth(t *testing.T) {
	var got string
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	})
	c := client.New(srv.URL)

	if _, err := c.Get("/").BearerToken("abc").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got != "Bearer abc" {
		t.Errorf("Bearer Authorization = %q", got)
	}

	if _, err := c.Get("/").BasicAuth("user", "pass").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(got, "Basic ") {
		t.Errorf("Basic Authorization = %q", got)
	}
}

func TestPath_Append(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users" {
			t.Errorf("path = %s, want /api/users", r.URL.Path)
		}
	})
	c := client.New(srv.URL + "/api")
	if _, err := c.Get("").Path("users").Send(); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestBuildErrorPropagates(t *testing.T) {
	c := client.New("http://example.invalid")
	_, err := c.Post("/").JSONBody(make(chan int)).Send() // chan is not encodable
	if err == nil {
		t.Fatal("expected build error from JSONBody")
	}
}
