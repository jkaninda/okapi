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
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

var SigningSecret = []byte("supersecret")

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
		ContextKey:       "user",
		ClaimsExpression: "Equals(`email_verified`, `true`) && Equals(`user.role`, `admin`) && Contains(`tags`,`gold`)",
		ForwardClaims: map[string]string{
			"email": "user.email",
			"role":  "user.role",
			"name":  "user.name",
		},
		ValidateRole: func(claims jwt.Claims) error {
			fPrint("Validating role using custom function")
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
	// Create a new group for the main routes
	admin := o.Group("/admin", adminAuth.Middleware)
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
	defer o.Stop()

	waitForServer()
	assertStatus(t, "GET", "http://localhost:8080/protected", nil, nil, "", http.StatusForbidden)
	assertStatus(t, "GET", "http://localhost:8080/admin/protected", nil, nil, "", http.StatusForbidden)

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	assertStatus(t, "GET", "http://localhost:8080/protected", headers, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/admin/protected", headers, nil, "", http.StatusUnauthorized)

	headers["Authorization"] = "Bearer " + adminToken
	assertStatus(t, "GET", "http://localhost:8080/admin/protected", headers, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/protected", headers, nil, "", http.StatusOK)

	headers["Authorization"] = "Bearer " + noAudToken
	assertStatus(t, "GET", "http://localhost:8080/protected", headers, nil, "", http.StatusUnauthorized)
	assertStatus(t, "GET", "http://localhost:8080/admin/protected", headers, nil, "", http.StatusUnauthorized)

}
func TestBasicAuth(t *testing.T) {
	username := "user"
	password := "password"
	auth := BasicAuth{Username: username, Password: password, ContextKey: "username"}

	app := Default()
	app.Use(auth.Middleware)

	app.Get("/protected", func(c Context) error {
		user, exists := c.Get(auth.ContextKey)
		if !exists {
			return c.ErrorForbidden(M{"error": "Unauthorized"})
		}
		return c.OK(user)
	})

	// Start server in background
	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
			return
		}
	}()
	defer app.Stop()

	waitForServer()

	credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	headers := map[string]string{
		"Authorization": "Basic " + credentials,
	}

	assertStatus(t, "GET", "http://localhost:8080/protected", nil, nil, "", http.StatusUnauthorized)
	assertStatus(t, "GET", "http://localhost:8080/protected", headers, nil, "", http.StatusOK)
}
func TestStdMiddleware(t *testing.T) {
	o := Default()
	o.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP middleware function")
			handler.ServeHTTP(w, r)
		})

	})
	o.Get("/", func(c Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	})
	api := o.Group("/api")
	api.UseMiddleware(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("Hello Go standard HTTP Group middleware function")

			handler.ServeHTTP(w, r)
		})
	})
	api.Get("/", func(c Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	})
	o.Handle("GET", "hello", func(c Context) error {
		return c.JSON(http.StatusOK, M{"hello": "world"})
	})
	o.HandleStd("POST", "hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte("hello world"))
		if err != nil {
			return
		}
	})

	slog.Info("Route count", "count", len(o.Routes()))
	slog.Info("Middleware count", "count", len(o.Middlewares()))
	// Start server in background
	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Server failed to start: %v", err)
			return
		}
	}()
	defer o.Stop()

	waitForServer()

	assertStatus(t, "GET", "http://localhost:8080/", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/api/", nil, nil, "", http.StatusOK)
	assertStatus(t, "GET", "http://localhost:8080/hello", nil, nil, "", http.StatusOK)
	assertStatus(t, "POST", "http://localhost:8080/hello", nil, nil, "", http.StatusCreated)
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

func whoAmIHandler(c Context) error {
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
