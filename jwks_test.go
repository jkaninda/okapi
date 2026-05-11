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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// rsaJWK builds a JWK entry for the given RSA public key.
func rsaJWK(t *testing.T, kid string, pub *rsa.PublicKey) Jwk {
	t.Helper()
	eBytes := bigIntToBytes(int64(pub.E))
	return Jwk{
		Kid: kid,
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

// ecJWK builds a JWK entry for the given ECDSA public key.
func ecJWK(t *testing.T, kid, crv string, pub *ecdsa.PublicKey) Jwk {
	t.Helper()
	return Jwk{
		Kid: kid,
		Kty: "EC",
		Crv: crv,
		X:   base64.RawURLEncoding.EncodeToString(pub.X.Bytes()),
		Y:   base64.RawURLEncoding.EncodeToString(pub.Y.Bytes()),
	}
}

// bigIntToBytes encodes an int as the minimal big-endian byte representation
// expected by the JWK "e" field.
func bigIntToBytes(n int64) []byte {
	if n == 0 {
		return []byte{0}
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte(n & 0xff)}, b...)
		n >>= 8
	}
	return b
}

// parseRSAPublicKey

func TestParseRSAPublicKey_RoundTrip(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	jwk := rsaJWK(t, "k1", &priv.PublicKey)
	got, err := parseRSAPublicKey(jwk.N, jwk.E)
	if err != nil {
		t.Fatalf("parseRSAPublicKey: %v", err)
	}
	if got.N.Cmp(priv.N) != 0 {
		t.Errorf("N mismatch")
	}
	if got.E != priv.E {
		t.Errorf("E = %d, want %d", got.E, priv.E)
	}
}

func TestParseRSAPublicKey_InvalidBase64(t *testing.T) {
	t.Parallel()

	if _, err := parseRSAPublicKey("!!!not-base64", "AQAB"); err == nil {
		t.Error("expected error for invalid N")
	}
	if _, err := parseRSAPublicKey("AQAB", "!!!"); err == nil {
		t.Error("expected error for invalid E")
	}
}

// parseECDSAPublicKey

func TestParseECDSAPublicKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		curve elliptic.Curve
		crv   string
	}{
		{"P-256", elliptic.P256(), "P-256"},
		{"P-384", elliptic.P384(), "P-384"},
		{"P-521", elliptic.P521(), "P-521"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			priv, err := ecdsa.GenerateKey(tt.curve, rand.Reader)
			if err != nil {
				t.Fatalf("generate EC key: %v", err)
			}

			jwk := ecJWK(t, "k1", tt.crv, &priv.PublicKey)
			got, err := parseECDSAPublicKey(jwk.Crv, jwk.X, jwk.Y)
			if err != nil {
				t.Fatalf("parseECDSAPublicKey: %v", err)
			}
			if got.X.Cmp(priv.X) != 0 || got.Y.Cmp(priv.Y) != 0 {
				t.Errorf("X/Y mismatch")
			}
			if got.Curve != tt.curve {
				t.Errorf("curve mismatch")
			}
		})
	}
}

func TestParseECDSAPublicKey_UnsupportedCurve(t *testing.T) {
	t.Parallel()

	if _, err := parseECDSAPublicKey("P-192", "AAA", "AAA"); err == nil {
		t.Error("expected error for unsupported curve")
	}
}

func TestParseECDSAPublicKey_InvalidBase64(t *testing.T) {
	t.Parallel()

	if _, err := parseECDSAPublicKey("P-256", "!!!", "AAA"); err == nil {
		t.Error("expected error for invalid X")
	}
	if _, err := parseECDSAPublicKey("P-256", "AAA", "!!!"); err == nil {
		t.Error("expected error for invalid Y")
	}
}

// Jwks.getKey

func TestJwksGetKey(t *testing.T) {
	t.Parallel()

	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", err)
	}

	set := &Jwks{Keys: []Jwk{
		rsaJWK(t, "rsa-1", &rsaPriv.PublicKey),
		ecJWK(t, "ec-1", "P-256", &ecPriv.PublicKey),
		{Kid: "weird", Kty: "OCT"},
	}}

	t.Run("returns RSA key", func(t *testing.T) {
		k, err := set.getKey("rsa-1")
		if err != nil {
			t.Fatalf("getKey(rsa-1): %v", err)
		}
		if _, ok := k.(*rsa.PublicKey); !ok {
			t.Errorf("expected *rsa.PublicKey, got %T", k)
		}
	})

	t.Run("returns ECDSA key", func(t *testing.T) {
		k, err := set.getKey("ec-1")
		if err != nil {
			t.Fatalf("getKey(ec-1): %v", err)
		}
		if _, ok := k.(*ecdsa.PublicKey); !ok {
			t.Errorf("expected *ecdsa.PublicKey, got %T", k)
		}
	})

	t.Run("rejects unsupported key type", func(t *testing.T) {
		if _, err := set.getKey("weird"); err == nil {
			t.Error("expected error for unsupported key type")
		}
	})

	t.Run("rejects unknown kid", func(t *testing.T) {
		if _, err := set.getKey("missing"); err == nil {
			t.Error("expected error for unknown kid")
		}
	})
}

// fetchJWKS

func TestFetchJWKS(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	want := &Jwks{Keys: []Jwk{rsaJWK(t, "rsa-1", &priv.PublicKey)}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	t.Cleanup(srv.Close)

	got, err := fetchJWKS(srv.URL)
	if err != nil {
		t.Fatalf("fetchJWKS: %v", err)
	}
	if len(got.Keys) != 1 || got.Keys[0].Kid != "rsa-1" {
		t.Fatalf("unexpected JWKS: %+v", got)
	}
}

func TestFetchJWKS_BadResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	if _, err := fetchJWKS(srv.URL); err == nil {
		t.Error("expected error for non-JSON response")
	}
}

func TestFetchJWKS_NetworkError(t *testing.T) {
	t.Parallel()

	if _, err := fetchJWKS("http://127.0.0.1:1"); err == nil {
		t.Error("expected error for unreachable JWKS endpoint")
	}
}
