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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const ip = "198.51.100.7"

func TestNormalizeRoutePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "colon param",
			input:    "/users/:id",
			expected: "/users/{id}",
		},
		{
			name:     "colon param with type",
			input:    "/users/:id:int",
			expected: "/users/{id}",
		},
		{
			name:     "colon param with type and trailing slash",
			input:    "/users/:id:int/",
			expected: "/users/{id}/",
		},
		{
			name:     "brace param with type",
			input:    "/users/{id:int}",
			expected: "/users/{id}",
		},
		{
			name:     "wildcard only",
			input:    "/*",
			expected: "/{any...}",
		},
		{
			name:     "named wildcard",
			input:    "/*any",
			expected: "/{any...}",
		},
		{
			name:     "custom wildcard name ignored",
			input:    "/*path",
			expected: "/{any...}",
		},
		{
			name:     "mixed params and wildcard",
			input:    "/users/:id/books/*",
			expected: "/users/{id}/books/{any...}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRoutePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAddr(t *testing.T) {
	addr := "0.0.0.0:8080"
	if !ValidateAddr(addr) {
		t.Errorf("Invalid addr: %s", addr)
	}

}

func TestLoadJWKSFromFile(t *testing.T) {
	_, err := LoadJWKSFromFile("testdata/jwks.json")
	if err == nil {
		t.Errorf("LoadJWKSFromFile should have returned an error")
	}

}

// TestRealIPTrustedProxies covers the forwarding headers, which any client can
// set: without a trusted-proxy list RealIP returns attacker-controlled data.
func TestRealIPTrustedProxies(t *testing.T) {
	newReq := func(remote, xff, xrip string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = remote
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		if xrip != "" {
			r.Header.Set("X-Real-IP", xrip)
		}
		return r
	}

	t.Run("unconfigured trusts the headers", func(t *testing.T) {
		got := realIP(newReq("203.0.113.9:1234", "127.0.0.1", ""), nil)
		if got != "127.0.0.1" {
			t.Errorf("realIP = %q, want %q (documented default)", got, "127.0.0.1")
		}
	})

	trusted, err := parseCIDRs([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("parseCIDRs: %v", err)
	}

	t.Run("headers ignored from an untrusted peer", func(t *testing.T) {
		got := realIP(newReq("203.0.113.9:1234", "127.0.0.1", ""), trusted)
		if got != "203.0.113.9" {
			t.Errorf("realIP = %q, want the connection address", got)
		}
	})

	t.Run("headers honoured from a trusted proxy", func(t *testing.T) {
		got := realIP(newReq("10.1.2.3:4567", "198.51.100.7, 10.1.2.3", ""), trusted)
		if got != ip {
			t.Errorf("realIP = %q, want the forwarded client", got)
		}
	})

	t.Run("X-Real-IP follows the same rule", func(t *testing.T) {
		if got := realIP(newReq("203.0.113.9:1234", "", "127.0.0.1"), trusted); got != "203.0.113.9" {
			t.Errorf("untrusted peer: realIP = %q", got)
		}
		if got := realIP(newReq("10.1.2.3:4567", "", ip), trusted); got != ip {
			t.Errorf("trusted proxy: realIP = %q", got)
		}
	})

	t.Run("bare addresses are accepted as entries", func(t *testing.T) {
		only, err := parseCIDRs([]string{"192.0.2.7"})
		if err != nil {
			t.Fatalf("parseCIDRs: %v", err)
		}
		if got := realIP(newReq("192.0.2.7:80", ip, ""), only); got != ip {
			t.Errorf("realIP = %q, want the forwarded client", got)
		}
		if got := realIP(newReq("192.0.2.8:80", ip, ""), only); got != "192.0.2.8" {
			t.Errorf("realIP = %q, want the connection address", got)
		}
	})

	t.Run("invalid entries are rejected", func(t *testing.T) {
		if _, err := parseCIDRs([]string{"not-an-ip"}); err == nil {
			t.Error("expected an error for a malformed entry")
		}
	})
}

// TestDefaultServerTimeouts guards the zero-valued http.Server, on which every
// timeout means "no limit".
func TestDefaultServerTimeouts(t *testing.T) {
	app := New()

	if app.server.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout is 0: slow-header clients are held indefinitely")
	}
	if app.server.IdleTimeout == 0 {
		t.Error("IdleTimeout is 0: idle keep-alive connections are held indefinitely")
	}

	// Read and write timeouts stay unset: a default would cut off legitimate
	// uploads and streaming responses.
	if app.server.ReadTimeout != 0 || app.server.WriteTimeout != 0 {
		t.Errorf("ReadTimeout = %v, WriteTimeout = %v, want both unset",
			app.server.ReadTimeout, app.server.WriteTimeout)
	}

	configured := New(WithReadHeaderTimeout(3))
	if configured.server.ReadHeaderTimeout != 3*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 3s", configured.server.ReadHeaderTimeout)
	}
}
