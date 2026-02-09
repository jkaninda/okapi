package okapi

import (
	"encoding/json"
	"fmt"
	"github.com/jkaninda/okapi/okapitest"
	"net/http"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------
const validationFailed = "Validation failed"

func TestAbortWithJSON(t *testing.T) {
	payload := map[string]string{"error": "custom", "detail": "something went wrong"}

	ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
	err := ctx.AbortWithJSON(http.StatusBadRequest, payload)
	if err != nil {
		t.Fatalf("AbortWithJSON returned unexpected error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if got["error"] != "custom" || got["detail"] != "something went wrong" {
		t.Errorf("unexpected body: %v", got)
	}
}

func TestAbortWithStatus(t *testing.T) {
	ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
	err := ctx.AbortWithStatus(http.StatusForbidden, "access denied for this resource")
	if err != nil {
		t.Fatalf("AbortWithStatus returned unexpected error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal ErrorResponse: %v", err)
	}
	if resp.Code != http.StatusForbidden {
		t.Errorf("expected ErrorResponse.Code %d, got %d", http.StatusForbidden, resp.Code)
	}
	if resp.Message != http.StatusText(http.StatusForbidden) {
		t.Errorf("expected ErrorResponse.Message %q, got %q", http.StatusText(http.StatusForbidden), resp.Message)
	}
	if resp.Details != "access denied for this resource" {
		t.Errorf("expected ErrorResponse.Details %q, got %q", "access denied for this resource", resp.Details)
	}
	if resp.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set, got zero value")
	}
}

type errorAbortCase struct {
	name           string
	code           int
	defaultMessage string
	errorFn        func(ctx *Context) error
	abortFn        func(ctx *Context, msg string, err ...error) error
}

var allCases = []errorAbortCase{
	// --- 3xx ---
	{
		name: "NotModified", code: http.StatusNotModified, defaultMessage: "Not Modified",
		errorFn: func(c *Context) error { return c.ErrorNotModified("not modified") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortNotModified(msg, err...) },
	},

	// --- 4xx ---
	{
		name: "BadRequest", code: http.StatusBadRequest, defaultMessage: "Bad request",
		errorFn: func(c *Context) error { return c.ErrorBadRequest("bad request") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortBadRequest(msg, err...) },
	},
	{
		name: "Unauthorized", code: http.StatusUnauthorized, defaultMessage: "Unauthorized",
		errorFn: func(c *Context) error { return c.ErrorUnauthorized("unauthorized") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortUnauthorized(msg, err...) },
	},
	{
		name: "PaymentRequired", code: http.StatusPaymentRequired, defaultMessage: "Payment Required",
		errorFn: func(c *Context) error { return c.ErrorPaymentRequired("payment required") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortPaymentRequired(msg, err...) },
	},
	{
		name: "Forbidden", code: http.StatusForbidden, defaultMessage: "Forbidden",
		errorFn: func(c *Context) error { return c.ErrorForbidden("forbidden") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortForbidden(msg, err...) },
	},
	{
		name: "NotFound", code: http.StatusNotFound, defaultMessage: "Not Found",
		errorFn: func(c *Context) error { return c.ErrorNotFound("not found") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortNotFound(msg, err...) },
	},
	{
		name: "MethodNotAllowed", code: http.StatusMethodNotAllowed, defaultMessage: "Method Not Allowed",
		errorFn: func(c *Context) error { return c.ErrorMethodNotAllowed("method not allowed") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortMethodNotAllowed(msg, err...)
		},
	},
	{
		name: "NotAcceptable", code: http.StatusNotAcceptable, defaultMessage: "Not Acceptable",
		errorFn: func(c *Context) error { return c.ErrorNotAcceptable("not acceptable") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortNotAcceptable(msg, err...)
		},
	},
	{
		name: "ProxyAuthRequired", code: http.StatusProxyAuthRequired, defaultMessage: "Proxy Authentication Required",
		errorFn: func(c *Context) error { return c.ErrorProxyAuthRequired("proxy auth") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortProxyAuthRequired(msg, err...)
		},
	},
	{
		name: "RequestTimeout", code: http.StatusRequestTimeout, defaultMessage: "request Timeout",
		errorFn: func(c *Context) error { return c.ErrorRequestTimeout("timeout") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortRequestTimeout(msg, err...)
		},
	},
	{
		name: "Conflict", code: http.StatusConflict, defaultMessage: "Conflict",
		errorFn: func(c *Context) error { return c.ErrorConflict("conflict") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortConflict(msg, err...) },
	},
	{
		name: "Gone", code: http.StatusGone, defaultMessage: "Gone",
		errorFn: func(c *Context) error { return c.ErrorGone("gone") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortGone(msg, err...) },
	},
	{
		name: "LengthRequired", code: http.StatusLengthRequired, defaultMessage: "Length Required",
		errorFn: func(c *Context) error { return c.ErrorLengthRequired("length required") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortLengthRequired(msg, err...)
		},
	},
	{
		name: "PreconditionFailed", code: http.StatusPreconditionFailed, defaultMessage: "Precondition Failed",
		errorFn: func(c *Context) error { return c.ErrorPreconditionFailed("precondition failed") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortPreconditionFailed(msg, err...)
		},
	},
	{
		name: "RequestEntityTooLarge", code: http.StatusRequestEntityTooLarge, defaultMessage: "request Entity Too Large",
		errorFn: func(c *Context) error { return c.ErrorRequestEntityTooLarge("too large") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortRequestEntityTooLarge(msg, err...)
		},
	},
	{
		name: "RequestURITooLong", code: http.StatusRequestURITooLong, defaultMessage: "request-URI Too Long",
		errorFn: func(c *Context) error { return c.ErrorRequestURITooLong("uri too long") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortRequestURITooLong(msg, err...)
		},
	},
	{
		name: "UnsupportedMediaType", code: http.StatusUnsupportedMediaType, defaultMessage: "Unsupported Media Type",
		errorFn: func(c *Context) error { return c.ErrorUnsupportedMediaType("unsupported") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortUnsupportedMediaType(msg, err...)
		},
	},
	{
		name: "RequestedRangeNotSatisfiable", code: http.StatusRequestedRangeNotSatisfiable, defaultMessage: "Requested Range Not Satisfiable",
		errorFn: func(c *Context) error { return c.ErrorRequestedRangeNotSatisfiable("range invalid") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortRequestedRangeNotSatisfiable(msg, err...)
		},
	},
	{
		name: "ExpectationFailed", code: http.StatusExpectationFailed, defaultMessage: "Expectation Failed",
		errorFn: func(c *Context) error { return c.ErrorExpectationFailed("expectation failed") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortExpectationFailed(msg, err...)
		},
	},
	{
		name: "Teapot", code: http.StatusTeapot, defaultMessage: "I'm a teapot",
		errorFn: func(c *Context) error { return c.ErrorTeapot("I'm a teapot") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortTeapot(msg, err...) },
	},
	{
		name: "MisdirectedRequest", code: http.StatusMisdirectedRequest, defaultMessage: "Misdirected request",
		errorFn: func(c *Context) error { return c.ErrorMisdirectedRequest("misdirected") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortMisdirectedRequest(msg, err...)
		},
	},
	{
		name: "Locked", code: http.StatusLocked, defaultMessage: "Locked",
		errorFn: func(c *Context) error { return c.ErrorLocked("locked") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortLocked(msg, err...) },
	},
	{
		name: "FailedDependency", code: http.StatusFailedDependency, defaultMessage: "Failed Dependency",
		errorFn: func(c *Context) error { return c.ErrorFailedDependency("dependency failed") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortFailedDependency(msg, err...)
		},
	},
	{
		name: "TooEarly", code: http.StatusTooEarly, defaultMessage: "Too Early",
		errorFn: func(c *Context) error { return c.ErrorTooEarly("too early") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortTooEarly(msg, err...) },
	},
	{
		name: "UpgradeRequired", code: http.StatusUpgradeRequired, defaultMessage: "Upgrade Required",
		errorFn: func(c *Context) error { return c.ErrorUpgradeRequired("upgrade required") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortUpgradeRequired(msg, err...)
		},
	},
	{
		name: "PreconditionRequired", code: http.StatusPreconditionRequired, defaultMessage: "Precondition Required",
		errorFn: func(c *Context) error { return c.ErrorPreconditionRequired("precondition required") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortPreconditionRequired(msg, err...)
		},
	},
	{
		name: "TooManyRequests", code: http.StatusTooManyRequests, defaultMessage: "Too Many Requests",
		errorFn: func(c *Context) error { return c.ErrorTooManyRequests("rate limited") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortTooManyRequests(msg, err...)
		},
	},
	{
		name: "RequestHeaderFieldsTooLarge", code: http.StatusRequestHeaderFieldsTooLarge, defaultMessage: "request Header Fields Too Large",
		errorFn: func(c *Context) error { return c.ErrorRequestHeaderFieldsTooLarge("headers too large") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortRequestHeaderFieldsTooLarge(msg, err...)
		},
	},
	{
		name: "UnavailableForLegalReasons", code: http.StatusUnavailableForLegalReasons, defaultMessage: "Unavailable For Legal Reasons",
		errorFn: func(c *Context) error { return c.ErrorUnavailableForLegalReasons("legal hold") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortUnavailableForLegalReasons(msg, err...)
		},
	},

	// --- 5xx ---
	{
		name: "InternalServerError", code: http.StatusInternalServerError, defaultMessage: "Internal Server Error",
		errorFn: func(c *Context) error { return c.ErrorInternalServerError("internal error") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortInternalServerError(msg, err...)
		},
	},
	{
		name: "NotImplemented", code: http.StatusNotImplemented, defaultMessage: "Not Implemented",
		errorFn: func(c *Context) error { return c.ErrorNotImplemented("not implemented") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortNotImplemented(msg, err...)
		},
	},
	{
		name: "BadGateway", code: http.StatusBadGateway, defaultMessage: "Bad Gateway",
		errorFn: func(c *Context) error { return c.ErrorBadGateway("bad gateway") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortBadGateway(msg, err...) },
	},
	{
		name: "ServiceUnavailable", code: http.StatusServiceUnavailable, defaultMessage: "Service Unavailable",
		errorFn: func(c *Context) error { return c.ErrorServiceUnavailable("service down") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortServiceUnavailable(msg, err...)
		},
	},
	{
		name: "GatewayTimeout", code: http.StatusGatewayTimeout, defaultMessage: "Gateway Timeout",
		errorFn: func(c *Context) error { return c.ErrorGatewayTimeout("gateway timeout") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortGatewayTimeout(msg, err...)
		},
	},
	{
		name: "HTTPVersionNotSupported", code: http.StatusHTTPVersionNotSupported, defaultMessage: "HTTP version Not Supported",
		errorFn: func(c *Context) error { return c.ErrorHTTPVersionNotSupported("http version unsupported") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortHTTPVersionNotSupported(msg, err...)
		},
	},
	{
		name: "VariantAlsoNegotiates", code: http.StatusVariantAlsoNegotiates, defaultMessage: "Variant Also Negotiates",
		errorFn: func(c *Context) error { return c.ErrorVariantAlsoNegotiates("variant negotiates") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortVariantAlsoNegotiates(msg, err...)
		},
	},
	{
		name: "InsufficientStorage", code: http.StatusInsufficientStorage, defaultMessage: "Insufficient Storage",
		errorFn: func(c *Context) error { return c.ErrorInsufficientStorage("no space") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortInsufficientStorage(msg, err...)
		},
	},
	{
		name: "LoopDetected", code: http.StatusLoopDetected, defaultMessage: "Loop Detected",
		errorFn: func(c *Context) error { return c.ErrorLoopDetected("loop detected") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortLoopDetected(msg, err...) },
	},
	{
		name: "NotExtended", code: http.StatusNotExtended, defaultMessage: "Not Extended",
		errorFn: func(c *Context) error { return c.ErrorNotExtended("not extended") },
		abortFn: func(c *Context, msg string, err ...error) error { return c.AbortNotExtended(msg, err...) },
	},
	{
		name: "NetworkAuthenticationRequired", code: http.StatusNetworkAuthenticationRequired, defaultMessage: "Network Authentication Required",
		errorFn: func(c *Context) error { return c.ErrorNetworkAuthenticationRequired("network auth") },
		abortFn: func(c *Context, msg string, err ...error) error {
			return c.AbortNetworkAuthenticationRequired(msg, err...)
		},
	},
}

func TestErrorMethods(t *testing.T) {
	for _, tc := range allCases {
		t.Run(tc.name+"_Error", func(t *testing.T) {
			ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
			if err := tc.errorFn(ctx); err != nil {
				t.Fatalf("Error method returned unexpected error: %v", err)
			}

			if rec.Code != tc.code {
				t.Errorf("expected status %d, got %d", tc.code, rec.Code)
			}

			body := strings.TrimSpace(rec.Body.String())
			if body == "" {
				t.Error("expected non-empty response body")
			}
		})
	}
}

func TestAbortMethods_WithError(t *testing.T) {
	for _, tc := range allCases {
		t.Run(tc.name+"_Abort_WithError", func(t *testing.T) {
			ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
			customMsg := fmt.Sprintf("custom message for %s", tc.name)
			testErr := fmt.Errorf("underlying error for %s", tc.name)

			if err := tc.abortFn(ctx, customMsg, testErr); err != nil {
				t.Fatalf("Abort method returned unexpected error: %v", err)
			}

			if rec.Code != tc.code {
				t.Errorf("expected status %d, got %d", tc.code, rec.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal ErrorResponse: %v\nbody: %s", err, rec.Body.String())
			}

			if resp.Code != tc.code {
				t.Errorf("ErrorResponse.Code: expected %d, got %d", tc.code, resp.Code)
			}
			if resp.Message != customMsg {
				t.Errorf("ErrorResponse.Message: expected %q, got %q", customMsg, resp.Message)
			}
			if resp.Details != testErr.Error() {
				t.Errorf("ErrorResponse.Details: expected %q, got %q", testErr.Error(), resp.Details)
			}
			if resp.Timestamp.IsZero() {
				t.Error("expected Timestamp to be populated")
			}
		})
	}
}

func TestAbortMethods_WithoutError(t *testing.T) {
	for _, tc := range allCases {
		t.Run(tc.name+"_Abort_NoError", func(t *testing.T) {
			ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
			customMsg := fmt.Sprintf("message only for %s", tc.name)

			if err := tc.abortFn(ctx, customMsg); err != nil {
				t.Fatalf("Abort method returned unexpected error: %v", err)
			}

			if rec.Code != tc.code {
				t.Errorf("expected status %d, got %d", tc.code, rec.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal ErrorResponse: %v\nbody: %s", err, rec.Body.String())
			}

			if resp.Message != customMsg {
				t.Errorf("ErrorResponse.Message: expected %q, got %q", customMsg, resp.Message)
			}

			if resp.Details != customMsg {
				t.Errorf("ErrorResponse.Details: expected %q, got %q", customMsg, resp.Details)
			}
		})
	}
}

func TestAbortMethods_EmptyMessage(t *testing.T) {
	for _, tc := range allCases {
		t.Run(tc.name+"_Abort_EmptyMsg", func(t *testing.T) {
			ctx, rec := NewTestContext(http.MethodGet, "/test", nil)

			if err := tc.abortFn(ctx, ""); err != nil {
				t.Fatalf("Abort method returned unexpected error: %v", err)
			}

			if rec.Code != tc.code {
				t.Errorf("expected status %d, got %d", tc.code, rec.Code)
			}

			var resp ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal ErrorResponse: %v\nbody: %s", err, rec.Body.String())
			}

			// Empty msg triggers fallback to defaultMessage inside abortWithStatus.
			if resp.Details != tc.defaultMessage {
				t.Errorf("ErrorResponse.Details: expected default %q, got %q", tc.defaultMessage, resp.Details)
			}
		})
	}
}

func TestAbort(t *testing.T) {
	ctx, rec := NewTestContext(http.MethodGet, "/test", nil)
	rootErr := fmt.Errorf("database connection lost")

	if err := ctx.Abort(rootErr); err != nil {
		t.Fatalf("Abort returned unexpected error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal ErrorResponse: %v\nbody: %s", err, rec.Body.String())
	}
	if resp.Code != http.StatusInternalServerError {
		t.Errorf("ErrorResponse.Code: expected %d, got %d", http.StatusInternalServerError, resp.Code)
	}
	if resp.Message != "Internal Server Error" {
		t.Errorf("ErrorResponse.Message: expected %q, got %q", "Internal Server Error", resp.Message)
	}
}

func TestErrorUnprocessableEntity(t *testing.T) {
	ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
	ctx.request.Header.Set("Content-Type", "application/json")

	if err := ctx.ErrorUnprocessableEntity(validationFailed); err != nil {
		t.Fatalf("ErrorUnprocessableEntity returned unexpected error: %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}
}

func TestAbortValidationError(t *testing.T) {
	t.Run("with_message_and_error", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		testErr := fmt.Errorf("validation: X")
		if err := ctx.AbortValidationError("field X is invalid", testErr); err != nil {
			t.Fatalf("AbortValidationError returned unexpected error: %v", err)
		}

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}

		var resp ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ErrorResponse: %v", err)
		}
		if resp.Message != "field X is invalid" {
			t.Errorf("ErrorResponse.Message: expected %q, got %q", "field X is invalid", resp.Message)
		}
		if resp.Details != testErr.Error() {
			t.Errorf("ErrorResponse.Details: expected %q, got %q", testErr.Error(), resp.Details)
		}
	})

	t.Run("without_error", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		if err := ctx.AbortValidationError("missing required field"); err != nil {
			t.Fatalf("AbortValidationError returned unexpected error: %v", err)
		}

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}

		var resp ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ErrorResponse: %v", err)
		}
		if resp.Details != "missing required field" {
			t.Errorf("ErrorResponse.Details: expected %q, got %q", "missing required field", resp.Details)
		}
	})
}

func TestAbortValidationErrors(t *testing.T) {
	t.Run("with_custom_message", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		errs := []ValidationError{
			{Field: "email", Message: "invalid email format"},
			{Field: "name", Message: "name is required"},
		}

		if err := ctx.AbortValidationErrors(errs, "Input validation failed"); err != nil {
			t.Fatalf("AbortValidationErrors returned unexpected error: %v", err)
		}

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ValidationErrorResponse: %v\nbody: %s", err, rec.Body.String())
		}
		if resp.Code != http.StatusUnprocessableEntity {
			t.Errorf("ErrorResponse.Code: expected %d, got %d", http.StatusUnprocessableEntity, resp.Code)
		}
		if resp.Message != "Input validation failed" {
			t.Errorf("ErrorResponse.Message: expected %q, got %q", "Input validation failed", resp.Message)
		}
		if resp.Timestamp.IsZero() {
			t.Error("expected Timestamp to be populated")
		}
		if len(resp.Errors) != 2 {
			t.Fatalf("expected 2 validation errors, got %d", len(resp.Errors))
		}
		if resp.Errors[0].Field != "email" || resp.Errors[0].Message != "invalid email format" {
			t.Errorf("unexpected first validation error: %+v", resp.Errors[0])
		}
		if resp.Errors[1].Field != "name" || resp.Errors[1].Message != "name is required" {
			t.Errorf("unexpected second validation error: %+v", resp.Errors[1])
		}
	})

	t.Run("default_message_when_none_provided", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		errs := []ValidationError{
			{Field: "age", Message: "must be a positive integer"},
		}

		if err := ctx.AbortValidationErrors(errs); err != nil {
			t.Fatalf("AbortValidationErrors returned unexpected error: %v", err)
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ValidationErrorResponse: %v", err)
		}
		if resp.Message != validationFailed {
			t.Errorf("ErrorResponse.Message: expected %q, got %q", validationFailed, resp.Message)
		}
		if len(resp.Errors) != 1 {
			t.Fatalf("expected 1 validation error, got %d", len(resp.Errors))
		}
	})

	t.Run("empty_string_message_uses_default", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		errs := []ValidationError{
			{Field: "phone", Message: "invalid format"},
		}

		if err := ctx.AbortValidationErrors(errs, ""); err != nil {
			t.Fatalf("AbortValidationErrors returned unexpected error: %v", err)
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ValidationErrorResponse: %v", err)
		}
		if resp.Message != validationFailed {
			t.Errorf("ErrorResponse.Message: expected %q, got %q", validationFailed, resp.Message)
		}
	})

	t.Run("empty_errors_slice", func(t *testing.T) {
		ctx, rec := NewTestContext(http.MethodPost, "/test", strings.NewReader(`{}`))
		ctx.request.Header.Set("Content-Type", "application/json")

		if err := ctx.AbortValidationErrors([]ValidationError{}); err != nil {
			t.Fatalf("AbortValidationErrors returned unexpected error: %v", err)
		}

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal ValidationErrorResponse: %v", err)
		}
		if len(resp.Errors) != 0 {
			t.Errorf("expected 0 validation errors, got %d", len(resp.Errors))
		}
	})
}
func TestProblemDetailWithCustomFields(t *testing.T) {
	app := NewTestServerOn(t, 8001)

	app.With(
		WithProblemDetailErrorHandler(&ErrorHandlerConfig{
			Format:           ErrorFormatProblemJSON,
			TypePrefix:       "https://api.example.com/errors/",
			IncludeInstance:  true,
			IncludeTimestamp: true,
			CustomFields: map[string]any{
				"api_version": "v1.0.0",
				"support_url": "https://support.example.com",
				"environment": "production",
			},
		}),
	)

	app.Get("/test", func(c *Context) error {
		return c.AbortNotFound("Resource not found")
	})

	okapitest.GET(t, app.BaseURL+"/test").ExpectStatusNotFound().ExpectBodyContains("api_version").
		ExpectBodyContains("support_url").ExpectBodyContains("support_url").
		ExpectBodyContains("environment").ExpectBodyContains("type").
		ExpectBodyContains("status").ExpectBodyContains("timestamp")

}
func TestOkapi_WithErrorHandler(t *testing.T) {
	app := NewTestServerOn(t, 8002)

	app.With(
		WithErrorHandler(func(c *Context, code int, message string, err error) error {
			return c.JSON(code, map[string]any{
				"status": "error",
				"code":   code,
				"msg":    message,
			})
		}),
	)

	app.Get("/test", func(c *Context) error {
		return c.AbortNotFound("Resource not found")
	})

	okapitest.GET(t, app.BaseURL+"/test").ExpectStatusNotFound().ExpectBodyContains("status").
		ExpectBodyContains("code").ExpectBodyContains("msg")
}
func TestOkapi_WithSimpleProblemDetailErrorHandler(t *testing.T) {
	app := NewTestServerOn(t, 8003)

	app.WithSimpleProblemDetailErrorHandler()

	okapitest.GET(t, app.BaseURL+"/test").ExpectStatusNotFound().ExpectBodyContains("page not found")
}

func TestProblemDetailMarshalJSON(t *testing.T) {
	problem := ProblemDetail{
		Type:   "https://example.com/errors/not-found",
		Title:  "Not Found",
		Status: 404,
		Detail: "Resource not found",
		Extensions: map[string]any{
			"api_version": "v1.0.0",
			"support_url": "https://support.example.com",
			"retry_after": 300,
		},
	}

	data, err := json.Marshal(problem)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	fmt.Println("Marshaled:", string(data))

	var result map[string]any
	if err = json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify extensions are present
	if result["api_version"] != "v1.0.0" {
		t.Errorf("Expected api_version, got %v", result["api_version"])
	}
	if result["support_url"] != "https://support.example.com" {
		t.Errorf("Expected support_url, got %v", result["support_url"])
	}
	if result["retry_after"] != float64(300) {
		t.Errorf("Expected retry_after, got %v", result["retry_after"])
	}
}
