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
	"github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestJwtMiddleware(t *testing.T) {
	// Setup
	auth := JWTAuth{
		SecretKey:   []byte("supersecret"),
		TokenLookup: "header:Authorization",
		ContextKey:  "user",
	}

	// Generate token
	token := mustGenerateToken(t, auth.SecretKey)

	// Setup server
	o := Default()
	o.Use(auth.Middleware)
	o.Get("/protected", func(c Context) error {
		user, exists := c.Get(auth.ContextKey)
		if !exists {
			return c.JSON(http.StatusUnauthorized, M{"error": "Unauthorized"})
		}
		return c.JSON(http.StatusOK, M{"user": user})
	})

	go func() {
		if err := o.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("Failed to start server: %v", err)
		}
	}()
	defer o.Stop()

	// Wait briefly for the server to start
	time.Sleep(100 * time.Millisecond)

	// Make request
	resp := mustDoRequest(t, "http://localhost:8080/protected", token)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Error("Failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}
}

func mustGenerateToken(t *testing.T, secret []byte) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  "12345",
		"role": "admin",
		"exp":  time.Now().Add(2 * time.Hour).Unix(),
	}
	token, err := GenerateJwtToken(secret, claims, 2*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}
	return token
}

func mustDoRequest(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	return resp
}
