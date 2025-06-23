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

package middlewares

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jkaninda/okapi"
	"github.com/jkaninda/okapi/examples/route-definition/models"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

var (
	// signingSecret is used to sign the JWT tokens
	signingSecret = "supersecret"

	JWTAuth = &okapi.JWTAuth{
		SigningSecret:    []byte(signingSecret),
		TokenLookup:      "header:Authorization",
		ClaimsExpression: "Equals(`email_verified`, `true`) && Equals(`user.role`, `admin`) && Contains(`permissions`, `create`, `delete`, `update`)",
		ForwardClaims: map[string]string{
			"email": "user.email",
			"role":  "user.role",
			"name":  "user.name",
		},
		// CustomClaims claims validation function
		ValidateClaims: func(c okapi.Context, claims jwt.Claims) error {
			slog.Info("Validating JWT claims for role using custom function")
			// Simulate a custom claims validation
			mapClaims, ok := claims.(jwt.MapClaims)
			if !ok {
				return errors.New("invalid claims type")
			}
			if role, exist := mapClaims["user"].(map[string]interface{})["role"]; exist {
				fmt.Println("Role from claims:", role)
			}
			return nil
		},
	}
	jwtClaims = jwt.MapClaims{
		"sub": "12345",
		"iss": "okapi.example.com",
		"aud": "okapi.example.com",
		"user": map[string]string{
			"name":  "",
			"role":  "",
			"email": "",
		},
		"email_verified": true,
		"permissions":    []string{"read", "create"},
		"exp":            time.Now().Add(2 * time.Hour).Unix(),
	}
	adminPermissions = []string{"read", "create", "delete", "update"}
)

func Login(authRequest *models.AuthRequest) (models.AuthResponse, error) {
	// This is where you would typically validate the user credentials against a database

	slog.Info("Login attempt", "username", authRequest.Username)
	// Simulate a login function that returns a JWT token
	if authRequest.Username != "admin" && authRequest.Password != "password" ||
		authRequest.Username != "user" && authRequest.Password != "password" {
		return models.AuthResponse{
			Success: false,
			Message: "Invalid username or password",
		}, fmt.Errorf("username or password is wrong")
	}

	if _, ok := jwtClaims["user"].(map[string]string); ok {
		jwtClaims["user"].(map[string]string)["name"] = strings.ToUpper(authRequest.Username)
		jwtClaims["user"].(map[string]string)["role"] = authRequest.Username
		jwtClaims["user"].(map[string]string)["email"] = authRequest.Username + "@example.com"
		jwtClaims["permissions"] = []string{"read"}

		// If the user is an admin, add admin permissions
		if authRequest.Username == "admin" {
			jwtClaims["permissions"] = adminPermissions
		}

	}
	// Set the expiration time for the JWT token
	expireAt := 30 * time.Minute
	jwtClaims["exp"] = time.Now().Add(expireAt).Unix()

	token, err := okapi.GenerateJwtToken(JWTAuth.SigningSecret, jwtClaims, expireAt)
	if err != nil {

		return models.AuthResponse{
			Success: false,
			Message: "Invalid username or password",
		}, fmt.Errorf("failed to generate JWT token: %w", err)
	}
	return models.AuthResponse{
		Success:   true,
		Message:   "Welcome back " + authRequest.Username,
		Token:     token,
		ExpiresAt: time.Now().Add(expireAt).Unix(),
	}, nil

}
func CustomMiddleware(next okapi.HandleFunc) okapi.HandleFunc {
	return func(c okapi.Context) error {
		slog.Info("Custom middleware executed", "path", c.Request().URL.Path, "method", c.Request().Method)
		// You can add any custom logic here, such as logging, authentication, etc.
		// For example, let's log the request method and URL
		slog.Info("Request received", "method", c.Request().Method, "url", c.Request().URL.String())
		// Call the next handler in the chain
		if err := next(c); err != nil {
			// If an error occurs, log it and return a generic error response
			slog.Error("Error in custom middleware", "error", err)
			return c.JSON(http.StatusInternalServerError, okapi.M{"error": "Internal Server Error"})
		}
		return nil
	}
}
