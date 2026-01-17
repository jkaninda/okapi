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
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"valid email with plus", "user+tag@example.com", false},
		{"valid email with dash", "user-name@example.com", false},
		{"valid email with underscore", "user_name@example.com", false},
		{"valid email with numbers", "user123@example123.com", false},
		{"invalid - no @", "testexample.com", true},
		{"invalid - no domain", "test@", true},
		{"invalid - no local part", "@example.com", true},
		{"invalid - no TLD", "test@example", true},
		{"invalid - double @", "test@@example.com", true},
		{"invalid - spaces", "test @example.com", true},
		{"invalid - special chars", "test!#$@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDateTime(t *testing.T) {
	tests := []struct {
		name     string
		datetime string
		wantErr  bool
	}{
		{"valid RFC3339", "2024-01-15T10:30:00Z", false},
		{"valid RFC3339 with timezone", "2024-01-15T10:30:00+01:00", false},
		{"valid RFC3339 with negative timezone", "2024-01-15T10:30:00-05:00", false},
		{"valid RFC3339 with milliseconds", "2024-01-15T10:30:00.123Z", false},
		{"invalid - wrong format", "2024-01-15 10:30:00", true},
		{"invalid - date only", "2024-01-15", true},
		{"invalid - time only", "10:30:00", true},
		{"invalid - missing timezone", "2024-01-15T10:30:00", true},
		{"invalid - wrong separator", "2024/01/15T10:30:00Z", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateTime(tt.datetime)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDateTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	tests := []struct {
		name    string
		date    string
		wantErr bool
	}{
		{"valid date", "2024-01-15", false},
		{"valid date - leap year", "2024-02-29", false},
		{"valid date - start of year", "2024-01-01", false},
		{"valid date - end of year", "2024-12-31", false},
		{"invalid - with time", "2024-01-15T10:30:00", true},
		{"invalid - wrong separator", "2024/01/15", true},
		{"invalid - wrong format", "15-01-2024", true},
		{"invalid - month out of range", "2024-13-01", true},
		{"invalid - day out of range", "2024-01-32", true},
		{"invalid - non-leap year", "2023-02-29", true},
		{"invalid - text", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDate(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		wantErr  bool
	}{
		{"valid - seconds", "30s", false},
		{"valid - minutes", "5m", false},
		{"valid - hours", "2h", false},
		{"valid - milliseconds", "300ms", false},
		{"valid - microseconds", "500us", false},
		{"valid - nanoseconds", "1000ns", false},
		{"valid - combined", "2h45m30s", false},
		{"valid - decimal", "1.5h", false},
		{"valid - negative", "-5m", false},
		{"invalid - no unit", "30", true},
		{"invalid - wrong unit", "30x", true},
		{"invalid - text", "thirty seconds", true},
		{"invalid - spaces", "30 s", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDuration(tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDuration() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIPv4(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv4 - localhost", "127.0.0.1", false},
		{"valid IPv4 - zero", "0.0.0.0", false},
		{"valid IPv4 - max", "255.255.255.255", false},
		{"invalid - IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"invalid - out of range", "256.1.1.1", true},
		{"invalid - missing octet", "192.168.1", true},
		{"invalid - extra octet", "192.168.1.1.1", true},
		{"invalid - text", "not-an-ip", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPv4(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPv4() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIPv6(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv6 - full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"valid IPv6 - compressed", "2001:db8::1", false},
		{"valid IPv6 - localhost", "::1", false},
		{"valid IPv6 - all zeros", "::", false},
		{"invalid - IPv4", "192.168.1.1", true},
		{"invalid - wrong format", "2001:0db8:85a3::8a2e::7334", true},
		{"invalid - out of range", "gggg::1", true},
		{"invalid - text", "not-an-ip", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPv6(tt.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPv6() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		uuid    string
		wantErr bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid UUID - lowercase", "123e4567-e89b-12d3-a456-426614174000", false},
		{"valid UUID - uppercase", "550E8400-E29B-41D4-A716-446655440000", false},
		{"valid UUID - mixed case", "550e8400-E29B-41d4-A716-446655440000", false},
		{"invalid - no dashes", "550e8400e29b41d4a716446655440000", true},
		{"invalid - wrong format", "550e8400-e29b-41d4-a716", true},
		{"invalid - extra characters", "550e8400-e29b-41d4-a716-446655440000-extra", true},
		{"invalid - non-hex", "550e8400-e29b-41d4-a716-44665544000g", true},
		{"invalid - text", "not-a-uuid", true},
		{"invalid - empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUUID(tt.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRegex(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		pattern string
		wantErr bool
	}{
		{"valid phone number", "+1234567890", `^\+?[1-9]\d{1,14}$`, false},
		{"valid postal code", "12345", `^[0-9]{5}$`, false},
		{"valid alphanumeric", "abc123", `^[a-zA-Z0-9]+$`, false},
		{"invalid phone - letters", "abc1234567", `^\+?[1-9]\d{1,14}$`, true},
		{"invalid postal - too short", "1234", `^[0-9]{5}$`, true},
		{"invalid alphanumeric - special chars", "abc-123", `^[a-zA-Z0-9]+$`, true},
		{"invalid pattern", "test", `[invalid(`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegex(tt.value, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateEmail(b *testing.B) {
	email := "test@example.com"
	for i := 0; i < b.N; i++ {
		err := validateEmail(email)
		if err != nil {
			return
		}
	}
}

func BenchmarkValidateUUID(b *testing.B) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	for i := 0; i < b.N; i++ {
		err := validateUUID(uuid)
		if err != nil {
			return
		}
	}
}

func BenchmarkValidateIPv4(b *testing.B) {
	ip := "192.168.1.1"
	for i := 0; i < b.N; i++ {
		err := validateIPv4(ip)
		if err != nil {
			return
		}
	}
}

func BenchmarkValidateDateTime(b *testing.B) {
	datetime := "2024-01-15T10:30:00Z"
	for i := 0; i < b.N; i++ {
		err := validateDateTime(datetime)
		if err != nil {
			return
		}
	}
}

// Integration tests with Context binding
func TestContextBindWithFormatValidation(t *testing.T) {
	type TestRequest struct {
		Email     string `json:"email" format:"email" required:"true"`
		BirthDate string `json:"birth_date" format:"date"`
		CreatedAt string `json:"created_at" format:"date-time"`
		UserID    string `json:"user_id" format:"uuid"`
		Timeout   string `json:"timeout" format:"duration"`
		IPAddress string `json:"ip_address" format:"ipv4"`
		Phone     string `json:"phone" format:"regex" pattern:"^\\+?[1-9]\\d{1,14}$"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid all formats",
			body: `{
				"email": "test@example.com",
				"birth_date": "1990-01-15",
				"created_at": "2024-01-15T10:30:00Z",
				"user_id": "550e8400-e29b-41d4-a716-446655440000",
				"timeout": "30s",
				"ip_address": "192.168.1.1",
				"phone": "+1234567890"
			}`,
			wantErr: false,
		},
		{
			name: "invalid email format",
			body: `{
				"email": "invalid-email",
				"birth_date": "1990-01-15",
				"created_at": "2024-01-15T10:30:00Z",
				"user_id": "550e8400-e29b-41d4-a716-446655440000"
			}`,
			wantErr:     true,
			errContains: "invalid email format",
		},
		{
			name: "invalid date format",
			body: `{
				"email": "test@example.com",
				"birth_date": "01/15/1990",
				"created_at": "2024-01-15T10:30:00Z"
			}`,
			wantErr:     true,
			errContains: "invalid date format",
		},
		{
			name: "invalid date-time format",
			body: `{
				"email": "test@example.com",
				"birth_date": "1990-01-15",
				"created_at": "2024-01-15 10:30:00"
			}`,
			wantErr:     true,
			errContains: "invalid date-time format",
		},
		{
			name: "invalid UUID format",
			body: `{
				"email": "test@example.com",
				"user_id": "not-a-uuid"
			}`,
			wantErr:     true,
			errContains: "invalid UUID format",
		},
		{
			name: "invalid duration format",
			body: `{
				"email": "test@example.com",
				"timeout": "30seconds"
			}`,
			wantErr:     true,
			errContains: "invalid duration format",
		},
		{
			name: "invalid IPv4 format",
			body: `{
				"email": "test@example.com",
				"ip_address": "256.1.1.1"
			}`,
			wantErr:     true,
			errContains: "invalid IP address",
		},
		{
			name: "invalid regex pattern",
			body: `{
				"email": "test@example.com",
				"phone": "invalid-phone"
			}`,
			wantErr:     true,
			errContains: "does not match pattern",
		},
		{
			name: "missing required email",
			body: `{
				"birth_date": "1990-01-15"
			}`,
			wantErr:     true,
			errContains: "required",
		},
		{
			name: "empty optional fields - should pass",
			body: `{
				"email": "test@example.com"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.request.Header.Set("Content-Type", "application/json")

			var testReq TestRequest
			err := c.Bind(&testReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestNestedStructFormatValidation(t *testing.T) {
	type Address struct {
		City    string `json:"city" required:"true"`
		ZipCode string `json:"zip_code" format:"regex" pattern:"^[0-9]{5}$"`
	}

	type UserRequest struct {
		Email   string  `json:"email" format:"email" required:"true"`
		Address Address `json:"address"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid nested struct",
			body: `{
				"email": "test@example.com",
				"address": {
					"city": "New York",
					"zip_code": "10001"
				}
			}`,
			wantErr: false,
		},
		{
			name: "invalid nested zip code format",
			body: `{
				"email": "test@example.com",
				"address": {
					"city": "New York",
					"zip_code": "ABC12"
				}
			}`,
			wantErr:     true,
			errContains: "does not match pattern",
		},
		{
			name: "missing required nested field",
			body: `{
				"email": "test@example.com",
				"address": {
					"zip_code": "10001"
				}
			}`,
			wantErr:     true,
			errContains: "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.request.Header.Set("Content-Type", "application/json")
			var userReq UserRequest
			err := c.Bind(&userReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestFormatValidationWithQueryParams(t *testing.T) {
	type QueryRequest struct {
		Email  string `query:"email" format:"email"`
		UserID string `query:"user_id" format:"uuid"`
		IP     string `query:"ip" format:"ipv4"`
	}

	tests := []struct {
		name        string
		queryString string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid query params",
			queryString: "email=test@example.com&user_id=550e8400-e29b-41d4-a716-446655440000&ip=192.168.1.1",
			wantErr:     false,
		},
		{
			name:        "invalid email in query",
			queryString: "email=invalid-email&user_id=550e8400-e29b-41d4-a716-446655440000",
			wantErr:     true,
			errContains: "invalid email format",
		},
		{
			name:        "invalid UUID in query",
			queryString: "email=test@example.com&user_id=not-a-uuid",
			wantErr:     true,
			errContains: "invalid UUID format",
		},
		{
			name:        "invalid IP in query",
			queryString: "email=test@example.com&ip=256.1.1.1",
			wantErr:     true,
			errContains: "invalid IP address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test?"+tt.queryString, nil)
			w := httptest.NewRecorder()
			c := NewContext(nil, w, req)

			var queryReq QueryRequest
			err := c.Bind(&queryReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}
func TestCheckEnum(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		enumTag string
		wantErr bool
	}{
		{"valid - first value", "pending", "pending,processing,shipped", false},
		{"valid - middle value", "processing", "pending,processing,shipped", false},
		{"valid - last value", "shipped", "pending,processing,shipped", false},
		{"valid - single option", "active", "active", false},
		{"valid - with spaces", "processing", "pending, processing, shipped", false},
		{"invalid - not in list", "completed", "pending,processing,shipped", true},
		{"invalid - case sensitive", "PENDING", "pending,processing,shipped", true},
		{"invalid - partial match", "proc", "pending,processing,shipped", true},
		{"invalid - empty not allowed", "invalid", "pending,processing,shipped", true},
		{"empty string - should pass", "", "pending,processing,shipped", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.ValueOf(tt.value)
			err := checkEnum(field, tt.enumTag)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkEnum() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "not one of the allowed values") {
					t.Errorf("checkEnum() error should mention allowed values, got: %v", err)
				}
			}
		})
	}
}

func TestCheckEnumNonStringField(t *testing.T) {
	field := reflect.ValueOf(123)
	err := checkEnum(field, "1,2,3")

	if err == nil {
		t.Error("checkEnum() should return error for non-string field")
	}

	if !strings.Contains(err.Error(), "can only be applied to string fields") {
		t.Errorf("checkEnum() error should mention string fields only, got: %v", err)
	}
}

func TestEnumValidationIntegration(t *testing.T) {
	type OrderRequest struct {
		Status        string `json:"status" enum:"pending,processing,shipped,delivered" required:"true"`
		PaymentMethod string `json:"payment_method" enum:"credit_card,paypal,stripe"`
		Priority      string `json:"priority" enum:"low,medium,high"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid all enums",
			body: `{
                "status": "pending",
                "payment_method": "credit_card",
                "priority": "high"
            }`,
			wantErr: false,
		},
		{
			name: "valid with optional empty",
			body: `{
                "status": "processing",
                "payment_method": "paypal"
            }`,
			wantErr: false,
		},
		{
			name: "invalid status enum",
			body: `{
                "status": "completed",
                "payment_method": "credit_card"
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name: "invalid payment method enum",
			body: `{
                "status": "pending",
                "payment_method": "bitcoin"
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name: "case sensitive - uppercase not allowed",
			body: `{
                "status": "PENDING",
                "payment_method": "credit_card"
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name: "missing required enum field",
			body: `{
                "payment_method": "credit_card"
            }`,
			wantErr:     true,
			errContains: "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.request.Header.Set("Content-Type", "application/json")

			var orderReq OrderRequest
			err := c.Bind(&orderReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestEnumWithOtherValidations(t *testing.T) {
	type ComplexRequest struct {
		Status   string `json:"status" enum:"active,inactive,pending" required:"true"`
		Email    string `json:"email" format:"email" required:"true"`
		Role     string `json:"role" enum:"admin,user,guest" required:"true"`
		Username string `json:"username" minLength:"3" maxLength:"20" pattern:"^[a-zA-Z0-9_]+$"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid all validations",
			body: `{
                "status": "active",
                "email": "test@example.com",
                "role": "admin",
                "username": "john_doe"
            }`,
			wantErr: false,
		},
		{
			name: "invalid enum but valid other fields",
			body: `{
                "status": "deleted",
                "email": "test@example.com",
                "role": "admin",
                "username": "john_doe"
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name: "valid enum but invalid email",
			body: `{
                "status": "active",
                "email": "invalid-email",
                "role": "admin",
                "username": "john_doe"
            }`,
			wantErr:     true,
			errContains: "invalid email format",
		},
		{
			name: "multiple enum fields, one invalid",
			body: `{
                "status": "active",
                "email": "test@example.com",
                "role": "superadmin",
                "username": "john_doe"
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.request.Header.Set("Content-Type", "application/json")

			var req ComplexRequest
			err := c.Bind(&req)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestNestedStructWithEnum(t *testing.T) {
	type Address struct {
		Country string `json:"country" enum:"US,CA,UK,FR,DE" required:"true"`
		State   string `json:"state"`
	}

	type UserRequest struct {
		Role    string  `json:"role" enum:"admin,user,guest" required:"true"`
		Address Address `json:"address"`
	}

	tests := []struct {
		name        string
		body        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid nested enum",
			body: `{
                "role": "admin",
                "address": {
                    "country": "US",
                    "state": "CA"
                }
            }`,
			wantErr: false,
		},
		{
			name: "invalid nested country enum",
			body: `{
                "role": "admin",
                "address": {
                    "country": "JP",
                    "state": "Tokyo"
                }
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name: "invalid parent enum",
			body: `{
                "role": "superadmin",
                "address": {
                    "country": "US"
                }
            }`,
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodPost, "/test", strings.NewReader(tt.body))
			c.request.Header.Set("Content-Type", "application/json")

			var userReq UserRequest
			err := c.Bind(&userReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestEnumWithQueryParams(t *testing.T) {
	type QueryRequest struct {
		Status string `query:"status" enum:"active,inactive,pending"`
		Sort   string `query:"sort" enum:"asc,desc" default:"asc"`
	}

	tests := []struct {
		name        string
		queryString string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid enum in query",
			queryString: "status=active&sort=desc",
			wantErr:     false,
		},
		{
			name:        "invalid status enum",
			queryString: "status=deleted&sort=asc",
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
		{
			name:        "invalid sort enum",
			queryString: "status=active&sort=random",
			wantErr:     true,
			errContains: "not one of the allowed values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := NewTestContext(http.MethodGet, "/test?"+tt.queryString, nil)

			var queryReq QueryRequest
			err := c.Bind(&queryReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("Context.Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Context.Bind() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func BenchmarkCheckEnum(b *testing.B) {
	field := reflect.ValueOf("processing")
	enumTag := "pending,processing,shipped,delivered,cancelled"

	for i := 0; i < b.N; i++ {
		err := checkEnum(field, enumTag)
		if err != nil {
			return
		}
	}
}
