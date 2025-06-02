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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type (
	Context struct {
		http.Handler
		okapi *Okapi
		// Request is the http.Request object
		Request *http.Request
		// Response http.ResponseWriter
		Response Response
		// CtxData is a key/value store for storing data in the context
		CtxData            map[string]any
		MaxMultipartMemory int64 // Maximum memory for multipart forms
		params             *Params
	}
)

// Mime types
const (
	JSON           = "application/json"
	XML            = "application/xml"
	HTML           = "text/html"
	FORM           = "application/x-www-form-urlencoded"
	FormData       = "multipart/form-data"
	PLAIN          = "text/plain"
	PLAINTEXT      = "text/plain"
	CSV            = "text/csv"
	JAVASCRIPT     = "application/javascript"
	YAML           = "application/yaml"
	YamlX          = "application/x-yaml"
	YamlText       = "text/yaml"
	PROTOBUF       = "application/protobuf"
	FormURLEncoded = "application/x-www-form-urlencoded"
)

// ************** Accessors *************

// Get retrieves a value from the context's data store with thread-safe access.
// Returns the value and a boolean indicating if the key exists.
func (c *Context) Get(key string) (any, bool) {

	val, ok := c.CtxData[key]
	return val, ok
}

// GetTime retrieves a time.Time value from the context's data store.
func (c *Context) GetTime(key string) (time.Time, bool) {
	if val, ok := c.CtxData[key]; ok {
		if t, ok := val.(time.Time); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

// Set stores a value in the context's data store with thread-safe access.
// Initializes the data map if it doesn't exist.
func (c *Context) Set(key string, value any) {
	if c.CtxData == nil {
		c.CtxData = make(map[string]any) // Initialize map if empty
	}
	c.CtxData[key] = value
}

// GetString retrieves a string value from the context.
// Returns empty string if key doesn't exist or value isn't a string.
func (c *Context) GetString(key string) string {
	if val, ok := c.Get(key); ok {
		if s, ok := val.(string); ok { // Type assertion to string
			return s
		}
	}
	return "" // Default value if not found or wrong type
}

// GetBool retrieves a boolean value from the context.
// Returns false if key doesn't exist or value isn't a bool.
func (c *Context) GetBool(key string) bool {
	if val, ok := c.Get(key); ok {
		if b, ok := val.(bool); ok { // Type assertion to bool
			return b
		}
	}
	return false // Default value if not found or wrong type
}

// GetInt retrieves an integer value from the context.
// Returns 0 if key doesn't exist or value isn't an int.
func (c *Context) GetInt(key string) int {
	if val, ok := c.Get(key); ok {
		if i, ok := val.(int); ok { // Type assertion to int
			return i
		}
	}
	return 0 // Default value if not found or wrong type
}

// GetInt64 retrieves an int64 value from the context.
// Returns 0 if key doesn't exist or value isn't an int64.
func (c *Context) GetInt64(key string) int64 {
	if val, ok := c.Get(key); ok {
		if i, ok := val.(int64); ok { // Type assertion to int64
			return i
		}
	}
	return 0 // Default value if not found or wrong type
}

// Copy creates a shallow copy of the context with a new data map.
// Maintains thread safety during the copy operation.
func (c *Context) Copy() *Context {
	newCtx := &Context{
		Request:            c.Request,                            // Copy request reference
		Response:           c.Response,                           // Copy response reference
		CtxData:            make(map[string]any, len(c.CtxData)), // Initialize new data map
		params:             c.params,                             // Copy params
		MaxMultipartMemory: c.MaxMultipartMemory,                 // Copy max memory for multipart forms
	}
	// Copy all key-value pairs to the new context
	for k, v := range c.CtxData {
		newCtx.CtxData[k] = v
	}
	return newCtx
}

// ************** Request Utilities *****************

// RealIP returns the client's real IP address, handling proxies.
func (c *Context) RealIP() string {
	return RealIP(c.Request) // Delegate to package-level RealIP function
}

// Referer retrieves the Referer header value from the request.
func (c *Context) Referer() string {
	return c.Request.Referer() // Get Referer header
}

// Param retrieves a URL path parameter value.
func (c *Context) Param(key string) string {
	return mux.Vars(c.Request)[key] // Get from router's path variables
}

// Query retrieves a URL query parameter value.
// Returns empty string if parameter doesn't exist.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key) // Get from URL query string
}

// QueryMap returns all query parameters as a map.
// Only includes the first value for each parameter.
func (c *Context) QueryMap() map[string]string {
	values := c.Request.URL.Query()
	result := make(map[string]string, len(values))
	for k, v := range values {
		if len(v) > 0 {
			result[k] = v[0] // Take first value for each key
		}
	}
	return result
}

// Accept returns the Accept header values as a slice.
func (c *Context) Accept() []string {
	accept := c.Request.Header.Get("Accept")
	if accept == "" {
		return nil // Return nil if header not present
	}
	return strings.Split(accept, ",") // Split by comma
}

// AcceptLanguage returns the Accept-Language header values as a slice.
// Trims whitespace from each language tag.
func (c *Context) AcceptLanguage() []string {
	languages := c.Request.Header.Get("Accept-Language")
	if languages == "" {
		return nil // Return nil if header not present
	}
	parts := strings.Split(languages, ",")
	for i, lang := range parts {
		parts[i] = strings.TrimSpace(lang) // Clean up each language tag
	}
	return parts
}

// ContentType returns the Content-Type header value.
func (c *Context) ContentType() string {
	return c.Request.Header.Get(ContentTypeHeader)
}

// Form retrieves a form value after parsing the form data.
func (c *Context) Form(key string) string {
	_ = c.Request.ParseForm() // Parse form if not already done
	return c.Request.FormValue(key)
}

// FormValue retrieves a form value, including multipart form data.
func (c *Context) FormValue(key string) string {
	_ = c.Request.ParseMultipartForm(defaultMaxMemory) // Parse multipart form
	return c.Request.FormValue(key)
}

// FormFile retrieves a file from multipart form data.
// Returns the file and any error encountered.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	f, fh, err := c.Request.FormFile(key)
	if err != nil {
		return nil, err
	}
	err = f.Close()
	if err != nil {
		return nil, err
	}
	return fh, err
}

// Cookie retrieves a cookie value by name.
// Returns empty string and error if cookie not found.
func (c *Context) Cookie(name string) (string, error) {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// SetCookie sets a cookie with various configurable options.
// Defaults path to "/" if empty.
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if path == "" {
		path = "/" // Default path to root
	}
	http.SetCookie(c.Response, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value), // URL-encode cookie value
		MaxAge:   maxAge,
		Path:     path,
		Domain:   domain,
		Secure:   secure,
		HttpOnly: httpOnly,
	})
}

// IsWebSocketUpgrade checks if the request is a WebSocket upgrade request.
func (c *Context) IsWebSocketUpgrade() bool {
	// Check if the request is a WebSocket upgrade request
	return c.Request.Header.Get("Upgrade") == "websocket" && c.Request.Method == http.MethodGet
}

// IsSSE checks if the request is for Server-Sent Events (SSE).
func (c *Context) IsSSE() bool {
	// Check if the request is for Server-Sent Events
	return c.Request.Header.Get("Accept") == "text/event-stream" && c.Request.Method == http.MethodGet
}

// ************* Response Utilities *************

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// Header gets a request header by key.
func (c *Context) Header(key string) string {
	return c.Request.Header.Get(key)
}

// Headers returns all request headers as a map.
func (c *Context) Headers() map[string][]string {
	return c.Request.Header
}

// WriteStatus writes the HTTP status code to the response.
func (c *Context) WriteStatus(code int) {
	c.Response.WriteHeader(code)
}

// writeResponse is a helper for writing responses with common headers and status.
// Takes care of content type, status code, and error handling.
func (c *Context) writeResponse(code int, contentType string, writeFunc func() error) error {
	c.Response.Header().Set(ContentTypeHeader, contentType)
	c.Response.WriteHeader(code)
	if err := writeFunc(); err != nil {
		http.Error(c.Response, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

// JSON writes a JSON response with the given status code.
func (c *Context) JSON(code int, v any) error {
	return c.writeResponse(code, JSON, func() error {
		return json.NewEncoder(c.Response).Encode(v)
	})
}

// XML writes an XML response with the given status code.
func (c *Context) XML(code int, v any) error {
	return c.writeResponse(code, XML, func() error {
		return xml.NewEncoder(c.Response).Encode(v)
	})
}

// YAML writes a YAML response with the given status code.
func (c *Context) YAML(code int, data any) error {
	return c.writeResponse(code, YAML, func() error {
		return yaml.NewEncoder(c.Response).Encode(data)
	})
}

// Text writes a plain text response with the given status code.
func (c *Context) Text(code int, v any) error {
	return c.writeResponse(code, PLAIN, func() error {
		_, err := fmt.Fprint(c.Response, v)
		return err
	})
}

// String is an alias for Text for convenience.
func (c *Context) String(code int, data any) error {
	return c.Text(code, data)
}

// Data writes a raw byte response with the given content type and status code.
func (c *Context) Data(code int, contentType string, data []byte) error {
	return c.writeResponse(code, contentType, func() error {
		_, err := c.Response.Write(data)
		return err
	})
}

// HTML renders an HTML template from a file with the given status code.
func (c *Context) HTML(code int, file string, data any) error {
	tmpl, err := template.ParseFiles(file) // Parse template file
	if err != nil {
		return err
	}
	return c.renderHTML(code, tmpl, data)
}

// HTMLView renders an HTML template from a string with the given status code.
func (c *Context) HTMLView(code int, templateStr string, data any) error {
	tmpl, err := template.New("inline").Parse(templateStr) // Parse template string
	if err != nil {
		return err
	}
	return c.renderHTML(code, tmpl, data)
}

// Render renders a template using the configured Renderer.
func (c *Context) Render(code int, name string, data interface{}) error {
	if c.okapi.Renderer == nil {
		return ErrNoRenderer
	}
	if name == "" {
		return c.writeResponse(code, HTML, func() error {
			return c.okapi.Renderer.Render(c.Response, "", nil, *c)
		})
	}
	return c.writeResponse(code, HTML, func() error {
		return c.okapi.Renderer.Render(c.Response, name, data, *c)
	})
}

// renderHTML is a helper for rendering HTML templates.
func (c *Context) renderHTML(code int, tmpl *template.Template, data any) error {
	return c.writeResponse(code, HTML, func() error {
		return tmpl.Execute(c.Response, data) // Execute template with data
	})
}

// Redirect sends a redirect response to the specified location.
func (c *Context) Redirect(code int, location string) {
	c.SetHeader(LocationHeader, location)                         // Set Location header
	c.WriteStatus(code)                                           // Write status code
	_, _ = fmt.Fprintf(c.Response, "Redirecting to %s", location) // Optional message
}

// *********** File Serving **************

// ServeFile serves a file from the filesystem.
func (c *Context) ServeFile(path string) {
	http.ServeFile(c.Response, c.Request, path) // Use standard library file server
}

// ServeFileFromFS serves a file from a custom http.FileSystem.
func (c *Context) ServeFileFromFS(filepath string, fs http.FileSystem) {
	// Sanitize the path to prevent directory traversal
	filepath = path.Clean(filepath)
	if filepath == "." || strings.Contains(filepath, "..") {
		err := c.ErrorNotFound("Not found")
		if err != nil {
			return
		}
	}

	// Preserve original URL.Path
	oldPath := c.Request.URL.Path
	defer func() { c.Request.URL.Path = oldPath }()
	c.Request.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.Response, c.Request)
}

// ServeFileAttachment serves a file as an attachment (download).
func (c *Context) ServeFileAttachment(path, filename string) {
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(c.Response, c.Request, path)
}

// ServeFileInline serves a file to be displayed inline in the browser.
func (c *Context) ServeFileInline(path, filename string) {
	c.SetHeader("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	http.ServeFile(c.Response, c.Request, path)
}

// ********** Error Handling *************

// Error writes an error response with the given status code and message.
// Returns an error if writing the response fails.
func (c *Context) Error(code int, message string) error {
	c.Response.WriteHeader(code)
	_, err := c.Response.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write error response: %w", err)
	}
	return nil
}

// ErrorNotFound writes a 404 Not Found response.
// Returns an error if writing the response fails.
func (c *Context) ErrorNotFound(message string) error {
	return c.Error(http.StatusNotFound, message)
}

// ErrorInternalServerError writes a 500 Internal Server Error response.
// Returns an error if writing the response fails.
func (c *Context) ErrorInternalServerError(message string) error {
	return c.Error(http.StatusInternalServerError, message)
}

// ErrorBadRequest writes a 400 Bad Request response.
// Returns an error if writing the response fails.
func (c *Context) ErrorBadRequest(message string) error {
	return c.Error(http.StatusBadRequest, message)
}

// ErrorUnauthorized writes a 401 Unauthorized response.
func (c *Context) ErrorUnauthorized(message string) error {
	return c.Error(http.StatusUnauthorized, message)
}

// ErrorForbidden writes a 403 Forbidden response.
func (c *Context) ErrorForbidden(message string) error {
	return c.Error(http.StatusForbidden, message)
}

// ErrorConflict writes a 409 Conflict response.
func (c *Context) ErrorConflict(message string) error {
	return c.Error(http.StatusConflict, message)
}

// ErrorUnprocessableEntity writes a 422 Unprocessable Entity response.
func (c *Context) ErrorUnprocessableEntity(message string) error {
	return c.Error(http.StatusUnprocessableEntity, message)
}

// ErrorTooManyRequests writes a 429 Too Many Requests response.
func (c *Context) ErrorTooManyRequests(message string) error {
	return c.Error(http.StatusTooManyRequests, message)
}

// ErrorServiceUnavailable writes a 503 Service Unavailable response.
func (c *Context) ErrorServiceUnavailable(message string) error {
	return c.Error(http.StatusServiceUnavailable, message)
}

// AbortWithError writes an error response with the given status code and standardized format.
// Returns the error for chaining.
func (c *Context) AbortWithError(code int, msg string, err error) error {
	details := ""
	if err != nil {
		details = err.Error()
	}

	return c.JSON(code, ErrorResponse{
		Code:    code,
		Message: msg,
		Details: details,
	})
}

// Abort writes an error response with 500 status code and standardized format.
func (c *Context) Abort(err error) error {
	return c.AbortWithError(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), err)
}

// abortWithStatus is a helper for status-only abort functions
func (c *Context) abortWithStatus(code int, defaultMsg string, msg ...string) error {
	message := defaultMsg
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	return c.AbortWithError(code, message, nil)
}

// AbortBadRequest writes an error response with 400 status code and standardized format.
func (c *Context) AbortBadRequest(msg ...string) error {
	return c.abortWithStatus(http.StatusBadRequest, http.StatusText(http.StatusBadRequest), msg...)
}

// AbortUnauthorized writes an error response with 401 status code.
func (c *Context) AbortUnauthorized(msg ...string) error {
	return c.abortWithStatus(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), msg...)
}

// AbortForbidden writes an error response with 403 status code.
func (c *Context) AbortForbidden(msg ...string) error {
	return c.abortWithStatus(http.StatusForbidden, http.StatusText(http.StatusForbidden), msg...)
}

// AbortNotFound writes an error response with 404 status code.
func (c *Context) AbortNotFound(msg ...string) error {
	return c.abortWithStatus(http.StatusNotFound, http.StatusText(http.StatusNotFound), msg...)
}

// AbortConflict writes an error response with 409 status code.
func (c *Context) AbortConflict(msg ...string) error {
	return c.abortWithStatus(http.StatusConflict, http.StatusText(http.StatusConflict), msg...)
}

// AbortValidationError writes an error response with 422 status code.
func (c *Context) AbortValidationError(msg ...string) error {
	return c.abortWithStatus(http.StatusUnprocessableEntity, http.StatusText(http.StatusUnprocessableEntity), msg...)
}

// AbortTooManyRequests writes an error response with 429 status code.
func (c *Context) AbortTooManyRequests(msg ...string) error {
	return c.abortWithStatus(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests), msg...)
}

// AbortWithStatus writes an error response with the given status code and message.
// Useful when you don't have an error object but just a message.
func (c *Context) AbortWithStatus(code int, message string) error {
	return c.JSON(code, ErrorResponse{
		Code:    code,
		Message: http.StatusText(code),
		Details: message,
	})
}

// AbortWithJSON writes a custom JSON error response with the given status code.
func (c *Context) AbortWithJSON(code int, jsonObj interface{}) error {
	return c.JSON(code, jsonObj)
}
