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
	"fmt"
	"net/http"
	"time"
)

// ErrorResponse represents a standardized error response structure
type ErrorResponse struct {
	Code      int       `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ValidationError represents validation error details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

// ValidationErrorResponse extends ErrorResponse for validation errors
type ValidationErrorResponse struct {
	ErrorResponse
	Errors []ValidationError `json:"errors"`
}

// ************* Context Errors ****************

// ********** Core Error Methods *************

// Error writes a basic error response with the given status code and message.
func (c *Context) Error(code int, message string) error {
	c.response.WriteHeader(code)
	_, err := c.response.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write error response: %w", err)
	}
	return nil
}

// AbortWithError writes a standardized error response and stops execution.
func (c *Context) AbortWithError(code int, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return c.JSON(code, ErrorResponse{
		Code:      code,
		Message:   http.StatusText(code),
		Details:   details,
		Timestamp: time.Now(),
	})
}

// abortWithError writes a standardized error response and stops execution.
func (c *Context) abortWithError(code int, msg string, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return c.JSON(code, ErrorResponse{
		Code:      code,
		Message:   msg,
		Details:   details,
		Timestamp: time.Now(),
	})
}

// AbortWithJSON writes a custom JSON error response.
func (c *Context) AbortWithJSON(code int, jsonObj interface{}) error {
	return c.JSON(code, jsonObj)
}

// AbortWithStatus writes an error response with status code and custom message.
func (c *Context) AbortWithStatus(code int, message string) error {
	return c.JSON(code, ErrorResponse{
		Code:      code,
		Message:   http.StatusText(code),
		Details:   message,
		Timestamp: time.Now(),
	})
}

// ********** Helper Method *************

// abortWithStatus is a helper for consistent status-based error responses
func (c *Context) abortWithStatus(code int, defaultMsg string, msg string, err ...error) error {
	var internalErr error
	message := defaultMsg
	if len(msg) > 0 {
		message = msg
	}
	// TODO
	if len(err) > 0 && err[0] != nil {
		internalErr = err[0]
	} else {
		internalErr = errors.New(message)
	}
	return c.abortWithError(code, message, internalErr)
}

// ********** 4xx Client Error Methods *************

// ErrorBadRequest writes a 400 Bad request response.
func (c *Context) ErrorBadRequest(message any) error {
	return c.JSON(http.StatusBadRequest, message)
}

// AbortBadRequest writes a standardized 400 Bad request response.
func (c *Context) AbortBadRequest(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusBadRequest, "Bad request", msg, err...)
}

// ErrorUnauthorized writes a 401 Unauthorized response.
func (c *Context) ErrorUnauthorized(message any) error {
	return c.JSON(http.StatusUnauthorized, message)
}

// AbortUnauthorized writes a standardized 401 Unauthorized response.
func (c *Context) AbortUnauthorized(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusUnauthorized, "Unauthorized", msg, err...)
}

// ErrorPaymentRequired writes a 402 Payment Required response.
func (c *Context) ErrorPaymentRequired(message any) error {
	return c.JSON(http.StatusPaymentRequired, message)
}

// AbortPaymentRequired writes a standardized 402 Payment Required response.
func (c *Context) AbortPaymentRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusPaymentRequired, "Payment Required", msg, err...)
}

// ErrorForbidden writes a 403 Forbidden response.
func (c *Context) ErrorForbidden(message any) error {
	return c.JSON(http.StatusForbidden, message)
}

// AbortForbidden writes a standardized 403 Forbidden response.
func (c *Context) AbortForbidden(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusForbidden, "Forbidden", msg, err...)
}

// ErrorNotFound writes a 404 Not Found response.
func (c *Context) ErrorNotFound(message any) error {
	return c.JSON(http.StatusNotFound, message)
}

// AbortNotFound writes a standardized 404 Not Found response.
func (c *Context) AbortNotFound(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNotFound, "Not Found", msg, err...)
}

// ErrorMethodNotAllowed writes a 405 Method Not Allowed response.
func (c *Context) ErrorMethodNotAllowed(message any) error {
	return c.JSON(http.StatusMethodNotAllowed, message)
}

// AbortMethodNotAllowed writes a standardized 405 Method Not Allowed response.
func (c *Context) AbortMethodNotAllowed(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusMethodNotAllowed, "Method Not Allowed", msg, err...)
}

// ErrorNotAcceptable writes a 406 Not Acceptable response.
func (c *Context) ErrorNotAcceptable(message any) error {
	return c.JSON(http.StatusNotAcceptable, message)
}

// AbortNotAcceptable writes a standardized 406 Not Acceptable response.
func (c *Context) AbortNotAcceptable(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNotAcceptable, "Not Acceptable", msg, err...)
}

// ErrorProxyAuthRequired writes a 407 Proxy Authentication Required response.
func (c *Context) ErrorProxyAuthRequired(message any) error {
	return c.JSON(http.StatusProxyAuthRequired, message)
}

// AbortProxyAuthRequired writes a standardized 407 Proxy Authentication Required response.
func (c *Context) AbortProxyAuthRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusProxyAuthRequired, "Proxy Authentication Required", msg, err...)
}

// ErrorRequestTimeout writes a 408 request Timeout response.
func (c *Context) ErrorRequestTimeout(message any) error {
	return c.JSON(http.StatusRequestTimeout, message)
}

// AbortRequestTimeout writes a standardized 408 request Timeout response.
func (c *Context) AbortRequestTimeout(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusRequestTimeout, "request Timeout", msg, err...)
}

// ErrorConflict writes a 409 Conflict response.
func (c *Context) ErrorConflict(message any) error {
	return c.JSON(http.StatusConflict, message)
}

// AbortConflict writes a standardized 409 Conflict response.
func (c *Context) AbortConflict(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusConflict, "Conflict", msg, err...)
}

// ErrorGone writes a 410 Gone response.
func (c *Context) ErrorGone(message any) error {
	return c.JSON(http.StatusGone, message)
}

// AbortGone writes a standardized 410 Gone response.
func (c *Context) AbortGone(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusGone, "Gone", msg, err...)
}

// ErrorLengthRequired writes a 411 Length Required response.
func (c *Context) ErrorLengthRequired(message any) error {
	return c.JSON(http.StatusLengthRequired, message)
}

// AbortLengthRequired writes a standardized 411 Length Required response.
func (c *Context) AbortLengthRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusLengthRequired, "Length Required", msg, err...)
}

// ErrorPreconditionFailed writes a 412 Precondition Failed response.
func (c *Context) ErrorPreconditionFailed(message any) error {
	return c.JSON(http.StatusPreconditionFailed, message)
}

// AbortPreconditionFailed writes a standardized 412 Precondition Failed response.
func (c *Context) AbortPreconditionFailed(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusPreconditionFailed, "Precondition Failed", msg, err...)
}

// ErrorRequestEntityTooLarge writes a 413 request Entity Too Large response.
func (c *Context) ErrorRequestEntityTooLarge(message any) error {
	return c.JSON(http.StatusRequestEntityTooLarge, message)
}

// AbortRequestEntityTooLarge writes a standardized 413 request Entity Too Large response.
func (c *Context) AbortRequestEntityTooLarge(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusRequestEntityTooLarge, "request Entity Too Large", msg, err...)
}

// ErrorRequestURITooLong writes a 414 request-URI Too Long response.
func (c *Context) ErrorRequestURITooLong(message any) error {
	return c.JSON(http.StatusRequestURITooLong, message)
}

// AbortRequestURITooLong writes a standardized 414 request-URI Too Long response.
func (c *Context) AbortRequestURITooLong(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusRequestURITooLong, "request-URI Too Long", msg, err...)
}

// ErrorUnsupportedMediaType writes a 415 Unsupported Media Type response.
func (c *Context) ErrorUnsupportedMediaType(message any) error {
	return c.JSON(http.StatusUnsupportedMediaType, message)
}

// AbortUnsupportedMediaType writes a standardized 415 Unsupported Media Type response.
func (c *Context) AbortUnsupportedMediaType(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusUnsupportedMediaType, "Unsupported Media Type", msg, err...)
}

// ErrorRequestedRangeNotSatisfiable writes a 416 Requested Range Not Satisfiable response.
func (c *Context) ErrorRequestedRangeNotSatisfiable(message any) error {
	return c.JSON(http.StatusRequestedRangeNotSatisfiable, message)
}

// AbortRequestedRangeNotSatisfiable writes a standardized 416 Requested Range Not Satisfiable response.
func (c *Context) AbortRequestedRangeNotSatisfiable(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusRequestedRangeNotSatisfiable, "Requested Range Not Satisfiable", msg, err...)
}

// ErrorExpectationFailed writes a 417 Expectation Failed response.
func (c *Context) ErrorExpectationFailed(message any) error {
	return c.JSON(http.StatusExpectationFailed, message)
}

// AbortExpectationFailed writes a standardized 417 Expectation Failed response.
func (c *Context) AbortExpectationFailed(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusExpectationFailed, "Expectation Failed", msg, err...)
}

// ErrorTeapot writes a 418 I'm a teapot response (RFC 2324).
func (c *Context) ErrorTeapot(message any) error {
	return c.JSON(http.StatusTeapot, message)
}

// AbortTeapot writes a standardized 418 I'm a teapot response.
func (c *Context) AbortTeapot(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusTeapot, "I'm a teapot", msg, err...)
}

// ErrorMisdirectedRequest writes a 421 Misdirected request response.
func (c *Context) ErrorMisdirectedRequest(message any) error {
	return c.JSON(http.StatusMisdirectedRequest, message)
}

// AbortMisdirectedRequest writes a standardized 421 Misdirected request response.
func (c *Context) AbortMisdirectedRequest(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusMisdirectedRequest, "Misdirected request", msg, err...)
}

// ErrorUnprocessableEntity writes a 422 Unprocessable Entity response.
func (c *Context) ErrorUnprocessableEntity(message any) error {
	return c.JSON(http.StatusUnprocessableEntity, message)
}

// AbortValidationError writes a standardized 422 Unprocessable Entity response.
func (c *Context) AbortValidationError(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusUnprocessableEntity, "Unprocessable Entity", msg, err...)
}

// AbortValidationErrors writes a detailed validation error response.
func (c *Context) AbortValidationErrors(errors []ValidationError, msg ...string) error {
	message := "Validation failed"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}

	return c.JSON(http.StatusUnprocessableEntity, ValidationErrorResponse{
		ErrorResponse: ErrorResponse{
			Code:      http.StatusUnprocessableEntity,
			Message:   message,
			Timestamp: time.Now(),
		},
		Errors: errors,
	})
}

// ErrorNotModified writes a 304 Not Modified response.
func (c *Context) ErrorNotModified(message any) error {
	return c.JSON(http.StatusNotModified, message)
}

// AbortNotModified writes a standardized 304 Not Modified response.
func (c *Context) AbortNotModified(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNotModified, "Not Modified", msg, err...)
}

// ErrorLocked writes a 423 Locked response.
func (c *Context) ErrorLocked(message any) error {
	return c.JSON(http.StatusLocked, message)
}

// AbortLocked writes a standardized 423 Locked response.
func (c *Context) AbortLocked(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusLocked, "Locked", msg, err...)
}

// ErrorFailedDependency writes a 424 Failed Dependency response.
func (c *Context) ErrorFailedDependency(message any) error {
	return c.JSON(http.StatusFailedDependency, message)
}

// AbortFailedDependency writes a standardized 424 Failed Dependency response.
func (c *Context) AbortFailedDependency(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusFailedDependency, "Failed Dependency", msg, err...)
}

// ErrorTooEarly writes a 425 Too Early response.
func (c *Context) ErrorTooEarly(message any) error {
	return c.JSON(http.StatusTooEarly, message)
}

// AbortTooEarly writes a standardized 425 Too Early response.
func (c *Context) AbortTooEarly(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusTooEarly, "Too Early", msg, err...)
}

// ErrorUpgradeRequired writes a 426 Upgrade Required response.
func (c *Context) ErrorUpgradeRequired(message any) error {
	return c.JSON(http.StatusUpgradeRequired, message)
}

// AbortUpgradeRequired writes a standardized 426 Upgrade Required response.
func (c *Context) AbortUpgradeRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusUpgradeRequired, "Upgrade Required", msg, err...)
}

// ErrorPreconditionRequired writes a 428 Precondition Required response.
func (c *Context) ErrorPreconditionRequired(message any) error {
	return c.JSON(http.StatusPreconditionRequired, message)
}

// AbortPreconditionRequired writes a standardized 428 Precondition Required response.
func (c *Context) AbortPreconditionRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusPreconditionRequired, "Precondition Required", msg, err...)
}

// ErrorTooManyRequests writes a 429 Too Many Requests response.
func (c *Context) ErrorTooManyRequests(message any) error {
	return c.JSON(http.StatusTooManyRequests, message)
}

// AbortTooManyRequests writes a standardized 429 Too Many Requests response.
func (c *Context) AbortTooManyRequests(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusTooManyRequests, "Too Many Requests", msg, err...)
}

// ErrorRequestHeaderFieldsTooLarge writes a 431 request Header Fields Too Large response.
func (c *Context) ErrorRequestHeaderFieldsTooLarge(message any) error {
	return c.JSON(http.StatusRequestHeaderFieldsTooLarge, message)
}

// AbortRequestHeaderFieldsTooLarge writes a standardized 431 request Header Fields Too Large response.
func (c *Context) AbortRequestHeaderFieldsTooLarge(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusRequestHeaderFieldsTooLarge, "request Header Fields Too Large", msg, err...)
}

// ErrorUnavailableForLegalReasons writes a 451 Unavailable For Legal Reasons response.
func (c *Context) ErrorUnavailableForLegalReasons(message any) error {
	return c.JSON(http.StatusUnavailableForLegalReasons, message)
}

// AbortUnavailableForLegalReasons writes a standardized 451 Unavailable For Legal Reasons response.
func (c *Context) AbortUnavailableForLegalReasons(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusUnavailableForLegalReasons, "Unavailable For Legal Reasons", msg, err...)
}

// ********** 5xx Server Error Methods *************

// ErrorInternalServerError writes a 500 Internal Server Error response.
func (c *Context) ErrorInternalServerError(message any) error {
	return c.JSON(http.StatusInternalServerError, message)
}

// Abort writes a standardized 500 Internal Server Error response.
func (c *Context) Abort(err error) error {
	return c.abortWithError(http.StatusInternalServerError, "Internal Server Error", err)
}

// AbortInternalServerError writes a standardized 500 Internal Server Error response.
func (c *Context) AbortInternalServerError(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusInternalServerError, "Internal Server Error", msg, err...)
}

// ErrorNotImplemented writes a 501 Not Implemented response.
func (c *Context) ErrorNotImplemented(message any) error {
	return c.JSON(http.StatusNotImplemented, message)
}

// AbortNotImplemented writes a standardized 501 Not Implemented response.
func (c *Context) AbortNotImplemented(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNotImplemented, "Not Implemented", msg, err...)
}

// ErrorBadGateway writes a 502 Bad Gateway response.
func (c *Context) ErrorBadGateway(message any) error {
	return c.JSON(http.StatusBadGateway, message)
}

// AbortBadGateway writes a standardized 502 Bad Gateway response.
func (c *Context) AbortBadGateway(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusBadGateway, "Bad Gateway", msg, err...)
}

// ErrorServiceUnavailable writes a 503 Service Unavailable response.
func (c *Context) ErrorServiceUnavailable(message any) error {
	return c.JSON(http.StatusServiceUnavailable, message)
}

// AbortServiceUnavailable writes a standardized 503 Service Unavailable response.
func (c *Context) AbortServiceUnavailable(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusServiceUnavailable, "Service Unavailable", msg, err...)
}

// ErrorGatewayTimeout writes a 504 Gateway Timeout response.
func (c *Context) ErrorGatewayTimeout(message any) error {
	return c.JSON(http.StatusGatewayTimeout, message)
}

// AbortGatewayTimeout writes a standardized 504 Gateway Timeout response.
func (c *Context) AbortGatewayTimeout(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusGatewayTimeout, "Gateway Timeout", msg, err...)
}

// ErrorHTTPVersionNotSupported writes a 505 HTTP version Not Supported response.
func (c *Context) ErrorHTTPVersionNotSupported(message any) error {
	return c.JSON(http.StatusHTTPVersionNotSupported, message)
}

// AbortHTTPVersionNotSupported writes a standardized 505 HTTP version Not Supported response.
func (c *Context) AbortHTTPVersionNotSupported(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusHTTPVersionNotSupported, "HTTP version Not Supported", msg, err...)
}

// ErrorVariantAlsoNegotiates writes a 506 Variant Also Negotiates response.
func (c *Context) ErrorVariantAlsoNegotiates(message any) error {
	return c.JSON(http.StatusVariantAlsoNegotiates, message)
}

// AbortVariantAlsoNegotiates writes a standardized 506 Variant Also Negotiates response.
func (c *Context) AbortVariantAlsoNegotiates(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusVariantAlsoNegotiates, "Variant Also Negotiates", msg, err...)
}

// ErrorInsufficientStorage writes a 507 Insufficient Storage response.
func (c *Context) ErrorInsufficientStorage(message any) error {
	return c.JSON(http.StatusInsufficientStorage, message)
}

// AbortInsufficientStorage writes a standardized 507 Insufficient Storage response.
func (c *Context) AbortInsufficientStorage(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusInsufficientStorage, "Insufficient Storage", msg, err...)
}

// ErrorLoopDetected writes a 508 Loop Detected response.
func (c *Context) ErrorLoopDetected(message any) error {
	return c.JSON(http.StatusLoopDetected, message)
}

// AbortLoopDetected writes a standardized 508 Loop Detected response.
func (c *Context) AbortLoopDetected(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusLoopDetected, "Loop Detected", msg, err...)
}

// ErrorNotExtended writes a 510 Not Extended response.
func (c *Context) ErrorNotExtended(message any) error {
	return c.JSON(http.StatusNotExtended, message)
}

// AbortNotExtended writes a standardized 510 Not Extended response.
func (c *Context) AbortNotExtended(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNotExtended, "Not Extended", msg, err...)
}

// ErrorNetworkAuthenticationRequired writes a 511 Network Authentication Required response.
func (c *Context) ErrorNetworkAuthenticationRequired(message any) error {
	return c.JSON(http.StatusNetworkAuthenticationRequired, message)
}

// AbortNetworkAuthenticationRequired writes a standardized 511 Network Authentication Required response.
func (c *Context) AbortNetworkAuthenticationRequired(msg string, err ...error) error {
	return c.abortWithStatus(http.StatusNetworkAuthenticationRequired, "Network Authentication Required", msg, err...)
}

// ********** Utility Methods *************

// IsClientError checks if the status code is a 4xx client error.
func IsClientError(code int) bool {
	return code >= 400 && code < 500
}

// IsServerError checks if the status code is a 5xx server error.
func IsServerError(code int) bool {
	return code >= 500 && code < 600
}

// IsError checks if the status code represents an error (4xx or 5xx).
func IsError(code int) bool {
	return IsClientError(code) || IsServerError(code)
}
