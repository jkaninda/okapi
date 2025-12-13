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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// realIP extracts the real IP address of the client from the HTTP Request.
func realIP(r *http.Request) string {
	// Check the X-Forwarded-For header for the client IP.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the comma-separated list.
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check the X-Real-IP header as a fallback.
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}

	// Use the remote address if headers are not set.
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	// Return the raw remote address as a last resort.
	return r.RemoteAddr
}

// normalizeRoutePath ensures a clean path starting with '/'
// and converts ':param' to '{param}' and /* or /*any to /{any:.*} for mux compatibility.
func normalizeRoutePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Remove double slashes
	path = strings.ReplaceAll(path, "//", "/")

	// Convert /*any or /* to /{any:.*}
	if strings.HasSuffix(path, "/*") {
		path = strings.TrimSuffix(path, "/*") + "/{any:.*}"
	} else {
		path = wildcardRegex.ReplaceAllString(path, "/{any:.*}")
	}

	// Process each segment to convert :param or :param:type to {param}
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			// Remove leading :
			segment = strings.TrimPrefix(segment, ":")
			// Extract only the parameter name (ignore type if present)
			paramName := strings.SplitN(segment, ":", 2)[0]
			segments[i] = fmt.Sprintf("{%s}", paramName)
		} else if strings.HasPrefix(segment, "{") && strings.Contains(segment, ":") && !strings.Contains(segment, ".*") {
			// Handle {id:int} -> {id} (but preserve {any:.*})
			segment = strings.TrimPrefix(segment, "{")
			segment = strings.TrimSuffix(segment, "}")
			// Extract only the parameter name (ignore type if present)
			paramName := strings.SplitN(segment, ":", 2)[0]
			segments[i] = fmt.Sprintf("{%s}", paramName)
		}
	}

	return strings.Join(segments, "/")
}

// ValidateAddr checks if the entrypoint address is valid.
// A valid entrypoint address should be in the format ":<port>" or "<IP>:<port>",
// where <IP> is a valid IP address and <port> is a valid port number (1-65535).
func ValidateAddr(addr string) bool {
	// Split the addr into IP and port parts
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}

	// If the host is empty, it means the addr is in the format ":<port>"
	// Otherwise, validate the IP address
	if host != "" {
		ip := net.ParseIP(host)
		if ip == nil {
			return false
		}
	}

	// Convert the port string to an integer
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}

	// Check if the port is within the valid range
	if port < 1 || port > 65535 {
		return false
	}
	return true
}

func joinPaths(basePath, path string) string {
	// Ensure both segments have exactly one slash between them
	joined := strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(path, "/")

	// Normalize any remaining double slashes
	joined = strings.ReplaceAll(joined, "//", "/")

	// Ensure leading slash
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}
func allowedOrigin(allowed []string, origin string) bool {
	for _, o := range allowed {
		if o == "*" || o == origin {
			return true
		}
	}
	return false

}

// LoadTLSConfig creates a TLS configuration from certificate and key files
// Parameters:
//   - certFile: Path to the certificate file (PEM format)
//   - keyFile: Path to the private key file (PEM format)
//   - caFile: Optional path to CA certificate file for client verification (set to "" to disable)
//   - clientAuth: Whether to require client certificate verification
//
// Returns:
//   - *tls.Config configured with the certificate and settings
//   - error if any occurred during loading
func LoadTLSConfig(certFile, keyFile, caFile string, clientAuth bool) (*tls.Config, error) {
	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12, // Enforce minimum TLS version 1.2
	}

	// If caFile is provided, set up client certificate verification
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			_, _ = fmt.Fprintf(defaultErrorWriter, "Warning: failed to append CA certs from PEM")
		}

		config.ClientCAs = caCertPool
		if clientAuth {
			config.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			config.ClientAuth = tls.VerifyClientCertIfGiven
		}
	}

	return config, nil
}

// LoadJWKSFromFile loads a JWKS (JSON Web Key Set) from a file path or a base64-encoded string.
//
// If the input is a base64-encoded string, it decodes it.
// Otherwise, it treats the input as a file path and attempts to open and read it.
// The function returns a parsed *Jwks struct or an error if any step fails.
func LoadJWKSFromFile(jwksInput string) (*Jwks, error) {
	// Trim surrounding whitespace from the input string
	trimmed := strings.TrimSpace(jwksInput)

	var reader io.Reader

	if isBase64(trimmed) {
		// If the input looks like base64, attempt to decode it
		decoded, err := base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 JWKS content: %w", err)
		}
		reader = bytes.NewReader(decoded)
	} else {
		file, err := os.Open(trimmed)
		if err != nil {
			return nil, fmt.Errorf("failed to open JWKS file: %w", err)
		}

		defer func() {
			if cerr := file.Close(); cerr != nil {
				_, _ = fmt.Fprint(defaultErrorWriter, "Error closing JWKS file", "error", cerr)
			}
		}()

		reader = file
	}

	// Decode the JSON content into a Jwks struct
	var keySet Jwks
	if err := json.NewDecoder(reader).Decode(&keySet); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS content: %w", err)
	}
	return &keySet, nil
}

// isBase64 checks if the input is valid Base64-encoded content.
func isBase64(input string) bool {
	_, err := base64.StdEncoding.DecodeString(input)
	return err == nil
}
