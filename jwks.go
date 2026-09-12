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
	"errors"
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
	// retried no more often than that, whether or not an older key set is
	// still cached. Without it an attacker inventing kids drives one outbound
	// request per inbound request, and an identity provider outage turns every
	// request into a fetch attempt.
	jwksRefreshCooldown = time.Minute

	// jwksStaleGrace is how long past its TTL a key set keeps being served
	// while refreshes fail. It lets authentication ride out a brief identity
	// provider outage, but it is bounded: a key the issuer has withdrawn must
	// eventually stop verifying tokens even if the endpoint never recovers.
	// With the default TTL, no set is trusted more than 30 minutes after it
	// was fetched.
	jwksStaleGrace = 15 * time.Minute

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

var errJWKSUnavailable = errors.New("jwks key set unavailable")

type jwksEntry struct {
	mu        sync.Mutex
	keys      *Jwks
	err       error         // cause of the latest fetch failure, cleared on success
	fetchedAt time.Time     // when keys was fetched
	lastFetch time.Time     // when the latest fetch started, successful or not
	inflight  chan struct{} // non-nil while a fetch runs; closed when it completes
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

	done := e.inflight
	if done == nil && time.Since(e.lastFetch) >= min(ttl, jwksRefreshCooldown) {
		done = e.startFetch(url)
	}
	if e.keys != nil && time.Since(e.fetchedAt) < ttl+jwksStaleGrace {
		return e.keys, nil
	}
	if done == nil {
		// Cooling down after a failed fetch, with no set recent enough to serve.
		return nil, e.failure()
	}

	e.await(done)
	if e.err != nil || e.keys == nil {
		return nil, e.failure()
	}
	return e.keys, nil
}

func jwksRefresh(url string) (*Jwks, bool) {
	e := jwksEntryFor(url)
	e.mu.Lock()
	defer e.mu.Unlock()

	done := e.inflight
	if done == nil {
		if time.Since(e.lastFetch) < jwksRefreshCooldown {

			return e.keys, e.err == nil && e.keys != nil
		}
		done = e.startFetch(url)
	}

	e.await(done)
	if e.err != nil || e.keys == nil {
		return nil, false
	}
	return e.keys, true
}

// startFetch performs the outbound request in the background and returns a
// channel that is closed when it completes.
func (e *jwksEntry) startFetch(url string) chan struct{} {
	done := make(chan struct{})
	e.inflight = done
	e.lastFetch = time.Now()

	go func() {
		keys, err := fetchJWKS(url)

		e.mu.Lock()
		defer e.mu.Unlock()
		if err != nil {
			e.err = err
		} else {
			e.keys, e.fetchedAt, e.err = keys, time.Now(), nil
		}
		e.inflight = nil
		close(done)
	}()
	return done
}

// await waits for the fetch behind done to complete.
func (e *jwksEntry) await(done chan struct{}) {
	e.mu.Unlock()
	<-done
	e.mu.Lock()
}

// failure returns the cause of the latest fetch failure. The caller must hold
// e.mu.
func (e *jwksEntry) failure() error {
	if e.err != nil {
		return e.err
	}
	return errJWKSUnavailable
}

func (j *Jwks) getKey(kid string) (interface{}, error) {
	for _, key := range j.Keys {
		if key.Kid != kid {
			continue
		}

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
