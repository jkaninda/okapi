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
	"crypto/rsa"
	"crypto/subtle"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	goutils "github.com/jkaninda/go-utils"
	"io"
	"net/http"
	"time"
)

// BasicAuthMiddleware is a middleware that adds basic authentication to the request context.

type (
	// BasicAuth provides basic authentication for routes.
	BasicAuth struct {
		Username   string
		Password   string
		Realm      string
		ContextKey string // where to store the username e.g. "user", default(username)

	}
	// BasicAuthMiddleware provides basic authentication for routes
	//
	// deprecated, use BasicAuth
	BasicAuthMiddleware BasicAuth
	// Logger is a middleware that logs request details such as method, URL,
	// client IP, status, duration, referer, and user agent.
	Logger struct {
	}
	// BodyLimit is a middleware that limits the size of the request body.
	BodyLimit struct {
		MaxBytes int64
	}
	// JWTAuth is a configuration struct for JWT-based authentication middleware.
	//
	// You must configure at least one token verification mechanism:
	// - SigningSecret: for HMAC algorithms
	// - RsaKey: for RSA algorithms (e.g. RS256)
	// - JwksUrl: to fetch public keys dynamically from a JWKS endpoint
	// - JwksFile: to load static JWKS from a file or base64 string, use okapi.LoadJWKSFromFile()
	//
	// Fields:
	// JWTAuth holds configuration for JWT-based authentication.
	JWTAuth struct {
		// SecretKey is a legacy secret key used for HMAC algorithms (e.g., HS256).
		// Deprecated: Use SigningSecret instead.
		SecretKey []byte

		// SigningSecret is the key used for signing/validating tokens when using symmetric algorithms like HS256.
		SigningSecret []byte

		// JwksFile provides a static JWKS (JSON Web Key Set), either from a file or base64-encoded string.
		// Use okapi.LoadJWKSFromFile() to load the JWKS from a file.
		// Optional.
		JwksFile *Jwks

		// JwksUrl specifies a remote JWKS endpoint URL for key discovery.
		// Optional.
		JwksUrl string

		// Audience is the expected "aud" (audience) claim in the token.
		// Optional.
		Audience string

		// Issuer is the expected "iss" (issuer) claim in the token.
		// Optional.
		Issuer string

		// RsaKey is a public RSA key used to verify tokens signed with RS256.
		// Optional.
		RsaKey *rsa.PublicKey

		// Algo specifies the expected signing algorithm (e.g., "RS256", "HS256").
		// Optional.
		Algo string

		// TokenLookup defines how and where to extract the token from the request.
		// Supported formats include:
		//   - "header:Authorization" (default)
		//   - "query:token"
		//   - "cookie:jwt"
		TokenLookup string
		// ContextKey is the key used to store the full validated JWT claims in the request context.
		//
		// Use this when you need access to the entire set of claims for advanced processing or custom logic
		// within your handler or middleware.
		//
		// If you only need specific claim values (e.g., "user.email", "user.id"), consider using ForwardClaims instead.
		//
		// Example:
		//   ContextKey: "user"
		ContextKey string
		// ForwardClaims maps context keys to JWT claim paths (supports dot notation for nested fields).
		// This extracts selected claims and stores them in the request context under the specified keys.
		//
		// Use this when you want to expose only specific claims to handlers or middleware, without
		// needing access to the entire token.
		//
		// Example:
		//   ForwardClaims: map[string]string{
		//     "email": "user.email",
		//     "uid":   "user.id",
		//   }
		ForwardClaims map[string]string
		// ClaimsExpression defines a custom expression to validate JWT claims.
		// Useful for enforcing advanced conditions on claims such as role, scope, or custom fields.
		//
		// Supported functions:
		//   - Equals(field, value)
		//   - Prefix(field, prefix)
		//   - Contains(field, val1, val2, ...)
		//   - OneOf(field, val1, val2, ...)
		//
		// Logical Operators:
		//   - !   — NOT
		//   - &&  — AND (evaluated before OR)
		//   - ||  — OR  (evaluated after AND)
		//
		// These operators allow you to combine multiple expressions to create complex validation logic.
		// Example:
		//   jwtAuth.ClaimsExpression = "Equals(`email_verified`, `true`) && OneOf(`user.role`, `admin`, `owner`) && Contains(`tags`, `vip`, `premium`)"
		//
		// In the above:
		//   - The expression ensures the user is verified AND either has an admin/owner role,
		//     OR belongs to a premium tag group.
		ClaimsExpression string
		// parsedExpression holds the compiled version of ClaimsExpression.
		parsedExpression Expression
		// ValidateClaims is an optional custom validation function for processing JWT claims.
		// This provides full control over claim validation logic and can be used alongside or
		// instead of ClaimsExpression.
		//
		// Return an error to reject the request.
		//
		// Example:
		//   ValidateClaims: func(claims jwt.Claims) error {
		//     mapClaims, ok := claims.(jwt.MapClaims)
		//     if !ok {
		//       return errors.New("invalid claims type")
		//     }
		//     if emailVerified, _ := mapClaims["email_verified"].(bool); !emailVerified {
		//       return errors.New("email not verified")
		//     }
		//     if role, _ := mapClaims["role"].(string); role != "admin" {
		//       return errors.New("unauthorized role")
		//     }
		//     return nil
		//   }
		ValidateClaims func(claims jwt.Claims) error

		// Deprecated: Use ValidateClaims instead.
		//
		// ValidateRole was previously used for role-based access control, but has been
		// replaced by the more general ValidateClaims function which allows for flexible
		// validation of any JWT claims.
		ValidateRole func(claims jwt.Claims) error
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
		status := c.response.StatusCode()
		duration := goutils.FormatDuration(time.Since(startTime), 2)

		logger := c.okapi.logger
		args := []any{
			"method", c.request.Method,
			"url", c.request.URL.Path,
			"ip", c.RealIP(),
			"host", c.request.Host,
			"status", status,
			"duration", duration,
			"referer", c.request.Referer(),
			"user_agent", c.request.UserAgent(),
		}
		switch {
		case status >= 500:
			logger.Error("[okapi]", args...)
		case status >= 400:
			logger.Warn("[okapi]", args...)
		default:
			logger.Info("[okapi]", args...)
		}
		return err
	}
}

// Middleware is a basic authentication middleware that checks Basic Auth credentials.
// It returns 401 Unauthorized and sets the WWW-Authenticate header on failure.
func (b *BasicAuth) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		username, password, ok := c.request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(username), []byte(b.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(password), []byte(b.Password)) != 1 {

			realm := b.Realm
			if realm == "" {
				realm = okapiName
			}
			c.response.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			return c.String(http.StatusUnauthorized, "Unauthorized")
		}
		contextKey := b.ContextKey
		if contextKey == "" {
			contextKey = "username"
		}
		c.Set(contextKey, username)
		return next(c)
	}
}

// Middleware
//
// deprecate, use BasicAuth.Middleware
func (b *BasicAuthMiddleware) Middleware(next HandleFunc) HandleFunc {
	auth := BasicAuth{Username: b.Username, Password: b.Password, ContextKey: b.ContextKey}
	return auth.Middleware(next)
}

// Middleware is a middleware that limits the size of the request body to prevent excessive memory usage.
func (b BodyLimit) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		const errReadBody = "Failed to read request body"
		const errTooLarge = "Request body too large"

		// LimitReader prevents reading more than MaxBytes+1
		body, err := io.ReadAll(io.LimitReader(c.request.Body, b.MaxBytes+1))
		if err != nil {
			return c.String(http.StatusInternalServerError, errReadBody)
		}
		if int64(len(body)) > b.MaxBytes {
			return c.String(http.StatusRequestEntityTooLarge, errTooLarge)
		}

		// Reset request body for downstream handlers
		c.request.Body = io.NopCloser(bytes.NewReader(body))
		return next(c)
	}
}

// Middleware validates JWT tokens from the configured source
func (jwtAuth *JWTAuth) Middleware(next HandleFunc) HandleFunc {
	return func(c Context) error {
		tokenStr, err := jwtAuth.extractToken(c)
		if err != nil || tokenStr == "" {
			return c.AbortForbidden("Missing or invalid token", err)
		}

		keyFunc, err := jwtAuth.resolveKeyFunc()
		if err != nil {
			return c.AbortInternalServerError("Failed to resolve key function", err)

		}
		if jwtAuth.Algo != "" {
			jwtAlgo = []string{jwtAuth.Algo}
		}
		token, err := jwt.Parse(tokenStr, keyFunc,
			jwt.WithValidMethods(jwtAlgo),
			jwt.WithAudience(jwtAuth.Audience),
			jwt.WithIssuer(jwtAuth.Issuer))
		if err != nil || !token.Valid {
			return c.AbortUnauthorized("Invalid or expired token", err)
		}

		// If claims expression is configured, validate the claims
		if jwtAuth.ClaimsExpression != "" {
			valid, err := jwtAuth.validateJWTClaims(token)
			if err != nil {
				fPrintError("Failed to validate JWT claims expression", "error", err)
				return c.AbortUnauthorized("failed to validate JWT claims", err)
			}
			if !valid {
				fPrintError("JWT claims did not meet required expression ")
				return c.AbortUnauthorized("JWT claims did not meet required expression", err)
			}
		}
		// If custom claims validation function is provided, use it
		if jwtAuth.ValidateClaims != nil {
			if err = jwtAuth.ValidateClaims(token.Claims); err != nil {
				fPrintError("Failed to validate JWT role", "function", "ValidateClaims", "error", err)
				return c.AbortUnauthorized("Insufficient role", err)
			}
		}
		// If ValidateRole is configured, validate the role claim
		if jwtAuth.ValidateRole != nil {
			if err = jwtAuth.ValidateRole(token.Claims); err != nil {
				fPrintError("Failed to validate JWT role", "function", "ValidateRole", "error", err)
				return c.AbortUnauthorized("Insufficient role", err)
			}
		}
		// Store claims in context
		if jwtAuth.ContextKey != "" && token.Claims != nil {
			c.Set(jwtAuth.ContextKey, token.Claims)
		}
		// Forward specific claims to context if configured
		if jwtAuth.ForwardClaims != nil {
			if err = jwtAuth.forwardContextFromClaims(token, &c); err != nil {
				fPrintError("Failed to forward context from claims", "error", err)
			}
		}
		return next(c)
	}
}
