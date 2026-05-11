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
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// sampleClaims is the canonical claim set used by leaf-expression tests.
func sampleClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"email_verified": true,
		"role":           "admin",
		"name":           "Jane Doe",
		"user": map[string]any{
			"role":  "admin",
			"email": "jane@example.com",
		},
		"tags":    []any{"vip", "gold"},
		"groups":  []any{"engineering", "leadership"},
		"counter": float64(7),
	}
}

// extractClaimValue (dot-notation traversal)

func TestExtractClaimValue(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	t.Run("top-level key", func(t *testing.T) {
		v, err := extractClaimValue(claims, "role")
		if err != nil || v != "admin" {
			t.Errorf("got (%v, %v), want (admin, nil)", v, err)
		}
	})

	t.Run("nested key", func(t *testing.T) {
		v, err := extractClaimValue(claims, "user.email")
		if err != nil || v != "jane@example.com" {
			t.Errorf("got (%v, %v), want (jane@example.com, nil)", v, err)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		if _, err := extractClaimValue(claims, "user.missing"); err == nil {
			t.Error("expected error for missing key")
		}
	})

	t.Run("traverse non-object", func(t *testing.T) {
		if _, err := extractClaimValue(claims, "role.deeper"); err == nil {
			t.Error("expected error when traversing into a non-object")
		}
	})
}

// EqualsExpr

func TestEqualsExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		want    string
		expect  bool
		wantErr bool
	}{
		{"string match", "role", "admin", true, false},
		{"string mismatch", "role", "guest", false, false},
		{"nested match", "user.email", "jane@example.com", true, false},
		{"array contains expected", "tags", "vip", true, false},
		{"array does not contain", "tags", "missing", false, false},
		{"non-string scalar coerced", "counter", "7", true, false},
		{"missing key errors", "missing", "x", false, true},
	}

	claims := sampleClaims()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Equals(tt.key, tt.want).Evaluate(claims)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expect {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// PrefixExpr

func TestPrefixExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    string
		prefix string
		expect bool
	}{
		{"string match", "role", "adm", true},
		{"string no match", "role", "user", false},
		{"array element matches", "tags", "vip", true},
		{"array no element matches", "tags", "zz", false},
		{"non-string scalar formatted", "counter", "7", true},
	}

	claims := sampleClaims()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Prefix(tt.key, tt.prefix).Evaluate(claims)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expect {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// ContainsExpr

func TestContainsExpr_Substring(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	got, err := Contains("name", "Jane").Evaluate(claims)
	if err != nil || !got {
		t.Errorf("substring match: got (%v, %v), want (true, nil)", got, err)
	}

	got, err = Contains("name", "Bob").Evaluate(claims)
	if err != nil || got {
		t.Errorf("substring miss: got (%v, %v), want (false, nil)", got, err)
	}
}

func TestContainsExpr_ArrayMembership(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	// Multiple values switch ContainsExpr into "array membership" mode.
	got, err := Contains("tags", "vip", "premium").Evaluate(claims)
	if err != nil || !got {
		t.Errorf("array membership match: got (%v, %v), want (true, nil)", got, err)
	}

	got, err = Contains("tags", "missing", "also-missing").Evaluate(claims)
	if err != nil || got {
		t.Errorf("array membership miss: got (%v, %v), want (false, nil)", got, err)
	}
}

// OneOfExpr

func TestOneOfExpr(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	tests := []struct {
		name   string
		key    string
		values []string
		expect bool
	}{
		{"string is one of", "role", []string{"admin", "owner"}, true},
		{"string is none", "role", []string{"guest"}, false},
		{"array intersects", "tags", []string{"vip", "x"}, true},
		{"array disjoint", "tags", []string{"x", "y"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := OneOf(tt.key, tt.values...).Evaluate(claims)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expect {
				t.Errorf("got %v, want %v", got, tt.expect)
			}
		})
	}
}

// And / Or / Not, including short-circuit

// fixedExpr is a controllable expression for short-circuit testing.
type fixedExpr struct {
	value bool
	err   error
	calls *int
}

func (f *fixedExpr) Evaluate(_ jwt.MapClaims) (bool, error) {
	if f.calls != nil {
		*f.calls++
	}
	return f.value, f.err
}

func TestAndExpr(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	t.Run("both true", func(t *testing.T) {
		t.Parallel()
		got, err := And(Equals("role", "admin"), Equals("user.email", "jane@example.com")).Evaluate(claims)
		if err != nil || !got {
			t.Errorf("got (%v, %v), want (true, nil)", got, err)
		}
	})

	t.Run("short-circuits when left is false", func(t *testing.T) {
		t.Parallel()
		var rightCalls int
		got, err := And(&fixedExpr{value: false}, &fixedExpr{value: true, calls: &rightCalls}).Evaluate(claims)
		if err != nil || got {
			t.Errorf("got (%v, %v), want (false, nil)", got, err)
		}
		if rightCalls != 0 {
			t.Errorf("right side evaluated %d times, want 0", rightCalls)
		}
	})

	t.Run("propagates left error", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("boom")
		_, err := And(&fixedExpr{err: boom}, &fixedExpr{value: true}).Evaluate(claims)
		if !errors.Is(err, boom) {
			t.Errorf("got %v, want boom", err)
		}
	})
}

func TestOrExpr(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	t.Run("either true", func(t *testing.T) {
		t.Parallel()
		got, err := Or(Equals("role", "guest"), Equals("role", "admin")).Evaluate(claims)
		if err != nil || !got {
			t.Errorf("got (%v, %v), want (true, nil)", got, err)
		}
	})

	t.Run("short-circuits when left is true", func(t *testing.T) {
		t.Parallel()
		var rightCalls int
		got, err := Or(&fixedExpr{value: true}, &fixedExpr{value: false, calls: &rightCalls}).Evaluate(claims)
		if err != nil || !got {
			t.Errorf("got (%v, %v), want (true, nil)", got, err)
		}
		if rightCalls != 0 {
			t.Errorf("right side evaluated %d times, want 0", rightCalls)
		}
	})

	t.Run("both false", func(t *testing.T) {
		t.Parallel()
		got, err := Or(Equals("role", "x"), Equals("role", "y")).Evaluate(claims)
		if err != nil || got {
			t.Errorf("got (%v, %v), want (false, nil)", got, err)
		}
	})
}

func TestNotExpr(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	got, err := Not(Equals("role", "guest")).Evaluate(claims)
	if err != nil || !got {
		t.Errorf("Not(Equals(role,guest)) = (%v, %v), want (true, nil)", got, err)
	}

	got, err = Not(Equals("role", "admin")).Evaluate(claims)
	if err != nil || got {
		t.Errorf("Not(Equals(role,admin)) = (%v, %v), want (false, nil)", got, err)
	}
}

// ParseExpression — string-based DSL

func TestParseExpression_LeafFunctions(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"Equals matches", "Equals(`role`, `admin`)", true},
		{"Equals miss", "Equals(`role`, `guest`)", false},
		{"Prefix matches", "Prefix(`role`, `adm`)", true},
		{"Contains substring", "Contains(`name`, `Jane`)", true},
		{"Contains array membership", "Contains(`tags`, `vip`, `gold`)", true},
		{"OneOf matches", "OneOf(`role`, `admin`, `owner`)", true},
		{"OneOf miss", "OneOf(`role`, `guest`, `viewer`)", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := expr.Evaluate(claims)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseExpression_LogicalOperators(t *testing.T) {
	t.Parallel()

	claims := sampleClaims()

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{"AND both true", "Equals(`role`, `admin`) && Equals(`user.email`, `jane@example.com`)", true},
		{"AND one false", "Equals(`role`, `admin`) && Equals(`user.email`, `nope`)", false},
		{"OR right true", "Equals(`role`, `guest`) || Equals(`role`, `admin`)", true},
		{"OR both false", "Equals(`role`, `x`) || Equals(`role`, `y`)", false},
		{"NOT", "!Equals(`role`, `guest`)", true},
		{"AND has higher precedence than OR",
			"Equals(`role`, `guest`) || Equals(`role`, `admin`) && Equals(`user.email`, `jane@example.com`)",
			true,
		},
		{"parens override precedence",
			"(Equals(`role`, `guest`) || Equals(`role`, `admin`)) && Equals(`user.email`, `nope`)",
			false,
		},
		{"complex from middleware example",
			"Equals(`email_verified`, `true`) && OneOf(`user.role`, `admin`, `owner`) && Contains(`tags`, `vip`, `premium`)",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := expr.Evaluate(claims)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseExpression_SyntaxErrors(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",                              // empty
		"NotAFunction(`role`, `admin`)", // unknown function
		"Equals(`role`",                 // missing closing paren
		"(Equals(`role`, `admin`)",      // unbalanced parens
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			t.Parallel()
			if _, err := ParseExpression(expr); err == nil {
				t.Errorf("ParseExpression(%q) expected error, got nil", expr)
			}
		})
	}
}
