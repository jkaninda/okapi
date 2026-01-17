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

package okapitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test server handlers
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Simple GET endpoint
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Hello, World!"))
		if err != nil {
			return
		}
	})

	// JSON endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]any{
			"message": "success",
			"data": map[string]any{
				"id":   123,
				"name": "test",
			},
		})
		if err != nil {
			return
		}
	})

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var data map[string]any
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte("Invalid JSON"))
			if err != nil {
				return
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			return
		}
	})

	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			w.Header().Set("X-Auth-Echo", auth)
		}
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			return
		}
	})

	// Status code endpoint
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		code := http.StatusOK
		path := r.URL.Path
		if strings.HasSuffix(path, "400") {
			code = http.StatusBadRequest
		} else if strings.HasSuffix(path, "401") {
			code = http.StatusUnauthorized
		} else if strings.HasSuffix(path, "404") {
			code = http.StatusNotFound
		} else if strings.HasSuffix(path, "500") {
			code = http.StatusInternalServerError
		}
		w.WriteHeader(code)
		_, err := w.Write([]byte(http.StatusText(code)))
		if err != nil {
			return
		}
	})

	// Form data endpoint
	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]string{
			"username": username,
			"password": password,
		})
		if err != nil {
			return
		}
	})

	// Slow endpoint for timeout testing
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Done"))
		if err != nil {
			return
		}
	})

	mux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return httptest.NewServer(mux)
}

func TestGET(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/hello").
		ExpectStatusOK().
		ExpectBody("Hello, World!")
}

func TestPOST(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	POST(t, server.URL+"/echo").
		JSONBody(map[string]string{"name": "John"}).
		ExpectStatusCreated().
		ExpectBodyContains("John")
}

func TestPUT(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := PUT(t, server.URL+"/echo")
	if rb.method != http.MethodPut {
		t.Errorf("Expected method PUT, got %s", rb.method)
	}
}

func TestDELETE(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := DELETE(t, server.URL+"/hello")
	if rb.method != http.MethodDelete {
		t.Errorf("Expected method DELETE, got %s", rb.method)
	}
}

func TestPATCH(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := PATCH(t, server.URL+"/hello")
	if rb.method != http.MethodPatch {
		t.Errorf("Expected method PATCH, got %s", rb.method)
	}
}

func TestHEAD(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := HEAD(t, server.URL+"/hello")
	if rb.method != http.MethodHead {
		t.Errorf("Expected method HEAD, got %s", rb.method)
	}
}

func TestOPTIONS(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := OPTIONS(t, server.URL+"/hello")
	if rb.method != http.MethodOptions {
		t.Errorf("Expected method OPTIONS, got %s", rb.method)
	}
}

func TestRequestBuilder_Headers(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/headers").
		Header("Authorization", "Bearer token123").
		ExpectStatusOK().
		ExpectHeader("X-Auth-Echo", "Bearer token123")
}

func TestRequestBuilder_MultipleHeaders(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/headers").
		Headers(map[string]string{
			"Authorization": "Bearer token123",
			"X-Request-ID":  "req-456",
		}).
		ExpectStatusOK().
		ExpectHeader("X-Auth-Echo", "Bearer token123")
}

func TestRequestBuilder_JSONBody(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user := User{Name: "Alice", Email: "alice@example.com"}

	POST(t, server.URL+"/echo").
		JSONBody(user).
		ExpectStatusCreated().
		ExpectBodyContains("Alice").
		ExpectBodyContains("alice@example.com")
}

func TestRequestBuilder_JSONBodyString(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	POST(t, server.URL+"/echo").
		JSONBody(`{"name":"Bob"}`).
		ExpectStatusCreated().
		ExpectBodyContains("Bob")
}

func TestRequestBuilder_FormBody(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	POST(t, server.URL+"/form").
		FormBody(map[string]string{
			"username": "admin",
			"password": "secret",
		}).
		ExpectStatusOK().
		ExpectBodyContains("admin").
		ExpectBodyContains("secret")
}

func TestRequestBuilder_ExpectStatus(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	tests := []struct {
		path   string
		status int
	}{
		{"/status/400", http.StatusBadRequest},
		{"/status/401", http.StatusUnauthorized},
		{"/status/404", http.StatusNotFound},
		{"/status/500", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			GET(t, server.URL+tt.path).ExpectStatus(tt.status)
		})
	}
}

func TestRequestBuilder_StatusHelpers(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	t.Run("ExpectStatusBadRequest", func(t *testing.T) {
		GET(t, server.URL+"/status/400").ExpectStatusBadRequest()
	})

	t.Run("ExpectStatusUnauthorized", func(t *testing.T) {
		GET(t, server.URL+"/status/401").ExpectStatusUnauthorized()
	})

	t.Run("ExpectStatusNotFound", func(t *testing.T) {
		GET(t, server.URL+"/status/404").ExpectStatusNotFound()
	})

	t.Run("ExpectStatusInternalServerError", func(t *testing.T) {
		GET(t, server.URL+"/status/500").ExpectStatusInternalServerError()
	})
}

func TestRequestBuilder_ExpectJSON(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	expected := map[string]any{
		"message": "success",
		"data": map[string]any{
			"id":   float64(123),
			"name": "test",
		},
	}

	GET(t, server.URL+"/json").
		ExpectStatusOK().
		ExpectJSON(expected)
}

func TestRequestBuilder_ParseJSON(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	type Response struct {
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}

	var resp Response
	GET(t, server.URL+"/json").
		ExpectStatusOK().
		ParseJSON(&resp)

	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got %s", resp.Message)
	}
}

func TestRequestBuilder_ExpectJSONPath(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/json").
		ExpectStatusOK().
		ExpectJSONPath("message", "success").
		ExpectJSONPath("data.name", "test")
}

func TestRequestBuilder_ExpectBodyContains(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/hello").
		ExpectStatusOK().
		ExpectBodyContains("Hello").
		ExpectBodyContains("World")
}

func TestRequestBuilder_ExpectBodyNotContains(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/hello").
		ExpectStatusOK().
		ExpectBodyNotContains("Goodbye")
}

func TestRequestBuilder_ExpectEmptyBody(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/empty").
		ExpectStatusNoContent().
		ExpectEmptyBody()
}

func TestRequestBuilder_ExpectHeaderExists(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/headers").
		ExpectStatusOK().
		ExpectHeaderExists("X-Custom-Header")
}

func TestRequestBuilder_ExpectHeaderContains(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/headers").
		Header("Authorization", "Bearer token123").
		ExpectStatusOK().
		ExpectHeaderContains("X-Auth-Echo", "Bearer")
}

func TestRequestBuilder_ExpectContentType(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/json").
		ExpectStatusOK().
		ExpectContentType("application/json")
}

func TestRequestBuilder_Chaining(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	GET(t, server.URL+"/hello").
		ExpectStatusOK().
		ExpectBody("Hello, World!").
		ExpectBodyContains("Hello").
		ExpectBodyNotContains("Goodbye")
}

func TestRequestBuilder_Execute(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	resp, body := GET(t, server.URL+"/hello").Execute()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != "Hello, World!" {
		t.Errorf("Expected body 'Hello, World!', got %s", string(body))
	}
}

func TestRequestBuilder_CachesResponse(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := GET(t, server.URL+"/hello")

	resp1, body1 := rb.do()

	resp2, body2 := rb.do()

	if resp1 != resp2 {
		t.Error("Expected response to be cached")
	}

	if string(body1) != string(body2) {
		t.Error("Expected body to be cached")
	}
}

func TestRequestBuilder_Timeout(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	rb := GET(t, server.URL+"/slow").Timeout(100 * time.Millisecond)

	if rb.timeout != 100*time.Millisecond {
		t.Errorf("Expected timeout 100ms, got %v", rb.timeout)
	}
}

// Legacy function tests
func TestAssertHTTPStatus(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	AssertHTTPStatus(
		t,
		http.MethodGet,
		server.URL+"/hello",
		nil,
		nil,
		"",
		http.StatusOK,
	)
}

func TestAssertHTTPResponse(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	AssertHTTPResponse(
		t,
		http.MethodGet,
		server.URL+"/hello",
		nil,
		nil,
		"",
		http.StatusOK,
		"Hello, World!",
	)
}

func TestExtractJSONPath(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "John",
			"age":  30,
		},
	}

	tests := []struct {
		path     string
		expected any
	}{
		{"user.name", "John"},
		{"user.age", 30},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractJSONPath(data, tt.path)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkRequestBuilder(b *testing.B) {
	server := setupTestServer()
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GET(&testing.T{}, server.URL+"/hello").Execute()
	}
}

func BenchmarkRequestBuilderWithJSON(b *testing.B) {
	server := setupTestServer()
	defer server.Close()

	payload := map[string]string{"name": "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		POST(&testing.T{}, server.URL+"/echo").
			JSONBody(payload).
			Execute()
	}
}
