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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// TestFetchJWKS_RejectsNonOK covers an endpoint whose error page happens to be
// valid JSON: without a status check it was accepted as a key set.
func TestFetchJWKS_RejectsNonOK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(srv.Close)

	if _, err := fetchJWKS(srv.URL); err == nil {
		t.Error("expected error for a non-200 JWKS response")
	}
}

// TestFetchJWKS_LimitsBodySize covers a hostile endpoint streaming an
// unbounded body into memory.
func TestFetchJWKS_LimitsBodySize(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[{"kid":"pad","kty":"RSA","n":"`))
		chunk := bytes.Repeat([]byte("A"), 64<<10)
		for written := 0; written < 4<<20; written += len(chunk) {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	// The reader is cut off mid-document, so decoding fails rather than
	// consuming the whole stream.
	if _, err := fetchJWKS(srv.URL); err == nil {
		t.Error("expected error for an oversized JWKS body")
	}
}

// TestJWKSIsCached guards against refetching the key set on every request,
// which lets unauthenticated traffic drive outbound calls to the identity
// provider one-for-one.
func TestJWKSIsCached(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(srv.Close)

	auth := &JWTAuth{JwksUrl: srv.URL, Audience: "api"}
	keyFunc, err := auth.resolveKeyFunc()
	if err != nil {
		t.Fatalf("resolveKeyFunc: %v", err)
	}

	tok := &jwt.Token{Header: map[string]any{"kid": "some-kid"}}
	for i := 0; i < 25; i++ {
		// Every lookup fails (the set is empty); the point is that failing
		// lookups must not each cost an outbound request.
		_, _ = keyFunc(tok)
	}

	if got := atomic.LoadInt64(&hits); got > 1 {
		t.Errorf("JWKS endpoint hit %d times for 25 validations, want 1", got)
	}
}

// TestJWKSFailureIsNotRetriedPerRequest covers the same amplification concern
// when the endpoint is down.
func TestJWKSFailureIsNotRetriedPerRequest(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	auth := &JWTAuth{JwksUrl: srv.URL}
	keyFunc, err := auth.resolveKeyFunc()
	if err != nil {
		t.Fatalf("resolveKeyFunc: %v", err)
	}

	tok := &jwt.Token{Header: map[string]any{"kid": "some-kid"}}
	for i := 0; i < 10; i++ {
		if _, err := keyFunc(tok); err == nil {
			t.Fatal("expected an error while the endpoint is failing")
		}
	}

	if got := atomic.LoadInt64(&hits); got > 1 {
		t.Errorf("failing endpoint hit %d times for 10 validations, want 1", got)
	}
}

// backdateJWKS ages the cached entry for url by age, as if its last fetch had
// happened that long ago.
func backdateJWKS(url string, age time.Duration) {
	e := jwksEntryFor(url)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.fetchedAt = e.fetchedAt.Add(-age)
	e.lastFetch = e.lastFetch.Add(-age)
}

// waitFor polls cond until it holds, failing the test after a few seconds.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 5s")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestJWKSServesStaleSetWhileEndpointFails guards against a refetch storm once
// the TTL expires during an identity provider outage: the cooldown applied only
// while nothing was cached, so every request refetched — serialised, with a 5s
// timeout each — and authentication failed although a usable set was cached.
func TestJWKSServesStaleSetWhileEndpointFails(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	set := &Jwks{Keys: []Jwk{rsaJWK(t, "k1", &key.PublicKey)}}

	var hits atomic.Int64
	var failing atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if failing.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(set)
	}))
	t.Cleanup(srv.Close)

	const ttl = time.Minute
	auth := &JWTAuth{JwksUrl: srv.URL, JwksCacheTTL: ttl}
	keyFunc, err := auth.resolveKeyFunc()
	if err != nil {
		t.Fatalf("resolveKeyFunc: %v", err)
	}
	tok := &jwt.Token{Header: map[string]any{"kid": "k1"}}
	if _, err := keyFunc(tok); err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	failing.Store(true)
	// Past the TTL, well within the grace period, and no recent fetch.
	backdateJWKS(srv.URL, 2*ttl)

	validate := func(phase string) {
		t.Helper()
		for i := 0; i < 20; i++ {
			if _, err := keyFunc(tok); err != nil {
				t.Fatalf("%s, validation %d: cached key set not served during an outage: %v", phase, i, err)
			}
		}
	}
	validate("while refreshing")
	waitFor(t, func() bool { return hits.Load() >= 2 })
	validate("after the refresh failed")

	if got := hits.Load(); got != 2 {
		t.Errorf("JWKS endpoint hit %d times, want 2 (the initial fetch and one refresh)", got)
	}
}

// TestJWKSStaleSetIsBounded covers the other side of the grace period: a set
// is not served indefinitely while the endpoint keeps failing, so a key the
// issuer withdrew eventually stops verifying tokens.
func TestJWKSStaleSetIsBounded(t *testing.T) {
	var failing atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failing.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(srv.Close)

	const ttl = time.Minute
	if _, err := jwksFromCache(srv.URL, ttl); err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	failing.Store(true)
	backdateJWKS(srv.URL, 24*time.Hour)

	if keys, err := jwksFromCache(srv.URL, ttl); err == nil {
		t.Fatalf("served a key set a day past its TTL: %+v", keys)
	}
	// Still refused while cooling down after that failure.
	if keys, err := jwksFromCache(srv.URL, ttl); err == nil {
		t.Errorf("served a key set a day past its TTL during the cooldown: %+v", keys)
	}
}

// TestJWKSRequestsDoNotQueueBehindSlowRefresh guards against every request
// waiting on the per-URL lock while a single refresh sits on a slow endpoint,
// although a usable key set is already cached.
func TestJWKSRequestsDoNotQueueBehindSlowRefresh(t *testing.T) {
	var hits atomic.Int64
	var slow atomic.Bool
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if slow.Load() {
			select {
			case <-release:
			case <-r.Context().Done():
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"keys":[]}`))
	}))
	t.Cleanup(srv.Close)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock) // runs before srv.Close

	const ttl = time.Minute
	if _, err := jwksFromCache(srv.URL, ttl); err != nil {
		t.Fatalf("initial fetch: %v", err)
	}

	slow.Store(true)
	backdateJWKS(srv.URL, 2*ttl)

	// The first request after expiry starts the refresh, which now hangs.
	go func() { _, _ = jwksFromCache(srv.URL, ttl) }()
	waitFor(t, func() bool { return hits.Load() >= 2 })

	const n = 16
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			_, err := jwksFromCache(srv.URL, ttl)
			errs <- err
		}()
	}

	deadline := time.After(2 * time.Second)
	for i := 0; i < n; i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Errorf("request failed during a slow refresh: %v", err)
			}
		case <-deadline:
			t.Fatalf("%d of %d requests still waiting on a slow JWKS refresh", n-i, n)
		}
	}

	unblock()
	if got := hits.Load(); got != 2 {
		t.Errorf("JWKS endpoint hit %d times, want 2 (the initial fetch and one refresh)", got)
	}
}

// TestParseRSAPublicKey_RejectsWeakKey covers degenerate moduli: a three-byte
// n yields a 17-bit key that is trivially factorable.
func TestParseRSAPublicKey_RejectsWeakKey(t *testing.T) {
	t.Parallel()

	if _, err := parseRSAPublicKey("AQAB", "AQAB"); err == nil {
		t.Error("expected error for a 17-bit RSA modulus")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	jwk := rsaJWK(t, "small", &priv.PublicKey)
	if _, err := parseRSAPublicKey(jwk.N, jwk.E); err == nil {
		t.Error("expected error for a 1024-bit RSA modulus")
	}
}

// TestParseECDSAPublicKey_RejectsOffCurvePoint covers points that are not on
// the named curve, which are not usable public keys.
func TestParseECDSAPublicKey_RejectsOffCurvePoint(t *testing.T) {
	t.Parallel()

	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	if _, err := parseECDSAPublicKey("P-256", enc([]byte{1}), enc([]byte{2})); err == nil {
		t.Error("expected error for a point that is not on P-256")
	}
}

// TestJwksGetKeyHonoursUseAndAlg covers keys a JWKS publishes for a purpose
// other than signature verification.
func TestJwksGetKeyHonoursUseAndAlg(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	encKey := rsaJWK(t, "enc-1", &priv.PublicKey)
	encKey.Use = "enc"
	wrongAlg := rsaJWK(t, "alg-1", &priv.PublicKey)
	wrongAlg.Alg = "ES256"
	sigKey := rsaJWK(t, "sig-1", &priv.PublicKey)
	sigKey.Use = "sig"
	sigKey.Alg = "RS256"

	set := &Jwks{Keys: []Jwk{encKey, wrongAlg, sigKey}}

	if _, err := set.getKey("enc-1"); err == nil {
		t.Error("expected error for a key published with use=enc")
	}
	if _, err := set.getKey("alg-1"); err == nil {
		t.Error("expected error for an RSA key declaring alg=ES256")
	}
	if _, err := set.getKey("sig-1"); err != nil {
		t.Errorf("getKey(sig-1): %v", err)
	}
}
