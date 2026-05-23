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

package client

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// RetryPolicy controls how the retry middleware behaves. The zero value
// disables retries (MaxAttempts <= 1).
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts including the first one.
	// Values <= 1 disable retries.
	MaxAttempts int

	// BaseDelay is the initial delay before the second attempt. Subsequent
	// attempts double the delay (exponential backoff) up to MaxDelay.
	BaseDelay time.Duration

	// MaxDelay caps the backoff delay. If zero, the delay grows unbounded.
	MaxDelay time.Duration

	// RetryOnStatus lists HTTP response status codes that trigger a retry.
	// Defaults to 408, 429, 500, 502, 503, 504 when nil and MaxAttempts > 1.
	RetryOnStatus []int

	// ShouldRetry, if non-nil, is consulted in addition to RetryOnStatus.
	// It receives the response (may be nil) and any transport error and
	// returns true to retry. It overrides RetryOnStatus when set.
	ShouldRetry func(resp *http.Response, err error) bool
}

// defaultRetryStatuses is used when RetryPolicy.RetryOnStatus is nil.
var defaultRetryStatuses = []int{
	http.StatusRequestTimeout,
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

// enabled reports whether the policy will retry at all.
func (p RetryPolicy) enabled() bool {
	return p.MaxAttempts > 1
}

func (p RetryPolicy) shouldRetry(resp *http.Response, err error) bool {
	if p.ShouldRetry != nil {
		return p.ShouldRetry(resp, err)
	}
	if err != nil {
		return true
	}
	if resp == nil {
		return false
	}
	statuses := p.RetryOnStatus
	if statuses == nil {
		statuses = defaultRetryStatuses
	}
	for _, s := range statuses {
		if resp.StatusCode == s {
			return true
		}
	}
	return false
}

// delay returns the backoff delay before attempt n (1-indexed; n>=2 only).
func (p RetryPolicy) delay(attempt int) time.Duration {
	if p.BaseDelay <= 0 {
		return 0
	}
	d := p.BaseDelay
	for i := 2; i < attempt; i++ {
		d *= 2
		if p.MaxDelay > 0 && d >= p.MaxDelay {
			return p.MaxDelay
		}
	}
	if p.MaxDelay > 0 && d > p.MaxDelay {
		return p.MaxDelay
	}
	return d
}

// retryMiddleware re-issues the request up to policy.MaxAttempts times.
// It buffers the request body once so subsequent attempts can rewind it.
func retryMiddleware(policy RetryPolicy) Middleware {
	return func(next RoundTripFunc) RoundTripFunc {
		return func(req *http.Request) (*http.Response, error) {
			if !policy.enabled() {
				return next(req)
			}

			var bodyBytes []byte
			if req.Body != nil {
				buf, err := io.ReadAll(req.Body)
				_ = req.Body.Close()
				if err != nil {
					return nil, err
				}
				bodyBytes = buf
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			var (
				resp *http.Response
				err  error
			)
			for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
				if attempt > 1 {
					if d := policy.delay(attempt); d > 0 {
						select {
						case <-time.After(d):
						case <-req.Context().Done():
							return nil, req.Context().Err()
						}
					}
					if bodyBytes != nil {
						req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
					}
				}

				resp, err = next(req)
				if !policy.shouldRetry(resp, err) {
					return resp, err
				}
				// Drain and close the previous response body before the next attempt.
				if resp != nil {
					_, _ = io.Copy(io.Discard, resp.Body)
					_ = resp.Body.Close()
				}
			}
			return resp, err
		}
	}
}
