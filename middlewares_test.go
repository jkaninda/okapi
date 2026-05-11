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
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jkaninda/okapi/okapitest"
)

// SigningSecret is shared by tests that exercise JWT/HMAC signing behaviour.
var SigningSecret = []byte("supersecret")

const userKey = "user"

// makeClaims builds a jwt.MapClaims from a base set, mutating role/name/email/tags.
func makeClaims(role, name, email string, tags []string, withAud bool) jwt.MapClaims {
	c := jwt.MapClaims{
		"sub": "12345",
		"iss": "okapi.example.com",
		"user": map[string]string{
			"role":  role,
			"name":  name,
			"email": email,
		},
		"email_verified": true,
		"tags":           tags,
		"exp":            time.Now().Add(2 * time.Hour).Unix(),
	}
	if withAud {
		c["aud"] = "okapi.example.com"
	}
	return c
}

// -----------------------------------------------------------------------------
// JWT middleware
// -----------------------------------------------------------------------------

func TestJWTMiddleware(t *testing.T) {
	auth := JWTAuth{
		Audience:      "okapi.example.com",
		Issuer:        "okapi.example.com",
		SigningSecret: SigningSecret,
		TokenLookup:   "header:Authorization",
		ClaimsExpression: "Equals(`email_verified`, `true`) " +
			"&& OneOf(`user.role`, `admin`, `user`) " +
			"&& Contains(`tags`, `vip`, `premium`, `gold`)",
		ForwardClaims: map[string]string{
			"email": "user.email",
			"role":  "user.role",
			"name":  "user.name",
		},
	}
	adminAuth := JWTAuth{
		Audience:         "okapi.example.com",
		SigningSecret:    SigningSecret,
		TokenLookup:      "header:Authorization",
		ContextKey:       "claims",
		ClaimsExpression: "Equals(`email_verified`, `true`) && Equals(`user.role`, `admin`) && Contains(`tags`,`gold`)",
		ForwardClaims: map[string]string{
			"email": "user.email",
			"role":  "user.role",
			"name":  "user.name",
		},
		ValidateClaims: func(c *Context, claims jwt.Claims) error {
			method := c.Request().Method
			if method != http.MethodGet && method != http.MethodPost {
				return fmt.Errorf("method %s is not allowed", method)
			}
			mapClaims, ok := claims.(jwt.MapClaims)
			if !ok {
				return errors.New("invalid claims type")
			}
			role, ok := mapClaims[userKey].(map[string]any)["role"]
			if !ok || role != "admin" {
				return fmt.Errorf("role %v not allowed", role)
			}
			return nil
		},
		OnUnauthorized: func(c *Context) error {
			return c.ErrorUnauthorized("Unauthorized")
		},
	}

	adminToken := mustGenerateToken(t, SigningSecret, makeClaims("admin", "Administrator", "admin@example.com", []string{"gold"}, true))
	userToken := mustGenerateToken(t, SigningSecret, makeClaims("user", "User Name", "user@example.com", []string{"vip"}, true))
	noAudToken := mustGenerateToken(t, SigningSecret, makeClaims("user", "User Name", "user@example.com", []string{"vip"}, false))

	ts := NewTestServer(t)
	admin := ts.Group("/admin", adminAuth.Middleware).WithSecurity(bearerAuthSecurity)
	ts.Use(auth.Middleware)
	ts.Get("/protected", whoAmIHandler)
	admin.Get("/protected", whoAmIHandler)

	tests := []struct {
		name   string
		path   string
		token  string
		status int
	}{
		{"protected without token", "/protected", "", http.StatusUnauthorized},
		{"admin without token", "/admin/protected", "", http.StatusUnauthorized},
		{"protected with user token", "/protected", userToken, http.StatusOK},
		{"admin denied to user", "/admin/protected", userToken, http.StatusUnauthorized},
		{"protected with admin token", "/protected", adminToken, http.StatusOK},
		{"admin allowed for admin", "/admin/protected", adminToken, http.StatusOK},
		{"protected rejects token without aud", "/protected", noAudToken, http.StatusUnauthorized},
		{"admin rejects token without aud", "/admin/protected", noAudToken, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := okapitest.GET(t, ts.BaseURL+tt.path)
			if tt.token != "" {
				req.Header("Authorization", "Bearer "+tt.token)
			}
			req.ExpectStatus(tt.status)
		})
	}
}

func TestJWTAuth_OnUnauthorizedHook(t *testing.T) {
	called := false
	auth := JWTAuth{
		SigningSecret: SigningSecret,
		OnUnauthorized: func(c *Context) error {
			called = true
			return c.ErrorUnauthorized("custom unauthorized")
		},
	}

	ts := NewTestServer(t)
	ts.Use(auth.Middleware)
	ts.Get("/p", func(c *Context) error { return c.OK("ok") })

	okapitest.GET(t, ts.BaseURL+"/p").
		ExpectStatusUnauthorized().
		ExpectBodyContains("custom unauthorized")

	if !called {
		t.Error("OnUnauthorized hook was not invoked")
	}
}

func TestJWTAuth_ContextKeyStoresClaims(t *testing.T) {
	auth := JWTAuth{
		SigningSecret: SigningSecret,
		Audience:      "okapi.example.com",
		ContextKey:    "claims",
	}

	ts := NewTestServer(t)
	ts.Use(auth.Middleware)
	ts.Get("/p", func(c *Context) error {
		v, ok := c.Get("claims")
		if !ok {
			return c.ErrorInternalServerError("claims missing")
		}
		mc, ok := v.(jwt.MapClaims)
		if !ok {
			return c.ErrorInternalServerError("wrong type")
		}
		return c.JSON(http.StatusOK, map[string]any{"sub": mc["sub"]})
	})

	tok := mustGenerateToken(t, SigningSecret, jwt.MapClaims{
		"sub": "alice",
		"aud": "okapi.example.com",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	okapitest.GET(t, ts.BaseURL+"/p").
		Header("Authorization", "Bearer "+tok).
		ExpectStatusOK().
		ExpectBodyContains(`"sub":"alice"`)
}

func TestJWTResolveKeyFunc_NilJwksFile(t *testing.T) {
	t.Parallel()

	auth := &JWTAuth{}
	keyFunc, err := auth.resolveKeyFunc()
	if err == nil {
		t.Fatal("expected error when no JWT verifier is configured")
	}
	if keyFunc != nil {
		t.Fatal("expected nil keyFunc when no verifier is configured")
	}
}

func TestJWTMiddleware_DoesNotMutateGlobalAlgorithms(t *testing.T) {
	original := append([]string(nil), jwtAlgo...)
	t.Cleanup(func() { jwtAlgo = original })

	auth := &JWTAuth{
		SigningSecret: SigningSecret,
		Algo:          "HS256",
		Audience:      "okapi.example.com",
		Issuer:        "okapi.example.com",
	}
	tok := mustGenerateToken(t, SigningSecret, jwt.MapClaims{
		"sub": "12345",
		"iss": "okapi.example.com",
		"aud": "okapi.example.com",
	})

	ctx, _ := NewTestContext(http.MethodGet, "/protected", nil)
	ctx.okapi = New(WithAccessLogDisabled())
	ctx.Request().Header.Set("Authorization", "Bearer "+tok)

	called := false
	ctx.handlers = []HandlerFunc{
		auth.Middleware,
		func(c *Context) error {
			called = true
			return nil
		},
	}
	ctx.index = -1
	if err := ctx.Next(); err != nil {
		t.Fatalf("middleware returned error: %v", err)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
	if !reflect.DeepEqual(jwtAlgo, original) {
		t.Fatalf("jwtAlgo was mutated: got %v want %v", jwtAlgo, original)
	}
}

// -----------------------------------------------------------------------------
// BasicAuth middleware
// -----------------------------------------------------------------------------

func TestBasicAuth(t *testing.T) {
	const username, password = "user", "password"
	auth := BasicAuth{Username: username, Password: password, ContextKey: "username"}

	ts := NewTestServer(t)
	ts.Use(auth.Middleware)
	ts.Get("/protected", func(c *Context) error {
		v, ok := c.Get(auth.ContextKey)
		if !ok {
			return c.ErrorUnauthorized(M{"error": "Unauthorized"})
		}
		return c.OK(v)
	}).WithSecurity(basicAuthSecurity)

	t.Run("missing credentials", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/protected").
			ExpectStatusUnauthorized().
			ExpectHeaderContains("WWW-Authenticate", "Basic realm=")
	})

	t.Run("wrong credentials", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/protected").
			SetBasicAuth(username, "wrong").
			ExpectStatusUnauthorized()
	})

	t.Run("valid credentials", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/protected").
			SetBasicAuth(username, password).
			ExpectStatusOK().
			ExpectBodyContains(username)
	})
}

func TestBasicAuth_DefaultRealm(t *testing.T) {
	auth := BasicAuth{Username: "u", Password: "p"}

	ts := NewTestServer(t)
	ts.Use(auth.Middleware)
	ts.Get("/p", func(c *Context) error { return c.OK("ok") })

	resp, _ := okapitest.GET(t, ts.BaseURL+"/p").
		ExpectStatusUnauthorized().
		Execute()

	if h := resp.Header.Get("WWW-Authenticate"); !strings.Contains(h, `realm=`) {
		t.Errorf("WWW-Authenticate = %q, want realm=...", h)
	}
}

// Deprecated BasicAuthMiddleware delegates to BasicAuth — verify behaviour.
func TestBasicAuthMiddleware_DeprecatedDelegate(t *testing.T) {
	auth := BasicAuthMiddleware{Username: "u", Password: "p", ContextKey: "user"}

	ts := NewTestServer(t)
	ts.Use(auth.Middleware)
	ts.Get("/p", func(c *Context) error {
		v, _ := c.Get("user")
		return c.OK(v)
	})

	okapitest.GET(t, ts.BaseURL+"/p").
		SetBasicAuth("u", "p").
		ExpectStatusOK().
		ExpectBodyContains("u")
}

// -----------------------------------------------------------------------------
// BodyLimit middleware
// -----------------------------------------------------------------------------

func TestBodyLimit(t *testing.T) {
	limit := BodyLimit{MaxBytes: 10}

	ts := NewTestServer(t)
	ts.Use(limit.Middleware)
	ts.Post("/echo", func(c *Context) error {
		return c.OK("ok")
	})

	t.Run("under limit", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/echo").
			Body(strings.NewReader("hello")).
			ExpectStatusOK()
	})

	t.Run("at limit", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/echo").
			Body(strings.NewReader("0123456789")).
			ExpectStatusOK()
	})

	t.Run("over limit", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/echo").
			Body(strings.NewReader("0123456789-too-many")).
			ExpectStatus(http.StatusRequestEntityTooLarge)
	})
}

// -----------------------------------------------------------------------------
// RequestID middleware
// -----------------------------------------------------------------------------

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	ts := NewTestServer(t)
	ts.Use(RequestID())
	ts.Get("/p", func(c *Context) error {
		return c.OK(M{"id": c.GetString("request_id")})
	})

	resp, body := okapitest.GET(t, ts.BaseURL+"/p").
		ExpectStatusOK().
		Execute()

	id := resp.Header.Get(requestIDHeader)
	if id == "" {
		t.Fatal("expected generated X-Request-ID header")
	}
	if !strings.Contains(string(body), id) {
		t.Errorf("body should echo request id %q, got %s", id, body)
	}
}

func TestRequestID_PropagatesIncoming(t *testing.T) {
	ts := NewTestServer(t)
	ts.Use(RequestID())
	ts.Get("/p", func(c *Context) error {
		return c.OK(M{"id": c.GetString("request_id")})
	})

	const incoming = "incoming-id-123"
	resp, body := okapitest.GET(t, ts.BaseURL+"/p").
		Header(requestIDHeader, incoming).
		ExpectStatusOK().
		Execute()

	if got := resp.Header.Get(requestIDHeader); got != incoming {
		t.Errorf("response header = %q, want %q", got, incoming)
	}
	if !strings.Contains(string(body), incoming) {
		t.Errorf("body should echo request id %q, got %s", incoming, body)
	}
}

// -----------------------------------------------------------------------------
// LoggerMiddleware skip paths
// -----------------------------------------------------------------------------

func TestLoggerMiddleware_AlwaysCallsNext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(c *Context)
		expect int
	}{
		{
			name:   "regular request",
			setup:  func(c *Context) {},
			expect: http.StatusOK,
		},
		{
			name: "websocket upgrade is logged-skipped but next still runs",
			setup: func(c *Context) {
				c.Request().Header.Set("Upgrade", "websocket")
			},
			expect: http.StatusOK,
		},
		{
			name: "SSE request",
			setup: func(c *Context) {
				c.Request().Header.Set("Accept", "text/event-stream")
			},
			expect: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, rec := NewTestContext(http.MethodGet, "/", nil)
			ctx.okapi = New(WithAccessLogDisabled())
			tt.setup(ctx)

			called := false
			ctx.handlers = []HandlerFunc{
				LoggerMiddleware,
				func(c *Context) error {
					called = true
					return c.Text(tt.expect, "ok")
				},
			}
			ctx.index = -1
			if err := ctx.Next(); err != nil {
				t.Fatalf("middleware error: %v", err)
			}
			if !called {
				t.Fatal("next handler was not called")
			}
			if rec.Code != tt.expect {
				t.Errorf("status = %d, want %d", rec.Code, tt.expect)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Std middleware: net/http handlers via UseMiddleware / HandleStd
// -----------------------------------------------------------------------------

func TestStdMiddleware(t *testing.T) {
	ts := NewTestServer(t)
	ts.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Version", "v1.0")
			handler.ServeHTTP(w, r)
		})
	})
	ts.Use(RequestID())
	ts.Get("/", func(c *Context) error { return c.JSON(http.StatusOK, M{"hello": "world"}) }).Use(helloMiddleware)

	api := ts.Group("/api")
	api.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Group", "api")
			handler.ServeHTTP(w, r)
		})
	})
	api.Get("/hello", func(c *Context) error { return c.JSON(http.StatusOK, M{"hello": "world"}) }, UseMiddleware(helloMiddleware)).Use(helloMiddleware)
	api.Get("/", func(c *Context) error { return c.JSON(http.StatusOK, M{"hello": "world"}) })

	ts.Handle("GET", "hello", func(c *Context) error { return c.JSON(http.StatusOK, M{"hello": "world"}) })
	ts.HandleStd("POST", "hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello world"))
	})

	apiV1 := api.Group("v1")
	apiV1.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s := r.Header.Get("Authorization"); s == "" || !strings.Contains(s, "Bearer") {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("Unauthorized"))
				return
			}
			handler.ServeHTTP(w, r)
		})
	})
	apiV1.HandleStd("POST", "protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello world"))
	})

	t.Run("root carries Version header from std middleware", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/").
			ExpectStatusOK().
			ExpectHeader("Version", "v1.0")
	})
	t.Run("group middleware adds Group header and request id propagates", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/api").
			ExpectStatusOK().
			ExpectHeader("Group", "api").
			ExpectHeaderContains(requestIDHeader, "")
	})
	t.Run("nested group route", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/api/hello").ExpectStatusOK()
	})
	t.Run("Handle vs HandleStd both reachable", func(t *testing.T) {
		okapitest.GET(t, ts.BaseURL+"/hello").ExpectStatusOK()
		okapitest.POST(t, ts.BaseURL+"/hello").
			ExpectStatusCreated().
			ExpectBody("hello world")
	})
	t.Run("std middleware enforces auth", func(t *testing.T) {
		okapitest.POST(t, ts.BaseURL+"/api/v1/protected").
			ExpectStatusUnauthorized().
			ExpectBody("Unauthorized")
		okapitest.POST(t, ts.BaseURL+"/api/v1/protected").
			SetBearerAuth("Token").
			ExpectStatusCreated().
			ExpectBody("hello world")
	})
}

// -----------------------------------------------------------------------------
// Shared helpers
// -----------------------------------------------------------------------------

var (
	bearerAuthSecurity = []map[string][]string{{"bearerAuth": {}}}
	basicAuthSecurity  = map[string][]string{"basicAuth": {}}
)

func mustGenerateToken(t *testing.T, secret []byte, claims jwt.MapClaims) string {
	t.Helper()
	token, err := GenerateJwtToken(secret, claims, 2*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}
	return token
}

func whoAmIHandler(c *Context) error {
	email := c.GetString("email")
	if email == "" {
		return c.AbortUnauthorized("Unauthorized", fmt.Errorf("user not authenticated"))
	}
	return c.JSON(http.StatusOK, M{
		"email": email,
		"role":  c.GetString("role"),
		"name":  c.GetString("name"),
	})
}

func helloMiddleware(c *Context) error { return c.Next() }
