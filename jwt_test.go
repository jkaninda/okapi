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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtTestSecret is a stable HMAC key used by tests in this file.
var jwtTestSecret = []byte("super-secret-jwt-test-key")

// signHMACToken signs claims with HS256 and the test secret.
func signHMACToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtTestSecret)
	if err != nil {
		t.Fatalf("sign HMAC token: %v", err)
	}
	return signed
}

// signRSAToken signs claims with RS256 and the supplied private key.
func signRSAToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign RSA token: %v", err)
	}
	return signed
}

// extractToken

func TestExtractToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lookup    string
		setup     func(req *http.Request)
		wantToken string
		wantErr   bool
	}{
		{
			name:   "default header lookup with Bearer prefix",
			lookup: "",
			setup: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer abc.def.ghi")
			},
			wantToken: "abc.def.ghi",
		},
		{
			name:   "header lookup without Bearer prefix returns raw value",
			lookup: "header:Authorization",
			setup: func(req *http.Request) {
				req.Header.Set("Authorization", "raw-token")
			},
			wantToken: "raw-token",
		},
		{
			name:   "custom header",
			lookup: "header:X-Auth",
			setup: func(req *http.Request) {
				req.Header.Set("X-Auth", "custom-token")
			},
			wantToken: "custom-token",
		},
		{
			name:   "query lookup",
			lookup: "query:token",
			setup: func(req *http.Request) {
				q := req.URL.Query()
				q.Set("token", "from-query")
				req.URL.RawQuery = q.Encode()
			},
			wantToken: "from-query",
		},
		{
			name:   "cookie lookup",
			lookup: "cookie:jwt",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: "jwt", Value: "from-cookie"})
			},
			wantToken: "from-cookie",
		},
		{
			name:    "missing cookie errors",
			lookup:  "cookie:jwt",
			setup:   func(req *http.Request) {},
			wantErr: true,
		},
		{
			name:    "invalid lookup format",
			lookup:  "garbage",
			setup:   func(req *http.Request) {},
			wantErr: true,
		},
		{
			name:    "unsupported source",
			lookup:  "body:token",
			setup:   func(req *http.Request) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, _ := NewTestContext(http.MethodGet, "/", nil)
			tt.setup(ctx.Request())

			auth := &JWTAuth{TokenLookup: tt.lookup}
			got, err := auth.extractToken(ctx)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got token %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantToken {
				t.Errorf("token = %q, want %q", got, tt.wantToken)
			}
		})
	}
}

// ValidateToken

func TestValidateToken(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{
		SigningSecret: jwtTestSecret,
		TokenLookup:   "header:Authorization",
	}

	t.Run("valid HMAC token", func(t *testing.T) {
		t.Parallel()
		token := signHMACToken(t, jwt.MapClaims{
			"sub": "alice",
			"exp": time.Now().Add(time.Hour).Unix(),
		})

		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Request().Header.Set("Authorization", "Bearer "+token)

		claims, err := auth.ValidateToken(ctx)
		if err != nil {
			t.Fatalf("ValidateToken: %v", err)
		}
		if claims["sub"] != "alice" {
			t.Errorf("sub = %v, want alice", claims["sub"])
		}
	})

	t.Run("expired token rejected", func(t *testing.T) {
		t.Parallel()
		token := signHMACToken(t, jwt.MapClaims{
			"sub": "alice",
			"exp": time.Now().Add(-time.Hour).Unix(),
		})

		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Request().Header.Set("Authorization", "Bearer "+token)

		if _, err := auth.ValidateToken(ctx); err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("RSA token rejected by HMAC validator", func(t *testing.T) {
		t.Parallel()
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
		token := signRSAToken(t, key, jwt.MapClaims{
			"sub": "alice",
			"exp": time.Now().Add(time.Hour).Unix(),
		}, "")

		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Request().Header.Set("Authorization", "Bearer "+token)

		if _, err := auth.ValidateToken(ctx); err == nil {
			t.Error("expected error for non-HMAC token")
		}
	})

	t.Run("malformed token rejected", func(t *testing.T) {
		t.Parallel()
		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Request().Header.Set("Authorization", "Bearer not.a.jwt")

		if _, err := auth.ValidateToken(ctx); err == nil {
			t.Error("expected error for malformed token")
		}
	})
}

// resolveKeyFunc

func TestResolveKeyFunc(t *testing.T) {
	t.Parallel()

	t.Run("uses SigningSecret when set", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret}
		kf, err := auth.resolveKeyFunc()
		if err != nil {
			t.Fatalf("resolveKeyFunc: %v", err)
		}
		got, err := kf(&jwt.Token{})
		if err != nil {
			t.Fatalf("keyfunc: %v", err)
		}
		if string(got.([]byte)) != string(jwtTestSecret) {
			t.Errorf("keyfunc returned wrong secret")
		}
	})

	t.Run("falls back to legacy SecretKey", func(t *testing.T) {
		t.Parallel()
		legacy := []byte("legacy")
		auth := &JWTAuth{SecretKey: legacy}
		kf, err := auth.resolveKeyFunc()
		if err != nil {
			t.Fatalf("resolveKeyFunc: %v", err)
		}
		got, err := kf(&jwt.Token{})
		if err != nil {
			t.Fatalf("keyfunc: %v", err)
		}
		if string(got.([]byte)) != "legacy" {
			t.Errorf("expected legacy secret, got %q", got)
		}
	})

	t.Run("RSA key", func(t *testing.T) {
		t.Parallel()
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
		auth := &JWTAuth{RsaKey: &key.PublicKey}
		kf, err := auth.resolveKeyFunc()
		if err != nil {
			t.Fatalf("resolveKeyFunc: %v", err)
		}
		got, err := kf(&jwt.Token{})
		if err != nil {
			t.Fatalf("keyfunc: %v", err)
		}
		if got != &key.PublicKey {
			t.Errorf("expected RSA public key pointer")
		}
	})

	t.Run("JwksFile", func(t *testing.T) {
		t.Parallel()
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
		jwks := &Jwks{Keys: []Jwk{rsaJWK(t, "test-kid", &key.PublicKey)}}

		auth := &JWTAuth{JwksFile: jwks}
		kf, err := auth.resolveKeyFunc()
		if err != nil {
			t.Fatalf("resolveKeyFunc: %v", err)
		}

		token := &jwt.Token{Header: map[string]any{"kid": "test-kid"}}
		got, err := kf(token)
		if err != nil {
			t.Fatalf("keyfunc: %v", err)
		}
		if _, ok := got.(*rsa.PublicKey); !ok {
			t.Errorf("expected *rsa.PublicKey, got %T", got)
		}
	})

	t.Run("JwksFile rejects token without kid", func(t *testing.T) {
		t.Parallel()
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
		jwks := &Jwks{Keys: []Jwk{rsaJWK(t, "k1", &key.PublicKey)}}
		auth := &JWTAuth{JwksFile: jwks}
		kf, _ := auth.resolveKeyFunc()
		if _, err := kf(&jwt.Token{Header: map[string]any{}}); err == nil {
			t.Error("expected error when kid missing")
		}
	})

	t.Run("JwksUrl fetches keys", func(t *testing.T) {
		t.Parallel()
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("generate RSA key: %v", err)
		}
		jwks := &Jwks{Keys: []Jwk{rsaJWK(t, "remote-kid", &key.PublicKey)}}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(jwks)
		}))
		t.Cleanup(srv.Close)

		auth := &JWTAuth{JwksUrl: srv.URL}
		kf, err := auth.resolveKeyFunc()
		if err != nil {
			t.Fatalf("resolveKeyFunc: %v", err)
		}
		token := &jwt.Token{Header: map[string]any{"kid": "remote-kid"}}
		got, err := kf(token)
		if err != nil {
			t.Fatalf("keyfunc: %v", err)
		}
		if _, ok := got.(*rsa.PublicKey); !ok {
			t.Errorf("expected *rsa.PublicKey, got %T", got)
		}
	})

	t.Run("nothing configured returns error", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{}
		kf, err := auth.resolveKeyFunc()
		if err == nil {
			t.Error("expected error when no verifier is configured")
		}
		if kf != nil {
			t.Error("expected nil keyfunc")
		}
	})
}

// validateJWTClaims (expression-driven)

func TestValidateJWTClaims_Expression(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{ClaimsExpression: "Equals(`role`, `admin`)"}
	claims := jwt.MapClaims{"role": "admin"}
	tok := &jwt.Token{Claims: claims}

	ok, err := auth.validateJWTClaims(tok)
	if err != nil {
		t.Fatalf("validateJWTClaims: %v", err)
	}
	if !ok {
		t.Errorf("expected expression to pass")
	}

	// Caches the parsed expression on first call.
	if auth.parsedExpression == nil {
		t.Errorf("parsedExpression should be cached")
	}

	// Second call should hit the cache and behave the same.
	if ok, err := auth.validateJWTClaims(tok); !ok || err != nil {
		t.Errorf("cached evaluation: got (%v, %v)", ok, err)
	}
}

func TestValidateJWTClaims_NoExpressionPassesThrough(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{}
	tok := &jwt.Token{Claims: jwt.MapClaims{}}
	ok, err := auth.validateJWTClaims(tok)
	if err != nil {
		t.Fatalf("validateJWTClaims: %v", err)
	}
	if !ok {
		t.Error("expected pass-through when no expression configured")
	}
}

func TestValidateJWTClaims_BadExpression(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{ClaimsExpression: "BadFn(`x`)"}
	tok := &jwt.Token{Claims: jwt.MapClaims{}}
	if _, err := auth.validateJWTClaims(tok); err == nil {
		t.Error("expected error for invalid expression")
	}
}

// forwardContextFromClaims & formatContextValue

func TestForwardContextFromClaims(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{
		ForwardClaims: map[string]string{
			"email":  "user.email",
			"role":   "user.role",
			"id":     "user.id",
			"flag":   "verified",
			"groups": "groups",
		},
	}
	tok := &jwt.Token{Claims: jwt.MapClaims{
		"user": map[string]any{
			"email": "jane@example.com",
			"role":  "admin",
			"id":    float64(42),
		},
		"verified": true,
		"groups":   []any{"eng", "ops"},
	}}

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)
	if err := auth.forwardContextFromClaims(tok, ctx); err != nil {
		t.Fatalf("forwardContextFromClaims: %v", err)
	}

	wantStrings := map[string]string{
		"email":  "jane@example.com",
		"role":   "admin",
		"id":     "42",
		"flag":   "true",
		"groups": "eng,ops",
	}
	for k, want := range wantStrings {
		if got := ctx.GetString(k); got != want {
			t.Errorf("ctx[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestForwardContextFromClaims_MissingClaim(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{ForwardClaims: map[string]string{
		"email": "user.email",
	}}
	tok := &jwt.Token{Claims: jwt.MapClaims{}}

	ctx, _ := NewTestContext(http.MethodGet, "/", nil)
	if err := auth.forwardContextFromClaims(tok, ctx); err != nil {
		t.Fatalf("forwardContextFromClaims should swallow missing claim, got %v", err)
	}
	if got := ctx.GetString("email"); got != "" {
		t.Errorf("ctx[email] = %q, want empty", got)
	}
}

func TestFormatContextValue(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{}
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"string", "hello", "hello"},
		{"float64 rendered as integer", float64(42), "42"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"array of strings joined by comma", []any{"a", "b", "c"}, "a,b,c"},
		{"array with mixed types", []any{"a", float64(2)}, "a,2"},
		{"fallback to fmt", map[string]any{"k": "v"}, "map[k:v]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := auth.formatContextValue(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// extractNestedClaimValue

func TestExtractNestedClaimValue(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{}
	claims := jwt.MapClaims{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep",
			},
		},
	}

	t.Run("nested found", func(t *testing.T) {
		got, err := auth.extractNestedClaimValue(claims, "a.b.c")
		if err != nil || got != "deep" {
			t.Errorf("got (%v, %v), want (deep, nil)", got, err)
		}
	})

	t.Run("missing nested key errors", func(t *testing.T) {
		if _, err := auth.extractNestedClaimValue(claims, "a.b.missing"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("traverse non-object errors", func(t *testing.T) {
		if _, err := auth.extractNestedClaimValue(claims, "a.b.c.deeper"); err == nil {
			t.Error("expected error when traversing into a string")
		}
	})
}

// GenerateJwtToken

func TestGenerateJwtToken_RoundTrip(t *testing.T) {
	t.Parallel()

	signed, err := GenerateJwtToken(jwtTestSecret, jwt.MapClaims{"sub": "alice"}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJwtToken: %v", err)
	}
	if signed == "" {
		t.Fatal("empty token")
	}

	parsed, err := jwt.Parse(signed, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			t.Fatalf("unexpected signing method: %T", token.Method)
		}
		return jwtTestSecret, nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse: %v, valid=%v", err, parsed.Valid)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "alice" {
		t.Errorf("sub = %v, want alice", claims["sub"])
	}
	// Both exp and iat should have been injected by the helper.
	if _, ok := claims["exp"]; !ok {
		t.Error("missing exp claim")
	}
	if _, ok := claims["iat"]; !ok {
		t.Error("missing iat claim")
	}
}
