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
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// defaultJWKSCacheTTL is how long a fetched key set is reused before the
	// endpoint is consulted again. Override with JWTAuth.JwksCacheTTL.
	defaultJWKSCacheTTL = 15 * time.Minute

	// jwksRefreshCooldown bounds out-of-band fetches: an unknown kid may
	// trigger at most one refresh per cooldown, and a failing endpoint is
	// retried no more often than that. Without it an attacker inventing kids
	// drives one outbound request per inbound request.
	jwksRefreshCooldown = time.Minute

	// jwksFetchTimeout bounds a single fetch. The default http.Client has no
	// timeout, so a slow endpoint would pin request goroutines indefinitely.
	jwksFetchTimeout = 5 * time.Second

	// jwksMaxBodyBytes caps the response body read from a JWKS endpoint.
	jwksMaxBodyBytes = 1 << 20

	// minRSAKeyBits is the smallest RSA modulus accepted from a key set.
	minRSAKeyBits = 2048
)

// jwksHTTPClient fetches key sets. It is a dedicated client so the timeout
// cannot be inherited from, or leak into, application code.
var jwksHTTPClient = &http.Client{Timeout: jwksFetchTimeout}

type Jwks struct {
	Keys []Jwk `json:"keys"`
}

type Jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	// Use is the key's intended use ("sig" or "enc"). When the JWK declares
	// it, anything other than "sig" is refused for signature verification.
	Use string `json:"use"`
	// Alg is the algorithm the key is published for. When declared, it must be
	// consistent with the key type.
	Alg string `json:"alg"`
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
	Crv string `json:"crv"` // for EC
	X   string `json:"x"`   // for EC
	Y   string `json:"y"`   // for EC
}

// fetchJWKS retrieves a key set over HTTP.
//
// Callers should prefer jwksFromCache: this performs an unconditional fetch.
func fetchJWKS(jwksURL string) (*Jwks, error) {
	resp, err := jwksHTTPClient.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_, _ = fmt.Fprintf(defaultErrorWriter, "error closing body: %v", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var keySet Jwks
	if err = json.NewDecoder(io.LimitReader(resp.Body, jwksMaxBodyBytes)).Decode(&keySet); err != nil {
		return nil, err
	}
	return &keySet, nil
}

// jwksEntry is one cached key set. Its mutex serialises fetches for a single
// URL, so a burst of requests collapses into one outbound call rather than a
// thundering herd, while other endpoints stay unblocked.
type jwksEntry struct {
	mu        sync.Mutex
	keys      *Jwks
	err       error
	fetchedAt time.Time
	lastFetch time.Time
}

var (
	jwksStoreMu sync.Mutex
	jwksStore   = map[string]*jwksEntry{}
)

func jwksEntryFor(url string) *jwksEntry {
	jwksStoreMu.Lock()
	defer jwksStoreMu.Unlock()

	e, ok := jwksStore[url]
	if !ok {
		e = &jwksEntry{}
		jwksStore[url] = e
	}
	return e
}

// jwksFromCache returns the key set for url, fetching it only when nothing is
// cached or the entry has aged past ttl.
func jwksFromCache(url string, ttl time.Duration) (*Jwks, error) {
	if ttl <= 0 {
		ttl = defaultJWKSCacheTTL
	}

	e := jwksEntryFor(url)
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.keys != nil && time.Since(e.fetchedAt) < ttl {
		return e.keys, nil
	}
	return e.fetch(url)
}

// jwksRefresh refetches when a kid is absent from the cached set, which
// usually means the issuer rotated keys. It is rate-limited, so an attacker
// presenting invented kids cannot turn inbound requests into outbound ones.
func jwksRefresh(url string) (*Jwks, bool) {
	e := jwksEntryFor(url)
	e.mu.Lock()
	defer e.mu.Unlock()

	if time.Since(e.lastFetch) < jwksRefreshCooldown {
		return nil, false
	}
	keys, err := e.fetch(url)
	if err != nil {
		return nil, false
	}
	return keys, true
}

// fetch performs the outbound request. The caller must hold e.mu.
//
// Failures are cached for the cooldown so a down endpoint is not retried on
// every request, and are returned rather than served from a stale key set: a
// key that has been rotated away should stop working.
func (e *jwksEntry) fetch(url string) (*Jwks, error) {
	if e.keys == nil && e.err != nil && time.Since(e.lastFetch) < jwksRefreshCooldown {
		return nil, e.err
	}

	e.lastFetch = time.Now()
	keys, err := fetchJWKS(url)
	if err != nil {
		e.err = err
		return nil, err
	}

	e.keys, e.fetchedAt, e.err = keys, time.Now(), nil
	return keys, nil
}

func (j *Jwks) getKey(kid string) (interface{}, error) {
	for _, key := range j.Keys {
		if key.Kid != kid {
			continue
		}
		// A key published for encryption must not be used to verify
		// signatures. Keys that declare nothing are accepted, as "use" is
		// optional in RFC 7517.
		if key.Use != "" && key.Use != "sig" {
			return nil, fmt.Errorf("jwk %q is published for %q, not signature verification", kid, key.Use)
		}

		switch key.Kty {
		case "RSA":
			if key.Alg != "" && !strings.HasPrefix(key.Alg, "RS") && !strings.HasPrefix(key.Alg, "PS") {
				return nil, fmt.Errorf("jwk %q declares alg %q for an RSA key", kid, key.Alg)
			}
			return parseRSAPublicKey(key.N, key.E)
		case "EC":
			if key.Alg != "" && !strings.HasPrefix(key.Alg, "ES") {
				return nil, fmt.Errorf("jwk %q declares alg %q for an EC key", kid, key.Alg)
			}
			return parseECDSAPublicKey(key.Crv, key.X, key.Y)
		default:
			return nil, fmt.Errorf("unsupported key type: %s", key.Kty)
		}
	}
	return nil, fmt.Errorf("no matching JWK found for kid: %s", kid)
}

func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}

	pubKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}

	// A key set is trusted input, but "trusted" should not mean "unchecked":
	// a degenerate modulus is trivially factorable, and these checks are what
	// stop a partially-compromised or sloppily-served JWKS from becoming an
	// authentication bypass.
	if bits := pubKey.N.BitLen(); bits < minRSAKeyBits {
		return nil, fmt.Errorf("rsa modulus is %d bits, minimum is %d", bits, minRSAKeyBits)
	}
	if pubKey.E < 3 || pubKey.E%2 == 0 {
		return nil, fmt.Errorf("invalid rsa public exponent %d", pubKey.E)
	}
	return pubKey, nil
}

func parseECDSAPublicKey(crv, xB64, yB64 string) (*ecdsa.PublicKey, error) {
	xBytes, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, err
	}

	var curve elliptic.Curve
	switch crv {
	case curveP256:
		curve = elliptic.P256()
	case curveP384:
		curve = elliptic.P384()
	case curveP521:
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve: %s", crv)
	}

	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	// A point that is not on the named curve is not a usable public key, and
	// accepting one invites invalid-curve attacks.
	if !curve.IsOnCurve(pubKey.X, pubKey.Y) {
		return nil, fmt.Errorf("ec public key is not on curve %s", crv)
	}
	return pubKey, nil
}
