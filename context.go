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
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
	"html/template"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"
)

type (
	Context struct {
		okapi *Okapi
		// request is the http.Request object
		request *http.Request
		// response http.ResponseWriter
		response ResponseWriter
		// store is a key/value store for storing data in the context
		store *Store
		// params *Params
	}
	Store struct {
		mu   sync.RWMutex
		data map[string]any
	}
	// C is a shortcut of *Context
	C = *Context
)

// Mime types
const (
	JSON           = "application/json"
	XML            = "application/xml"
	HTML           = "text/html"
	FORM           = "application/x-www-form-urlencoded"
	FormData       = "multipart/form-data"
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

// newStoreData creates a new instance of Store
func newStoreData() *Store {
	return &Store{
		data: make(map[string]any),
	}
}
func getAs[T any](c *Context, key string) (v T, ok bool) {
	raw, exists := c.Get(key)
	if !exists {
		return
	}
	v, ok = raw.(T)
	return
}

// Request a new Context instance with the given request
func (c *Context) Request() *http.Request {
	return c.request // Return the request object
}

// Response returns the http.ResponseWriter for writing responses.
// This is an alias for ResponseWriter for convenience.
func (c *Context) Response() ResponseWriter {
	return c.response // Return the response writer
}

// ResponseWriter returns the http.ResponseWriter for writing responses.
func (c *Context) ResponseWriter() http.ResponseWriter {
	return c.response // Return the http.ResponseWriter
}

// Logger returns the logger instance associated with the Okapi context.
func (c *Context) Logger() *slog.Logger {
	if c.okapi.logger == nil {
		return slog.Default()
	}
	return c.okapi.logger

}

// Get retrieves a value from the context's data store with thread-safe access.
// Returns the value and a boolean indicating if the key exists.
func (c *Context) Get(key string) (any, bool) {
	if c.store == nil {
		return nil, false
	}
	c.store.mu.RLock()
	defer c.store.mu.RUnlock()
	val, ok := c.store.data[key]
	return val, ok
}

// GetTime retrieves a time.Time value from the context's data store.
func (c *Context) GetTime(key string) (time.Time, bool) {
	if val, ok := getAs[time.Time](c, key); ok {
		return val, true
	}
	return time.Time{}, false
}

// Set stores a value in the context's data store with thread-safe access.
// Initializes the data map if it doesn't exist.
func (c *Context) Set(key string, value any) {
	if c.store == nil {
		c.store = newStoreData()
	}
	c.store.mu.Lock()
	c.store.data[key] = value
	c.store.mu.Unlock()
}

// GetString retrieves a string value from the context.
// Returns empty string if key doesn't exist or value isn't a string.
func (c *Context) GetString(key string) string {
	if val, ok := getAs[string](c, key); ok {
		return val
	}
	return ""
}

// GetBool retrieves a boolean value from the context.
// Returns false if key doesn't exist or value isn't a bool.
func (c *Context) GetBool(key string) bool {
	if val, ok := getAs[bool](c, key); ok {
		return val
	}
	return false
}

// GetInt retrieves an integer value from the context.
// Returns 0 if key doesn't exist or value isn't an int.
func (c *Context) GetInt(key string) int {
	if val, ok := getAs[int](c, key); ok {
		return val
	}
	return 0
}

// GetInt64 retrieves an int64 value from the context.
// Returns 0 if key doesn't exist or value isn't an int64.
func (c *Context) GetInt64(key string) int64 {
	if val, ok := getAs[int64](c, key); ok {
		return val
	}
	return 0 // Default value if not found or wrong type
}

// Copy creates a shallow copy of the context with a new data map.
func (c *Context) Copy() *Context {
	newCtx := &Context{
		request:  c.request,      // Copy request reference
		response: c.response,     // Copy response reference
		store:    newStoreData(), // Initialize new data map
	}
	// Copy all key-value pairs to the new context
	for k, v := range c.store.data {
		newCtx.store.data[k] = v
	}
	return newCtx
}

// ************** request Utilities *****************

// RealIP returns the client's real IP address, handling proxies.
func (c *Context) RealIP() string {
	return realIP(c.request)
}

// Referer retrieves the Referer header value from the request.
func (c *Context) Referer() string {
	return c.request.Referer() // Get Referer header
}

// Param retrieves a URL path parameter value.
func (c *Context) Param(key string) string {
	return mux.Vars(c.request)[key] // Get from router's path variables
}

// Query retrieves a URL query parameter value.
// Returns empty string if parameter doesn't exist.
func (c *Context) Query(key string) string {
	return c.request.URL.Query().Get(key) // Get from URL query string
}

// QueryArray retrieves all values for a query parameter.
// Supports both repeated params (?tags=a&tags=b) and comma-separated (?tags=a,b).
func (c *Context) QueryArray(key string) []string {
	values, ok := c.request.URL.Query()[key]
	if !ok {
		return nil
	}
	var result []string
	for _, v := range values {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

// QueryMap returns all query parameters as a map.
// Only includes the first value for each parameter.
func (c *Context) QueryMap() map[string]string {
	values := c.request.URL.Query()
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
	accept := c.request.Header.Get("Accept")
	if accept == "" {
		return nil // Return nil if header not present
	}
	return strings.Split(accept, ",") // Split by comma
}

// AcceptLanguage returns the Accept-Language header values as a slice.
// Trims whitespace from each language tag.
func (c *Context) AcceptLanguage() []string {
	languages := c.request.Header.Get("Accept-Language")
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
	return c.request.Header.Get(constContentTypeHeader)
}

// Form retrieves a form value after parsing the form data.
func (c *Context) Form(key string) string {
	_ = c.request.ParseForm() // Parse form if not already done
	return c.request.FormValue(key)
}

// FormValue retrieves a form value, including multipart form data.
func (c *Context) FormValue(key string) string {
	_ = c.request.ParseMultipartForm(c.okapi.maxMultipartMemory) // Parse multipart form
	return c.request.FormValue(key)
}

// FormFile retrieves a file from multipart form data.
// Returns the file and any error encountered.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	_ = c.request.ParseMultipartForm(c.okapi.maxMultipartMemory)
	f, fh, err := c.request.FormFile(key)
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
	cookie, err := c.request.Cookie(name)
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
	http.SetCookie(c.response, &http.Cookie{
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
	return c.request.Header.Get("Upgrade") == "websocket" && c.request.Method == http.MethodGet
}

// IsSSE checks if the request is for Server-Sent Events (SSE).
func (c *Context) IsSSE() bool {
	// Check if the request is for Server-Sent Events
	return c.request.Header.Get("Accept") == "text/event-stream" && c.request.Method == http.MethodGet
}

// ************* response Utilities *************

// SetHeader sets a response header.
func (c *Context) SetHeader(key, value string) {
	c.response.Header().Set(key, value)
}

// Header gets a request header by key.
func (c *Context) Header(key string) string {
	return c.request.Header.Get(key)
}

// Headers returns all request headers as a map.
func (c *Context) Headers() map[string][]string {
	return c.request.Header
}

// WriteStatus writes the HTTP status code to the response.
func (c *Context) WriteStatus(code int) {
	c.response.WriteHeader(code)
}

// writeResponse is a helper for writing responses with common headers and status.
// Takes care of content type, status code, and error handling.
func (c *Context) writeResponse(code int, contentType string, writeFunc func() error) error {
	c.response.Header().Set(constContentTypeHeader, contentType)
	c.response.WriteHeader(code)
	if err := writeFunc(); err != nil {
		http.Error(c.response, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

// JSON writes a JSON response with the given status code.
func (c *Context) JSON(code int, v any) error {
	return c.writeResponse(code, JSON, func() error {
		return json.NewEncoder(c.response).Encode(v)
	})
}

// OK writes a JSON response with 200 status code.
func (c *Context) OK(v any) error {
	return c.JSON(http.StatusOK, v)
}

// Created writes a JSON response with 201 status code.
func (c *Context) Created(v any) error {
	return c.JSON(http.StatusCreated, v)
}

// XML writes an XML response with the given status code.
func (c *Context) XML(code int, v any) error {
	return c.writeResponse(code, XML, func() error {
		return xml.NewEncoder(c.response).Encode(v)
	})
}

// YAML writes a YAML response with the given status code.
func (c *Context) YAML(code int, data any) error {
	return c.writeResponse(code, YAML, func() error {
		return yaml.NewEncoder(c.response).Encode(data)
	})
}

// Text writes a plain text response with the given status code.
func (c *Context) Text(code int, v any) error {
	return c.writeResponse(code, PLAINTEXT, func() error {
		_, err := fmt.Fprint(c.response, v)
		return err
	})
}

// sendSSE writes an SSE response with optional ID.
func (c *Context) sendSSE(id, name string, message any) error {
	msg := Message{
		ID:    id,
		Event: name,
		Data:  message,
	}
	_, err := msg.Send(c.response)
	return err
}

// SSEvent writes SSE response with optional ID.
func (c *Context) SSEvent(name string, message any) error {
	return c.sendSSE("", name, message)
}

// SendSSEvent writes SSE response with an ID.
func (c *Context) SendSSEvent(id, name string, message any) error {
	return c.sendSSE(id, name, message)
}

// SSEStream keeps connection open for multiple messages
func (c *Context) SSEStream(ctx context.Context, messageChan <-chan Message) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messageChan:
			if _, err := msg.Send(c.response); err != nil {
				return err
			}
		}
	}
}

// String is an alias for Text for convenience.
func (c *Context) String(code int, data any) error {
	return c.Text(code, data)
}

// Data writes a raw byte response with the given content type and status code.
func (c *Context) Data(code int, contentType string, data []byte) error {
	return c.writeResponse(code, contentType, func() error {
		_, err := c.response.Write(data)
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
	if c.okapi.renderer == nil {
		return ErrNoRenderer
	}
	if name == "" {
		return c.writeResponse(code, HTML, func() error {
			return c.okapi.renderer.Render(c.response, "", nil, c)
		})
	}
	return c.writeResponse(code, HTML, func() error {
		return c.okapi.renderer.Render(c.response, name, data, c)
	})
}

// renderHTML is a helper for rendering HTML templates.
func (c *Context) renderHTML(code int, tmpl *template.Template, data any) error {
	return c.writeResponse(code, HTML, func() error {
		return tmpl.Execute(c.response, data) // Execute template with data
	})
}

// Redirect sends a redirect response to the specified location.
func (c *Context) Redirect(code int, location string) {
	c.SetHeader(constLocationHeader, location)                    // Set Location header
	c.WriteStatus(code)                                           // Write status code
	_, _ = fmt.Fprintf(c.response, "Redirecting to %s", location) // Optional message
}

// *********** File Serving **************

// ServeFile serves a file from the filesystem.
func (c *Context) ServeFile(path string) {
	http.ServeFile(c.response, c.request, path) // Use standard library file server
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
	oldPath := c.request.URL.Path
	defer func() { c.request.URL.Path = oldPath }()
	c.request.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.response, c.request)
}

// ServeFileAttachment serves a file as an attachment (download).
func (c *Context) ServeFileAttachment(path, filename string) {
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(c.response, c.request, path)
}

// ServeFileInline serves a file to be displayed inline in the browser.
func (c *Context) ServeFileInline(path, filename string) {
	c.SetHeader("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	http.ServeFile(c.response, c.request, path)
}

// *********** MultipartMemory **************

// MaxMultipartMemory returns the maximum memory for multipart form
func (c *Context) MaxMultipartMemory() int64 {
	return c.okapi.maxMultipartMemory
}

// SetMaxMultipartMemory sets the maximum memory for multipart form (default: 32 MB)
func (c *Context) SetMaxMultipartMemory(max int64) {
	if max > 0 {
		c.okapi.maxMultipartMemory = max
	}
}

// ******************* Output ********************

// Return is an alias for Respond to improve readability when sending output.
func (c *Context) Return(output any) error {
	return c.Respond(output)
}

// Respond serializes the output struct into the HTTP response.
// It inspects struct tags to automatically set headers, cookies, and status code,
// and encodes the response body in the format requested by the `Accept` header.
//
// Supported formats: JSON, XML, YAML, plain text, HTML.
//
// Example:
//
//	type BookResponse struct {
//	  Status  int                           // HTTP status code
//	  version string `header:"version"`     // Response header
//	  Session string `cookie:"SessionID"`   // Response cookie
//	  Body    struct {
//	    ID    int    `json:"id"`
//	    Name  string `json:"name"`
//	    Price int    `json:"price"`
//	  }
//	}
//
//	okapi.Get("/books/:id", func(c okapi.Context) error {
//	  return c.Respond(BookResponse{
//	    version: "v1",
//	    Session: "abc123",
//	    Status:  200,
//	    Body: struct {
//	      ID    int    `json:"id"`
//	      Name  string `json:"name"`
//	      Price int    `json:"price"`
//	    }{
//	      ID: 1, Name: "Okapi Guide", Price: 50,
//	    },
//	  })
//	})
func (c *Context) Respond(output any) error {
	v := reflect.ValueOf(output)
	if !v.IsValid() {
		return c.AbortInternalServerError("Internal Server Error", fmt.Errorf("output is nil"))
	}

	// Dereference pointer if needed
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return c.AbortInternalServerError("Internal Server Error", fmt.Errorf("output is nil pointer"))
		}
		v = v.Elem()
	}
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return c.AbortInternalServerError("Internal Server Error", fmt.Errorf("output must be a struct"))
	}

	status := getResponseStatus(v)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		val := v.Field(i).Interface()

		// Header tag
		if header := field.Tag.Get(tagHeader); header != "" {
			c.Response().Header().Set(header, fmt.Sprint(val))
			continue
		}
		// Cookie tag
		if cookie := field.Tag.Get(tagCookie); cookie != "" {
			http.SetCookie(c.Response(), &http.Cookie{
				Name:  cookie,
				Value: fmt.Sprint(val),
				Path:  "/",
			})
			continue
		}
		// Fallback: expose non-status, non-body fields as headers
		if field.Name != "Status" && field.Name != bodyField {
			key := field.Name
			if jsonTag := field.Tag.Get(tagJSON); jsonTag != "" && jsonTag != "-" {
				key = strings.Split(jsonTag, ",")[0]
			}
			c.Response().Header().Set(key, fmt.Sprint(val))
		}
	}

	var body any
	if f := v.FieldByName(bodyField); f.IsValid() {
		body = f.Interface()
	}

	accept := c.request.Header.Get("Accept")
	switch {
	case strings.Contains(accept, XML):
		return c.XML(status, body)
	case strings.Contains(accept, YAML), strings.Contains(accept, YamlText), strings.Contains(accept, YamlX):
		return c.YAML(status, body)
	case strings.Contains(accept, JSON):
		return c.JSON(status, body)
	case strings.Contains(accept, PLAINTEXT), strings.Contains(accept, HTML):
		return c.String(status, body)
	default:
		return c.JSON(status, body)
	}
}

// NewContext creates a new Okapi Context
func NewContext(o *Okapi, w http.ResponseWriter, r *http.Request) *Context {
	if o == nil {
		Default()
	}
	return &Context{
		request:  r,
		okapi:    o,
		response: newResponseWriter(w),
		store:    newStoreData(),
	}
}

// ************ Errors in errors.go *****************
