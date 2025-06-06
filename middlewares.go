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
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	goutils "github.com/jkaninda/go-utils"
	"io"
	"net/http"
	"strings"
	"time"
)

// BasicAuthMiddleware is a middleware that adds basic authentication to the Request context.

type (
	// BasicAuthMiddleware provides basic authentication for routes.
	BasicAuthMiddleware struct {
		Username string
		Password string
		Realm    string
	}
	// Logger is a middleware that logs request details such as method, URL,
	// client IP, status, duration, referer, and user agent.
	Logger struct {
	}
	// BodyLimit is a middleware that limits the size of the request body.
	BodyLimit struct {
		MaxBytes int64
	}
	// JWTAuth holds the config for JWT verification middleware.
	JWTAuth struct {
		SecretKey   []byte
		TokenLookup string // e.g., "header:Authorization", "query:token", "cookie:jwt"
		ContextKey  string // where to store claims in context, e.g. "user"
	}
)

// LoggerMiddleware is a middleware that logs request details like method, URL, client IP,
// status, duration, referer, and user agent.
func LoggerMiddleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		if c.IsWebSocketUpgrade() || c.IsSSE() {
			// Skip logging for WebSocket upgrades or Server-Sent Events
			return next(c)
		}
		startTime := time.Now()
		err := next(c)
		duration := goutils.FormatDuration(time.Since(startTime), 2)
		c.okapi.logger.Info("[okapi]",
			"method", c.Request.Method,
			"url", c.Request.URL.Path,
			"client_ip", c.RealIP(),
			"status", c.Response.Status(),
			"duration", duration,
			"referer", c.Request.Referer(),
			"user_agent", c.Request.UserAgent(),
		)
		return err
	}
}

// Middleware is a basic authentication middleware that checks the request's Basic Auth credentials.
func (b *BasicAuthMiddleware) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		username, password, ok := c.Request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(username), []byte(b.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(b.Password)) != 1 {

			realm := b.Realm
			if realm == "" {
				realm = FrameworkName
			}
			c.Response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			return c.String(http.StatusUnauthorized, "Unauthorized")
		}

		c.Set("username", username)
		return next(c)
	}
}

// Middleware is a middleware that limits the size of the request body to prevent excessive memory usage.
func (b BodyLimit) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		const errReadBody = "Failed to read request body"
		const errTooLarge = "Request body too large"

		// LimitReader prevents reading more than MaxBytes+1
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, b.MaxBytes+1))
		if err != nil {
			return c.String(http.StatusInternalServerError, errReadBody)
		}
		if int64(len(body)) > b.MaxBytes {
			return c.String(http.StatusRequestEntityTooLarge, errTooLarge)
		}

		// Reset request body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		return next(c)
	}
}

// Middleware validates JWT tokens from the configured source
func (j JWTAuth) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		tokenStr, err := j.extractToken(c)
		if err != nil {
			return c.String(http.StatusUnauthorized, "Missing or invalid token")
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return j.SecretKey, nil
		})

		if err != nil || !token.Valid {
			return c.String(http.StatusUnauthorized, "Invalid or expired token")
		}

		if j.ContextKey != "" && token.Claims != nil {
			c.Set(j.ContextKey, token.Claims)
		}

		return next(c)
	}
}

// ValidateToken checks the JWT token and returns the claims if valid
func (j JWTAuth) ValidateToken(c Context) (jwt.MapClaims, error) {
	tokenStr, err := j.extractToken(c)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return j.SecretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, errors.New("invalid claims type")
}

// extractToken pulls the token from header, query or cookie
func (j JWTAuth) extractToken(c Context) (string, error) {
	parts := strings.Split(j.TokenLookup, ":")
	if len(parts) != 2 {
		return "", errors.New("invalid token lookup config")
	}

	source, name := parts[0], parts[1]
	switch source {
	case "header":
		auth := c.Request.Header.Get(name)
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer "), nil
		}
		return auth, nil
	case "query":
		return c.Query(name), nil
	case "cookie":
		cookie, err := c.Request.Cookie(name)
		if err != nil {
			return "", err
		}
		return cookie.Value, nil
	default:
		return "", errors.New("unsupported token source")
	}
}

// GenerateJwtToken generates a JWT with custom claims and expiry
func GenerateJwtToken(secret []byte, claims jwt.MapClaims, ttl time.Duration) (string, error) {
	claims["exp"] = time.Now().Add(ttl).Unix()
	claims["iat"] = time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
