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
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jkaninda/okapi/okapitest"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	SigningSecret      = []byte("supersecret")
	bearerAuthSecurity = []map[string][]string{
		{
			"bearerAuth": {},
		},
	}
	basicAuthSecurity = map[string][]string{

		"basicAuth": {},
	}
)

const user = "user"

func TestJwtMiddleware(t *testing.T) {
	// Setup
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
			fPrint("Validating claims using custom function")
			method := c.Request().Method
			fPrint("Request method,", "method", method)
			if method != http.MethodGet && method != http.MethodPost {
				return fmt.Errorf("method %s is not allowed", method)
			}
			mapClaims, ok := claims.(jwt.MapClaims)
			if !ok {
				return errors.New("invalid claims type")
			}
			role, ok := mapClaims["user"].(map[string]interface{})["role"]
			if !ok || role != "admin" {
				if role != "" {
					return fmt.Errorf("role %s is not allowed to", role)
				}
				return fmt.Errorf("unauthorized role")
			}
			return nil
		},
		OnUnauthorized: func(c *Context) error {
			return c.ErrorUnauthorized("Unauthorized")
		},
	}
	jwtClaims := jwt.MapClaims{
		"sub": "12345",
		"iss": "okapi.example.com",
		"aud": "okapi.example.com",
		"user": map[string]string{
			"name":  "",
			"role":  "",
			"email": "admin@example.com",
		},
		"email_verified": true,
		"tags":           []string{},
		"exp":            time.Now().Add(2 * time.Hour).Unix(),
	}
	jwtClaimsNoAud := jwt.MapClaims{
		"sub": "12345",
		"iss": "okapi.example.com",
		"user": map[string]string{
			"name":  "",
			"role":  "",
			"email": "admin@example.com",
		},
		"email_verified": true,
		"tags":           []string{},
		"exp":            time.Now().Add(2 * time.Hour).Unix(),
	}
	// Generate Admin token with audience
	jwtClaims[user].(map[string]string)["role"] = "admin"
	jwtClaims[user].(map[string]string)["name"] = "Administrator"
	jwtClaims["tags"] = []string{"gold"}
	// Generate Admin token
	adminToken := mustGenerateToken(t, auth.SigningSecret, jwtClaims)

	// Generate User token
	jwtClaims[user].(map[string]string)["role"] = user
	jwtClaims[user].(map[string]string)["name"] = "User Name"
	jwtClaims[user].(map[string]string)["email"] = "user@example.com"
	jwtClaims["tags"] = []string{"vip"}
	token := mustGenerateToken(t, auth.SigningSecret, jwtClaims)

	// Generate a token without audience
	jwtClaimsNoAud[user].(map[string]string)["role"] = user
	jwtClaimsNoAud[user].(map[string]string)["name"] = "User Name"
	jwtClaimsNoAud[user].(map[string]string)["email"] = "user@example.com"
	jwtClaimsNoAud["tags"] = []string{"vip"}

	noAudToken := mustGenerateToken(t, auth.SigningSecret, jwtClaimsNoAud)

	// Setup server
	o := New(WithAccessLogDisabled())
	o.WithOpenAPIDocs(OpenAPI{
		Title:   "Okapi Web Framework Example",
		Version: "1.0.0",
		License: License{
			Name: "MIT",
		},
		SecuritySchemes: SecuritySchemes{
			{
				Name:   "basicAuth",
				Type:   "http",
				Scheme: "basic",
			},
			{
				Name:         "bearerAuth",
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
			},
			{
				Name: "OAuth2",
				Type: "oauth2",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
		},
	})

	// Create a new group for the main routes
	admin := o.Group("/admin", adminAuth.Middleware).WithSecurity(bearerAuthSecurity)
	// Use the JWT middleware for the main routes
	o.Use(auth.Middleware)
	o.Use(LoggerMiddleware)
	o.Get("/protected", whoAmIHandler)

	admin.Get("/protected", whoAmIHandler)

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
	client := okapitest.NewClient(t, "http://localhost:8080")

	client.GET("/protected").ExpectStatusUnauthorized()
	client.GET("/admin/protected").ExpectStatusUnauthorized()

	client.Headers["Authorization"] = "Bearer " + token

	// client.Headers = headers
	client.GET("/protected").ExpectStatusOK()
	client.GET("/admin/protected").ExpectStatusUnauthorized()

	client.Headers["Authorization"] = "Bearer " + adminToken
	client.GET("/protected").ExpectStatusOK()
	client.GET("/admin/protected").ExpectStatusOK()

	client.Headers["Authorization"] = "Bearer " + noAudToken
	client.GET("/protected").ExpectStatusUnauthorized()
	client.GET("/admin/protected").ExpectStatusUnauthorized()

}
func TestBasicAuth(t *testing.T) {
	username := "user"
	password := "password"
	auth := BasicAuth{Username: username, Password: password, ContextKey: "username"}

	app := Default()
	app.Use(auth.Middleware)

	app.Get("/protected", func(c *Context) error {
		user, exists := c.Get(auth.ContextKey)
		if !exists {
			return c.ErrorUnauthorized(M{"error": "Unauthorized"})
		}
		return c.OK(user)
	}).WithSecurity(basicAuthSecurity)

	// Start server in background
	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
			return
		}
	}()
	defer func(app *Okapi) {
		err := app.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(app)

	waitForServer()

	okapitest.GET(t, "http://localhost:8080/protected").ExpectStatusUnauthorized()
	okapitest.GET(t, "http://localhost:8080/protected").SetBasicAuth(username, password).ExpectStatusOK()

}
func TestStdMiddleware(t *testing.T) {
	o := Default()
	o.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			w.Header().Add("Version", "v1.0")
			handler.ServeHTTP(w, r)
		})

	})
	o.Get("/", func(c *Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	}).Use(helloMiddleware)
	api := o.Group("/api")
	api.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP Group middleware function")
			w.Header().Add("Group", "api")

			handler.ServeHTTP(w, r)
		})
	})
	api.Get("/hello", func(c *Context) error {
		c.Logger().Info("Hello World")
		return c.JSON(http.StatusOK, M{"hello": "world"})
	}, UseMiddleware(helloMiddleware),
	).Use(helloMiddleware)
	api.Get("/", func(c *Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	})
	o.Handle("GET", "hello", func(c *Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	})
	o.HandleStd("POST", "hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("hello world"))
		if err != nil {
			return
		}
	})
	apiV1 := api.Group("v1")
	apiV1.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s := r.Header.Get("Authorization"); s == "" || !strings.Contains(s, "Bearer") {
				w.WriteHeader(http.StatusUnauthorized)
				_, err := w.Write([]byte("Unauthorized"))
				if err != nil {
					return
				}
				return
			}
			handler.ServeHTTP(w, r)

		})

	})
	apiV1.HandleStd("POST", "protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("hello world"))
		if err != nil {
			return
		}
	})

	slog.Info("Route count", "count", len(o.Routes()))
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
			return
		}
	}()
	defer func(o *Okapi) {
		err := o.Stop()
		if err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}
	}(o)

	waitForServer()
	okapitest.GET(t, "http://localhost:8080/").ExpectStatusOK().ExpectHeader("Version", "v1.0")
	okapitest.GET(t, "http://localhost:8080/api").ExpectStatusOK().ExpectHeader("Group", "api")
	okapitest.GET(t, "http://localhost:8080/api/hello").ExpectStatusOK()
	okapitest.GET(t, "http://localhost:8080/hello").ExpectStatusOK()
	okapitest.POST(t, "http://localhost:8080/hello").ExpectStatusCreated().ExpectBody("hello world")
	okapitest.POST(t, "http://localhost:8080/api/v1/protected").ExpectStatusUnauthorized().ExpectBody("Unauthorized")
	okapitest.POST(t, "http://localhost:8080/api/v1/protected").SetBearerAuth("Token").ExpectStatusCreated().ExpectBody("hello world")

}

func TestStdMiddleware_PreservesOkapiContextValues(t *testing.T) {
	type ctxKey string

	const requestKey ctxKey = "trace"
	server := NewTestServer(t)
	server.DisableAccessLog()

	injectRequestContext := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), requestKey, "trace-123"))
			next.ServeHTTP(w, r)
		})
	}

	server.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("root-value", "root-ok")
			return next(c)
		}
	})
	server.UseMiddleware(injectRequestContext)
	server.Get("/root", func(c *Context) error {
		if c.GetString("root-value") != "root-ok" {
			return c.String(http.StatusInternalServerError, "missing-root-store")
		}
		if got, _ := c.Request().Context().Value(requestKey).(string); got != "trace-123" {
			return c.String(http.StatusInternalServerError, "missing-root-request-context")
		}
		return c.String(http.StatusOK, "ok")
	})

	group := server.Group("/api")
	group.Use(func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			c.Set("group-value", "group-ok")
			return next(c)
		}
	})
	group.UseMiddleware(injectRequestContext)
	group.Get("/ctx", func(c *Context) error {
		if c.GetString("group-value") != "group-ok" {
			return c.String(http.StatusInternalServerError, "missing-group-store")
		}
		if got, _ := c.Request().Context().Value(requestKey).(string); got != "trace-123" {
			return c.String(http.StatusInternalServerError, "missing-group-request-context")
		}
		return c.String(http.StatusOK, "ok")
	})

	okapitest.GET(t, server.BaseURL+"/root").ExpectStatusOK().ExpectBody("ok")
	okapitest.GET(t, server.BaseURL+"/api/ctx").ExpectStatusOK().ExpectBody("ok")
}

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
	slog.Info("Who am I am ", "email", email, "role", c.GetString("role"), "name", c.GetString("name"))
	// Respond with the current user information
	return c.JSON(http.StatusOK, M{
		"email": email,
		"role":  c.GetString("role"),
		"name":  c.GetString("name"),
	},
	)
}

func helloMiddleware(next HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		slog.Info("Hello Okapi Route middleware function")
		return next(c)
	}

}
