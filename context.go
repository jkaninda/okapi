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
	"sync"
	"time"
)

type (
	Context struct {
		okapi *Okapi
		// Request is the http.Request object
		Request *http.Request
		// Response http.ResponseWriter
		Response Response
		// store is a key/value store for storing data in the context
		store *Store
		// params *Params
	}
	Store struct {
		mu   sync.RWMutex
		data map[string]any
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
		Request:  c.Request,      // Copy request reference
		Response: c.Response,     // Copy response reference
		store:    newStoreData(), // Initialize new data map
	}
	// Copy all key-value pairs to the new context
	for k, v := range c.store.data {
		newCtx.store.data[k] = v
	}
	return newCtx
}

// ************** Request Utilities *****************

// RealIP returns the client's real IP address, handling proxies.
func (c *Context) RealIP() string {
	return realIP(c.Request)
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
	_ = c.Request.ParseMultipartForm(c.okapi.maxMultipartMemory) // Parse multipart form
	return c.Request.FormValue(key)
}

// FormFile retrieves a file from multipart form data.
// Returns the file and any error encountered.
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	_ = c.Request.ParseMultipartForm(c.okapi.maxMultipartMemory)
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

// SSEvent writes SSE response.
func (c *Context) SSEvent(name string, message any) error {
	msg := Message{
		Event: name,
		Data:  message,
	}
	_, err := msg.Send(c.Response)
	if err != nil {
		return err
	}
	return nil
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
	if c.okapi.renderer == nil {
		return ErrNoRenderer
	}
	if name == "" {
		return c.writeResponse(code, HTML, func() error {
			return c.okapi.renderer.Render(c.Response, "", nil, *c)
		})
	}
	return c.writeResponse(code, HTML, func() error {
		return c.okapi.renderer.Render(c.Response, name, data, *c)
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

// ************ Errors in errors.go *****************
