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
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
)

// ********** Helpers **********************

// extractToken pulls the token from header, query or cookie
func (jwtAuth *JWTAuth) extractToken(c *Context) (string, error) {
	tokenLookup := jwtAuth.TokenLookup
	if tokenLookup == "" {
		tokenLookup = "header:Authorization"
	}
	parts := strings.Split(tokenLookup, ":")
	if len(parts) != 2 {
		return "", errors.New("invalid token lookup config")
	}

	source, name := parts[0], parts[1]
	switch source {
	case "header":
		auth := c.request.Header.Get(name)
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer "), nil
		}
		return auth, nil
	case "query":
		return c.Query(name), nil
	case "cookie":
		cookie, err := c.request.Cookie(name)
		if err != nil {
			return "", err
		}
		return cookie.Value, nil
	default:
		return "", errors.New("unsupported token source")
	}
}

// ValidateToken checks the JWT token and returns the claims if valid
func (jwtAuth *JWTAuth) ValidateToken(c *Context) (jwt.MapClaims, error) {
	tokenStr, err := jwtAuth.extractToken(c)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return signingSecret(jwtAuth.SigningSecret, jwtAuth.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid or expired token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return nil, errors.New("invalid claims type")
}
func (jwtAuth *JWTAuth) resolveKeyFunc() (jwt.Keyfunc, error) {
	if jwtAuth.JwksUrl != "" {
		return func(token *jwt.Token) (interface{}, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("missing 'kid' in JWT header")
			}
			jwks, err := fetchJWKS(jwtAuth.JwksUrl)
			if err != nil {
				return nil, err
			}
			return jwks.getKey(kid)
		}, nil
	}

	secret := signingSecret(jwtAuth.SigningSecret, jwtAuth.SecretKey)
	if secret != nil {
		return func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		}, nil
	}
	if jwtAuth.JwksFile != nil && len(jwtAuth.JwksFile.Keys) != 0 {
		return func(token *jwt.Token) (interface{}, error) {
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, fmt.Errorf("missing 'kid' in JWT header")
			}
			return jwtAuth.JwksFile.getKey(kid)
		}, nil
	}
	if jwtAuth.RsaKey != nil {
		return func(token *jwt.Token) (interface{}, error) {
			return jwtAuth.RsaKey, nil
		}, nil
	}

	return nil, fmt.Errorf("no JWT secret, RSA key, or JWKS URL configured")
}

func signingSecret(signingSecret, old []byte) []byte {
	if signingSecret != nil {
		return signingSecret
	}
	return old

}

// Updated validateJWTClaims method
func (jwtAuth *JWTAuth) validateJWTClaims(token *jwt.Token) (bool, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, fmt.Errorf("invalid claims format")
	}

	// Use expression-based validation if available
	if jwtAuth.ClaimsExpression != "" {
		// Parse expression if not already cached
		if jwtAuth.parsedExpression == nil {
			expr, err := ParseExpression(jwtAuth.ClaimsExpression)
			if err != nil {
				return false, fmt.Errorf("failed to parse claims expression: %v", err)
			}
			jwtAuth.parsedExpression = expr
		}

		result, err := jwtAuth.parsedExpression.Evaluate(claims)
		if err != nil {
			return false, fmt.Errorf("expression evaluation failed: %v", err)
		}
		return result, nil
	}

	return true, nil // No claims validation configured
}

// forwardContextFromClaims extracts values from JWT claims and sets them in the request context
func (jwtAuth *JWTAuth) forwardContextFromClaims(token *jwt.Token, c *Context) error {
	// Get claims as MapClaims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid claims format")
	}

	for contextKey, claimPath := range jwtAuth.ForwardClaims {
		// Extract claim value using nested key traversal with dot notation support
		claimValue, err := jwtAuth.extractNestedClaimValue(claims, claimPath)
		if err != nil {
			fPrintError("Warning: Could not extract claim ", "claimPath", claimPath, "error", err)
			continue
		}
		// Convert claim value to string
		value := jwtAuth.formatContextValue(claimValue)
		if value == "" {
			continue // Skip empty values
		}
		c.Set(contextKey, value)
	}

	return nil
}

// extractNestedClaimValue extracts a value from JWT claims using dot notation for nested keys
func (jwtAuth *JWTAuth) extractNestedClaimValue(claims jwt.MapClaims, claimKey string) (interface{}, error) {
	// Handle nested keys using dot notation (e.g., "user.profile.email")
	keys := strings.Split(claimKey, ".")
	var current interface{} = map[string]interface{}(claims)

	// Traverse nested keys
	for i, k := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[k]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("claim key '%s' not found at path '%s'", k, strings.Join(keys[:i+1], "."))
			}
		} else {
			return nil, fmt.Errorf("cannot traverse claim path at key '%s' (expected object, got %T)", k, current)
		}
	}

	return current, nil
}

// formatContextValue converts a claim value to a context string
func (jwtAuth *JWTAuth) formatContextValue(claimValue interface{}) string {
	// Convert claim value to string
	switch cv := claimValue.(type) {
	case string:
		return cv
	case float64:
		return fmt.Sprintf("%.0f", cv)
	case bool:
		return fmt.Sprintf("%t", cv)
	case []interface{}:
		// Join array values with comma
		var strValues []string
		for _, v := range cv {
			if vStr, ok := v.(string); ok {
				strValues = append(strValues, vStr)
			} else {
				strValues = append(strValues, fmt.Sprintf("%v", v))
			}
		}
		return strings.Join(strValues, ",")
	default:
		return fmt.Sprintf("%v", cv)
	}
}

// GenerateJwtToken generates a JWT with custom claims and expiry
func GenerateJwtToken(secret []byte, claims jwt.MapClaims, ttl time.Duration) (string, error) {
	claims["exp"] = time.Now().Add(ttl).Unix()
	claims["iat"] = time.Now().Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
