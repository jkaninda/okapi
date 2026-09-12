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
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
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
			name:   "combined lookup falls back to query",
			lookup: "header:Authorization,query:token",
			setup: func(req *http.Request) {
				q := req.URL.Query()
				q.Set("token", "from-query")
				req.URL.RawQuery = q.Encode()
			},
			wantToken: "from-query",
		},
		{
			name:   "combined lookup prefers first source",
			lookup: "header:Authorization,query:token",
			setup: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer from-header")
				q := req.URL.Query()
				q.Set("token", "from-query")
				req.URL.RawQuery = q.Encode()
			},
			wantToken: "from-header",
		},
		{
			name:   "combined lookup falls back past missing cookie",
			lookup: "cookie:jwt,header:Authorization",
			setup: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer from-header")
			},
			wantToken: "from-header",
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

	// The compiled expression is cached process-wide, not on the struct: a
	// single *JWTAuth is shared by every concurrent request.
	if _, cached := claimsExprCache[auth.ClaimsExpression]; !cached {
		t.Errorf("compiled expression should be cached")
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

// parserOptions

// serveJWT runs a request carrying tok through the JWT middleware and reports
// the resulting status code.
func serveJWT(auth *JWTAuth, tok string) int {
	app := New()
	app.Use(auth.Middleware)
	app.Get("/probe", func(c *Context) error { return c.String(http.StatusOK, "OK") })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Code
}

// TestParserOptionsAudienceAndIssuerAreOptional guards against passing
// jwt.WithAudience("") / jwt.WithIssuer(""), which registers "" as the
// expected value and makes the claim mandatory, rejecting every real token.
func TestParserOptionsAudienceAndIssuerAreOptional(t *testing.T) {
	t.Parallel()

	exp := time.Now().Add(time.Hour).Unix()

	t.Run("unset audience and issuer accept a valid token", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "exp": exp})
		if got := serveJWT(auth, tok); got != http.StatusOK {
			t.Errorf("status = %d, want %d for a token with no aud/iss", got, http.StatusOK)
		}
	})

	t.Run("unset audience accepts a token that carries one", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "aud": "other", "exp": exp})
		if got := serveJWT(auth, tok); got != http.StatusOK {
			t.Errorf("status = %d, want %d when Audience is unset", got, http.StatusOK)
		}
	})

	t.Run("set audience is still enforced", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, Audience: "api"}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "aud": "wrong", "exp": exp})
		if got := serveJWT(auth, tok); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d for a mismatched aud", got, http.StatusUnauthorized)
		}
	})

	t.Run("set issuer is still enforced", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, Issuer: "iss"}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "iss": "wrong", "exp": exp})
		if got := serveJWT(auth, tok); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d for a mismatched iss", got, http.StatusUnauthorized)
		}
	})
}

// TestParserOptionsRequiresExpiry guards against accepting signed tokens that
// omit "exp": golang-jwt validates the claim only when it is present, so such
// a token would never expire.
func TestParserOptionsRequiresExpiry(t *testing.T) {
	t.Parallel()

	t.Run("token without exp is rejected by default", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, Audience: "api"}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "aud": "api"})
		if got := serveJWT(auth, tok); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d for a token with no exp", got, http.StatusUnauthorized)
		}
	})

	t.Run("AllowMissingExpiry opts back in", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, Audience: "api", AllowMissingExpiry: true}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "aud": "api"})
		if got := serveJWT(auth, tok); got != http.StatusOK {
			t.Errorf("status = %d, want %d with AllowMissingExpiry", got, http.StatusOK)
		}
	})

	t.Run("expired token is still rejected with AllowMissingExpiry", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, AllowMissingExpiry: true}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(-time.Hour).Unix()})
		if got := serveJWT(auth, tok); got != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d for an expired token", got, http.StatusUnauthorized)
		}
	})
}

// TestValidateTokenRequiresExpiry mirrors TestParserOptionsRequiresExpiry for
// the ValidateToken entry point, which shares the same parser configuration.
func TestValidateTokenRequiresExpiry(t *testing.T) {
	t.Parallel()

	newCtx := func(tok string) *Context {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		return &Context{request: req, response: newResponseWriter(httptest.NewRecorder())}
	}

	t.Run("rejects a token with no exp", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice"})
		if _, err := auth.ValidateToken(newCtx(tok)); err == nil {
			t.Error("expected error for a token with no exp")
		}
	})

	t.Run("AllowMissingExpiry opts back in", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: jwtTestSecret, AllowMissingExpiry: true}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice"})
		if _, err := auth.ValidateToken(newCtx(tok)); err != nil {
			t.Errorf("ValidateToken: %v", err)
		}
	})
}

// TestClaimsExpressionConcurrentUse guards the compiled-expression cache
// against the unsynchronised check-then-write it replaced: a single *JWTAuth
// is shared by every concurrent request, so the race sat inside an
// authorization decision. Meaningful under -race.
func TestClaimsExpressionConcurrentUse(t *testing.T) {
	auth := &JWTAuth{
		SigningSecret:    jwtTestSecret,
		Audience:         "api",
		ClaimsExpression: "Equals(`role`, `admin`) && OneOf(`tier`, `gold`, `silver`)",
	}
	tok := signHMACToken(t, jwt.MapClaims{
		"role":  "admin",
		"tier":  "gold",
		"aud":   "api",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"scope": "read",
	})

	app := New()
	app.Use(auth.Middleware)
	app.Get("/probe", func(c *Context) error { return c.String(http.StatusOK, "OK") })

	var wg sync.WaitGroup
	codes := make([]int, 32)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}(i)
	}
	wg.Wait()

	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i, code, http.StatusOK)
		}
	}
}

// signHS256 signs claims with HS256 under key, which may be empty.
func signHS256(t *testing.T, key []byte, claims jwt.MapClaims, kid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if kid != "" {
		tok.Header["kid"] = kid
	}
	signed, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}
	return signed
}

// TestEmptySigningSecretIsNotAKey guards against treating a zero-length
// secret, such as []byte(os.Getenv("UNSET")), as configured: anyone can
// compute an HMAC under an empty key, so every forged token verified.
func TestEmptySigningSecretIsNotAKey(t *testing.T) {
	t.Parallel()

	claims := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": "mallory", "exp": time.Now().Add(time.Hour).Unix()}
	}
	forged := signHS256(t, []byte{}, claims(), "")

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	tests := []struct {
		name  string
		auth  *JWTAuth
		token string
		want  int
	}{
		{"empty SigningSecret rejects a forged token",
			&JWTAuth{SigningSecret: []byte{}}, forged, http.StatusUnauthorized},
		{"empty SigningSecret and SecretKey reject a forged token",
			&JWTAuth{SigningSecret: []byte{}, SecretKey: []byte{}}, forged, http.StatusUnauthorized},
		{"empty SigningSecret falls back to SecretKey",
			&JWTAuth{SigningSecret: []byte{}, SecretKey: jwtTestSecret}, forged, http.StatusUnauthorized},
		{"SecretKey still verifies behind an empty SigningSecret",
			&JWTAuth{SigningSecret: []byte{}, SecretKey: jwtTestSecret}, signHMACToken(t, claims()), http.StatusOK},
		{"empty SigningSecret does not shadow RsaKey",
			&JWTAuth{SigningSecret: []byte{}, RsaKey: &rsaKey.PublicKey}, forged, http.StatusUnauthorized},
		{"RsaKey still verifies behind an empty SigningSecret",
			&JWTAuth{SigningSecret: []byte{}, RsaKey: &rsaKey.PublicKey}, signRSAToken(t, rsaKey, claims(), ""), http.StatusOK},
		{"nothing configured rejects a forged token",
			&JWTAuth{}, forged, http.StatusUnauthorized},
		{"nothing configured rejects a real token",
			&JWTAuth{}, signHMACToken(t, claims()), http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := serveJWT(tt.auth, tt.token); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("empty secrets resolve to no key", func(t *testing.T) {
		t.Parallel()
		auth := &JWTAuth{SigningSecret: []byte{}, SecretKey: []byte{}}
		if _, err := auth.resolveKeyFunc(); err == nil {
			t.Error("expected an error when only empty secrets are configured")
		}
	})
}

// TestValidateTokenMatchesMiddleware guards against ValidateToken drifting
// from the middleware. It verified only HMAC signatures, under SigningSecret —
// nil, and so an empty key, when the configuration used RsaKey or a JWKS — and
// ignored Audience, Issuer, the algorithm allow-list and ClaimsExpression.
func TestValidateTokenMatchesMiddleware(t *testing.T) {
	t.Parallel()

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	jwks := &Jwks{Keys: []Jwk{rsaJWK(t, "k1", &rsaKey.PublicKey)}}

	exp := time.Now().Add(time.Hour).Unix()
	claims := func(extra jwt.MapClaims) jwt.MapClaims {
		c := jwt.MapClaims{"sub": "alice", "exp": exp}
		for k, v := range extra {
			c[k] = v
		}
		return c
	}
	deny := func(*Context, jwt.Claims) error { return errors.New("denied") }

	tests := []struct {
		name   string
		auth   *JWTAuth
		token  string
		wantOK bool
	}{
		{"empty-key HMAC token against RsaKey",
			&JWTAuth{RsaKey: &rsaKey.PublicKey}, signHS256(t, []byte{}, claims(nil), ""), false},
		{"empty-key HMAC token against JwksFile",
			&JWTAuth{JwksFile: jwks}, signHS256(t, []byte{}, claims(nil), "k1"), false},
		{"Audience enforced",
			&JWTAuth{SigningSecret: jwtTestSecret, Audience: "api"}, signHMACToken(t, claims(jwt.MapClaims{"aud": "other"})), false},
		{"Issuer enforced",
			&JWTAuth{SigningSecret: jwtTestSecret, Issuer: "iss"}, signHMACToken(t, claims(jwt.MapClaims{"iss": "other"})), false},
		{"algorithm allow-list enforced",
			&JWTAuth{SigningSecret: jwtTestSecret, Algorithms: []string{"HS512"}}, signHMACToken(t, claims(nil)), false},
		{"ClaimsExpression enforced",
			&JWTAuth{SigningSecret: jwtTestSecret, ClaimsExpression: "Equals(`role`, `admin`)"},
			signHMACToken(t, claims(jwt.MapClaims{"role": "user"})), false},
		{"ValidateClaims enforced",
			&JWTAuth{SigningSecret: jwtTestSecret, ValidateClaims: deny}, signHMACToken(t, claims(nil)), false},
		{"RSA token against RsaKey",
			&JWTAuth{RsaKey: &rsaKey.PublicKey}, signRSAToken(t, rsaKey, claims(nil), ""), true},
		{"RSA token against JwksFile",
			&JWTAuth{JwksFile: jwks}, signRSAToken(t, rsaKey, claims(nil), "k1"), true},
		{"every check satisfied",
			&JWTAuth{SigningSecret: jwtTestSecret, Audience: "api", Issuer: "iss", ClaimsExpression: "Equals(`role`, `admin`)"},
			signHMACToken(t, claims(jwt.MapClaims{"aud": "api", "iss": "iss", "role": "admin"})), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, _ := NewTestContext(http.MethodGet, "/", nil)
			ctx.Request().Header.Set("Authorization", "Bearer "+tt.token)
			if _, err := tt.auth.ValidateToken(ctx); (err == nil) != tt.wantOK {
				t.Errorf("ValidateToken error = %v, want ok=%v", err, tt.wantOK)
			}
			if got := serveJWT(tt.auth, tt.token); (got == http.StatusOK) != tt.wantOK {
				t.Errorf("middleware status = %d, want ok=%v", got, tt.wantOK)
			}
		})
	}

	t.Run("cause is reachable through the returned error", func(t *testing.T) {
		t.Parallel()

		auth := &JWTAuth{SigningSecret: jwtTestSecret}
		tok := signHMACToken(t, jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(-time.Hour).Unix()})
		ctx, _ := NewTestContext(http.MethodGet, "/", nil)
		ctx.Request().Header.Set("Authorization", "Bearer "+tok)

		_, err := auth.ValidateToken(ctx)
		if !errors.Is(err, jwt.ErrTokenExpired) {
			t.Errorf("errors.Is(%v, jwt.ErrTokenExpired) = false, want true", err)
		}
	})
}

// TestGenerateJwtToken_RejectsEmptySecret guards against signing with an empty
// key, which produces a token anyone can forge.
func TestGenerateJwtToken_RejectsEmptySecret(t *testing.T) {
	for _, secret := range [][]byte{nil, {}} {
		if signed, err := GenerateJwtToken(secret, jwt.MapClaims{}, time.Hour); err == nil {
			t.Errorf("GenerateJwtToken(%q) = %q, nil; want an error", secret, signed)
		}
	}
	if _, err := GenerateJwtToken(jwtTestSecret, nil, time.Hour); err != nil {
		t.Errorf("GenerateJwtToken with nil claims: %v", err)
	}
}

// TestJWTMiddleware_NoKeyCallsOnUnauthorized guards the misconfiguration path,
// which answered 401 without calling the application's OnUnauthorized hook.
func TestJWTMiddleware_NoKeyCallsOnUnauthorized(t *testing.T) {
	auth := &JWTAuth{
		OnUnauthorized: func(c *Context) error {
			return c.Error(http.StatusTeapot, "hook")
		},
	}
	signed, err := GenerateJwtToken(jwtTestSecret, jwt.MapClaims{}, time.Hour)
	if err != nil {
		t.Fatalf("GenerateJwtToken: %v", err)
	}
	if got := serveJWT(auth, signed); got != http.StatusTeapot {
		t.Errorf("status = %d, want the OnUnauthorized hook's 418", got)
	}
}
