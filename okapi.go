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
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	goutils "github.com/jkaninda/go-utils"
	httpSwagger "github.com/swaggo/http-swagger"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	DefaultWriter      io.Writer = os.Stdout
	DefaultErrorWriter io.Writer = os.Stderr
	DefaultPort                  = 8080
	DefaultAddr                  = ":8080"
)

type (
	Okapi struct {
		context           *Context
		router            *Router
		middlewares       []Middleware
		Server            *http.Server
		TLSServer         *http.Server
		tlsConfig         *tls.Config
		tlsServerConfig   *tls.Config
		withTlsServer     bool
		tlsAddr           string
		routes            []*Route
		debug             bool
		accessLog         bool
		strictSlash       bool
		logger            *slog.Logger
		Renderer          Renderer
		corsEnabled       bool
		cors              Cors
		writeTimeout      int
		readTimeout       int
		idleTimeout       int
		optionsRegistered map[string]bool
		openapiSpec       *openapi3.T
		openAPI           *OpenAPI
		openApiEnabled    bool
	}
	Router struct {
		mux *mux.Router
	}
	OptionFunc func(*Okapi)

	// M is shortcut of map[string]any
	M map[string]any

	Route struct {
		Name            string
		Path            string
		Method          string
		Handle          HandleFunc
		chain           chain
		GroupPath       string
		Tags            []string
		Summary         string
		Request         *openapi3.SchemaRef
		Response        *openapi3.SchemaRef
		PathParams      []*openapi3.ParameterRef
		QueryParams     []*openapi3.ParameterRef
		Headers         []*openapi3.ParameterRef
		RequiresAuth    bool
		RequestExample  map[string]interface{}
		ResponseExample map[string]interface{}
		Responses       map[int]any
		Description     string
	}
	// Response interface defines the methods for writing HTTP responses.
	Response interface {
		http.ResponseWriter
		BodyBytesSent() int
		Status() int
		Close()
		Hijack() (net.Conn, *bufio.ReadWriter, error)
	}
	// HandleFunc is a function type that takes a Context and returns an error.
	HandleFunc func(c Context) error

	Middleware func(next HandleFunc) HandleFunc
)

// chain is an interface that defines a method for chaining handlers.
type chain interface {
	Next(hf HandleFunc) HandleFunc
}

// Response implementation
type response struct {
	writer        http.ResponseWriter
	status        int
	headerWritten bool
}

func (r *response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.writer.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrHijacked
	}
	return hj.Hijack()
}

// ****** OKAPI OPTIONS ******

// WithMux sets the router for the Okapi instance
func WithMux(mux *mux.Router) OptionFunc {
	return func(o *Okapi) {
		if mux != nil {
			o.router.mux = mux
		}
	}
}

// WithServer sets the HTTP server for the Okapi instance
func WithServer(server *http.Server) OptionFunc {
	return func(o *Okapi) {
		if server != nil {
			o.Server = server
		}
	}
}

// WithTLS sets tls config to HTTP Server for the Okapi instance
//
// Use okapi.LoadTLSConfig() to create a TLS configuration from certificate and key files
func WithTLS(tlsConfig *tls.Config) OptionFunc {
	return func(o *Okapi) {
		if tlsConfig != nil {
			o.tlsConfig = tlsConfig
		}
	}
}

// WithTLSServer sets the TLS server for the Okapi instance
//
// Use okapi.LoadTLSConfig() to create a TLS configuration from certificate and key files
func WithTLSServer(addr string, tlsConfig *tls.Config) OptionFunc {
	return func(o *Okapi) {
		if len(addr) != 0 && tlsConfig != nil {
			if !ValidateAddr(addr) {
				log.Panicf("Invalid address for the TLS Server: %s", addr)
			}
			o.withTlsServer = true
			o.tlsAddr = addr
			o.tlsServerConfig = tlsConfig
		}
	}
}

// WithLogger sets the logger for the Okapi instance
func WithLogger(logger *slog.Logger) OptionFunc {
	return func(o *Okapi) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// WithCors returns an OptionFunc that configures CORS settings
func WithCors(cors Cors) OptionFunc {
	return func(o *Okapi) {
		o.corsEnabled = true
		o.cors = cors
	}
}

// WithWriteTimeout returns an OptionFunc that sets the write timeout
func WithWriteTimeout(t int) OptionFunc {
	return func(o *Okapi) {
		o.writeTimeout = t
		o.Server.WriteTimeout = secondsToDuration(t)
	}
}

// WithReadTimeout returns an OptionFunc that sets the read timeout
func WithReadTimeout(t int) OptionFunc {
	return func(o *Okapi) {
		o.readTimeout = t
		o.Server.ReadTimeout = secondsToDuration(t)
	}
}

// WithIdleTimeout returns an OptionFunc that sets the idle timeout
func WithIdleTimeout(t int) OptionFunc {
	return func(o *Okapi) {
		o.idleTimeout = t
		o.Server.IdleTimeout = secondsToDuration(t)
	}
}

// WithStrictSlash sets whether to enforce strict slash handling
func WithStrictSlash(strict bool) OptionFunc {
	return func(o *Okapi) {
		o.strictSlash = strict
	}
}

// WithDebug enables debug mode and access logging
func WithDebug() OptionFunc {
	return func(o *Okapi) {
		o.debug = true
		o.accessLog = true
	}
}

// WithAccessLogDisabled disables access logging
func WithAccessLogDisabled() OptionFunc {
	return func(o *Okapi) {
		o.accessLog = false
	}
}

// WithPort sets the server port
func WithPort(port int) OptionFunc {
	return func(o *Okapi) {
		if port <= 0 {
			port = DefaultPort
		}
		host, _, err := net.SplitHostPort(o.Server.Addr)
		if err != nil || host == "" {
			host = ""
		}
		o.Server.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	}
}

// WithAddr sets the server address
func WithAddr(addr string) OptionFunc {
	return func(o *Okapi) {
		if strings.TrimSpace(addr) == "" || addr == ":" {
			addr = DefaultAddr
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			addr = net.JoinHostPort(addr, strconv.Itoa(DefaultPort))
		}
		o.Server.Addr = addr
	}
}

// WithOpenAPIDisabled disabled OpenAPI Docs
func WithOpenAPIDisabled() OptionFunc {
	return func(o *Okapi) {
		o.openApiEnabled = false
	}
}

// ************* Chaining methods *************
// These methods reuse the OptionFunc implementations

func (o *Okapi) WithLogger(logger *slog.Logger) *Okapi {
	return o.apply(WithLogger(logger))
}

func (o *Okapi) WithCORS(cors Cors) *Okapi {
	return o.apply(WithCors(cors))
}

func (o *Okapi) WithWriteTimeout(seconds int) *Okapi {
	return o.apply(WithWriteTimeout(seconds))
}

func (o *Okapi) WithReadTimeout(seconds int) *Okapi {
	return o.apply(WithReadTimeout(seconds))
}

func (o *Okapi) WithIdleTimeout(seconds int) *Okapi {
	return o.apply(WithIdleTimeout(seconds))
}

func (o *Okapi) WithStrictSlash(strict bool) *Okapi {
	return o.apply(WithStrictSlash(strict))
}

func (o *Okapi) WithDebug() *Okapi {
	return o.apply(WithDebug())
}

func (o *Okapi) WithPort(port int) *Okapi {
	return o.apply(WithPort(port))
}

func (o *Okapi) WithAddr(addr string) *Okapi {
	return o.apply(WithAddr(addr))
}

func (o *Okapi) DisableAccessLog() *Okapi {
	return o.apply(WithAccessLogDisabled())
}

// WithOpenAPIDocs registers the OpenAPI JSON and Swagger UI handlers
// at the configured PathPrefix (default: /docs).
//
// UI Path: /docs
// JSON Path: /openapi.json
func (o *Okapi) WithOpenAPIDocs(cfg ...OpenAPI) *Okapi {
	o.openApiEnabled = true

	if len(cfg) > 0 {
		config := cfg[0]

		if config.Title != "" {
			o.openAPI.Title = config.Title
		}
		if config.PathPrefix != "" {
			o.openAPI.PathPrefix = config.PathPrefix
		}
		if config.Version != "" {
			o.openAPI.Version = config.Version
		}
		if len(config.Servers) > 0 {
			o.openAPI.Servers = config.Servers
		}
	}
	if !strings.HasSuffix(o.openAPI.PathPrefix, "/") {
		o.openAPI.PathPrefix += "/"
	}

	// Ensure /docs redirects to /docs/
	o.router.mux.HandleFunc(strings.TrimSuffix(o.openAPI.PathPrefix, "/"), func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, o.openAPI.PathPrefix, http.StatusMovedPermanently)
	})

	o.buildOpenAPISpec()

	o.router.mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(o.openapiSpec)
	})

	o.router.mux.PathPrefix(o.openAPI.PathPrefix).Handler(httpSwagger.Handler(
		httpSwagger.URL("/openapi.json"),
	))

	return o
}

// ****** END OKAPI OPTIONS ******

// ****** RESPONSE WRITER ******

// Header returns the header map that will be sent by the ResponseWriter
func (r *response) Header() http.Header {
	return r.writer.Header()
}

// // Write writes the data to the response writer.
func (r *response) Write(bytes []byte) (int, error) {
	if !r.headerWritten {
		r.WriteHeader(http.StatusOK)
	}
	return r.writer.Write(bytes)
}

// WriteHeader sends an HTTP response header with the specified status code.
func (r *response) WriteHeader(statusCode int) {
	if r.headerWritten {
		return // Header already written
	}
	r.status = statusCode
	r.writer.WriteHeader(statusCode)
	r.headerWritten = true
}

// BodyBytesSent returns the number of bytes sent in the response body.
func (r *response) BodyBytesSent() int {
	if rw, ok := r.writer.(Response); ok {
		return rw.BodyBytesSent()
	}
	return 0
}

// Status returns the HTTP status code of the response writer.
func (r *response) Status() int {
	if !r.headerWritten {
		return http.StatusOK
	}
	return r.status
}

// Close closes the response writer if it implements io.Closer.
func (r *response) Close() {
	// Close the response writer if needed
	if closer, ok := r.writer.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return
		}
	}
}

// Flush flushes the response writer if it implements http.Flusher.
func (r *response) Flush() {
	if flusher, ok := r.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// ************ Router ************/

// newRouter creates a new Router instance
func newRouter() *Router {
	return &Router{
		mux: mux.NewRouter(),
	}
}

// ********** OKAPI **********//

// New creates a new Okapi instance with the provided options.
func New(options ...OptionFunc) *Okapi {
	return initConfig(options...)
}

// Default creates a new Okapi instance with default settings.
func Default() *Okapi {
	return New(
		withDefaultConfig(),
	)
}

func withDefaultConfig() OptionFunc {
	return func(o *Okapi) {
		o.openApiEnabled = true
	}
}

// With applies the provided options to the Okapi instance
func (o *Okapi) With(options ...OptionFunc) *Okapi {

	o.apply(options...)

	o.applyServerConfig(o.Server)

	if o.tlsServerConfig != nil {
		o.TLSServer.TLSConfig = o.tlsServerConfig
		o.TLSServer.Addr = o.tlsAddr
		o.applyServerConfig(o.TLSServer)
	}
	if o.openApiEnabled {
		o.WithOpenAPIDocs()
	}
	return o
}

// Start starts the Okapi server
func (o *Okapi) Start() error {
	return o.StartServer(o.Server)
}

// Use registers one or more middleware functions to the Okapi instance.
// These middleware will be executed in the order they are added for every request
// before reaching the route handler. Middleware added here will apply to all routes
// registered on this Okapi instance and any groups created from it.
//
// Middleware functions have the signature:
//
//	func(next HandleFunc) HandleFunc
//
// Example:
//
//	// Add logging and authentication middleware
//	okapi.Use(LoggingMiddleware, AuthMiddleware)
//
// Note: For group-specific middleware, use Group.Use() instead.
func (o *Okapi) Use(middlewares ...Middleware) {
	o.middlewares = append(o.middlewares, middlewares...)
}

// StartServer starts the Okapi server with the specified HTTP server
func (o *Okapi) StartServer(server *http.Server) error {
	if !ValidateAddr(server.Addr) {
		o.logger.Error("Invalid server address", slog.String("addr", server.Addr))
		panic("Invalid server address")
	}
	if o.openApiEnabled {
		o.WithOpenAPIDocs()
	}
	o.Server = server
	server.Handler = o
	o.router.mux.StrictSlash(o.strictSlash)
	o.context.okapi = o

	_, _ = fmt.Fprintf(DefaultWriter, "Starting HTTP server at %s\n", o.Server.Addr)
	// Serve with TLS if configured
	if server.TLSConfig != nil {
		return server.ListenAndServeTLS("", "")
	}

	// Serve with separate TLS server if enabled
	if o.withTlsServer && o.tlsServerConfig != nil {
		go func() {
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				o.logger.Error("HTTP server error", slog.String("error", err.Error()))
				panic(err)
			}
		}()

		o.TLSServer.Handler = o
		_, _ = fmt.Fprintf(DefaultWriter, "Starting HTTP server at %s\n", o.TLSServer.Addr)
		return o.TLSServer.ListenAndServeTLS("", "")
	}

	// Default HTTP only
	return server.ListenAndServe()
}

// Stop gracefully shuts down the Okapi server(s)
func (o *Okapi) Stop() {
	_, _ = fmt.Fprintf(DefaultWriter, "Gracefully shutting down HTTP server at %s\n", o.TLSServer.Addr)
	if err := o.Shutdown(o.Server); err != nil {
		o.logger.Error("Failed to shutdown HTTP server", slog.String("error", err.Error()))
		panic(err)
	}
	o.Server = nil

	if o.withTlsServer && o.tlsServerConfig != nil && o.TLSServer != nil {
		_, _ = fmt.Fprintf(DefaultWriter, "Gracefully shutting down HTTPS server at %s\n", o.TLSServer.Addr)
		if err := o.Shutdown(o.TLSServer); err != nil {
			o.logger.Error("Failed to shutdown HTTPS server", slog.String("error", err.Error()))
			panic(err)
		}
		o.TLSServer = nil
	}
}

// Shutdown wraps graceful server shutdown with context
func (o *Okapi) Shutdown(server *http.Server) error {
	if server == nil {
		return nil
	}
	return server.Shutdown(context.Background())
}

// GetContext returns the current context
func (o *Okapi) GetContext() *Context {
	return o.context
}
func (o *Okapi) SetContext(ctx *Context) {
	o.context = ctx
}

// ********** HTTP METHODS ***************

// Get registers a new GET route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Get(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(GET, path, "", h, opts...)
}

// Post registers a new POST route with the given path and handler function.
// Returns the created *Route for possible chaining or modification.
func (o *Okapi) Post(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(POST, path, "", h, opts...)
}

// Put registers a new PUT route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Put(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(PUT, path, "", h, opts...)
}

// Delete registers a new DELETE route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Delete(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodDelete, path, "", h, opts...)
}

// Patch registers a new PATCH route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Patch(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(PATCH, path, "", h, opts...)
}

// Options registers a new OPTIONS route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Options(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodOptions, path, "", h, opts...)
}

// Head registers a new HEAD route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Head(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(HEAD, path, "", h, opts...)
}

// Connect registers a new CONNECT route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Connect(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodConnect, path, "", h, opts...)
}

// Trace registers a new TRACE route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Trace(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(TRACE, path, "", h, opts...)
}

// Any registers a route that matches any HTTP method with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Any(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute("", path, "", h, opts...)
}

// ********** Static Content ***************

// Static serves static files under a path prefix, without directory listing
func (o *Okapi) Static(prefix string, dir string) {
	fs := http.StripPrefix(prefix, http.FileServer(noDirListing{http.Dir(dir)}))
	o.router.mux.PathPrefix(prefix).Handler(fs).Methods(http.MethodGet)
}

// StaticFile serves a single file at the specified path.
func (o *Okapi) StaticFile(path string, filepath string) {
	o.router.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath)
	}).Methods(http.MethodGet)
}

// StaticFS serves static files from a custom http.FileSystem (e.g., embed.FS).
func (o *Okapi) StaticFS(prefix string, fs http.FileSystem) {
	fileServer := http.StripPrefix(prefix, http.FileServer(fs))
	o.router.mux.PathPrefix(prefix).Handler(fileServer).Methods(http.MethodGet)
}

// addRoute adds a route with the specified method to the Okapi instance
func (o *Okapi) addRoute(method, path, groupPath string, h HandleFunc, opts ...RouteOption) *Route {
	if path == "" {
		panic("Path cannot be empty")
	}
	if groupPath == "" {
		groupPath = "default"
	}
	path = normalizeRoutePath(path)
	route := &Route{
		Name:      handleName(h),
		Path:      path,
		Method:    method,
		GroupPath: groupPath,
		Handle:    h,
		chain:     o,
	}
	for _, opt := range opts {
		opt(route)
	}
	o.routes = append(o.routes, route)

	handler := o.Next(h)

	// Main handler
	o.router.mux.StrictSlash(o.strictSlash).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{
			Request:  r,
			Response: &response{writer: w},
			okapi:    o,
		}
		if err := handler(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods(method)
	// Register OPTIONS handler only once per path if CORS is enabled
	o.registerOptionsHandler(path)
	return route
}

// HandleFunc registers a new route with the specified HTTP method, path, and handler function.
// It performs the following operations:
//  1. Normalizes the route path
//  2. Creates a new Route instance
//  3. Applies all registered middlewares to the handler
//  4. Registers the route with the underlying router
//  5. Sets up error handling for the handler
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, etc.)
//   - path: URL path pattern (supports path parameters)
//   - h: Handler function that processes the request
//
// Middleware Processing:
//
//	All middlewares registered via Use() will be applied in order before the handler.
//	The middleware chain is built using the Next() method.
//
// Error Handling:
//
//	Any error returned by the handler will be converted to a 500 Internal Server Error.
//
// Example:
//
//	okapi.HandleFunc("GET", "/users/:id", func(c Context) error {
//	    id := c.Param("id")
//	    // ... handler logic
//	    return nil
//	})
func (o *Okapi) HandleFunc(method, path string, h HandleFunc, opts ...RouteOption) {
	path = normalizeRoutePath(path)

	route := &Route{
		Name:   handleName(h),
		Path:   path,
		Method: method,
		Handle: h,
		chain:  o,
	}
	for _, opt := range opts {
		opt(route)
	}
	o.routes = append(o.routes, route)

	handler := o.Next(h)

	// Register main method handler
	o.router.mux.StrictSlash(o.strictSlash).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{
			Request:  r,
			Response: &response{writer: w},
			okapi:    o,
		}
		if err := handler(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods(method)
	// Register OPTIONS handler only once per path if CORS is enabled
	o.registerOptionsHandler(path)

}

// Handle registers a new route with the specified HTTP method, path, and http.Handler.
// It wraps the standard http.Handler into a HandleFunc signature and processes it similarly to HandleFunc.
//
// Parameters:
//   - method: HTTP method (GET, POST, PUT, etc.)
//   - path: URL path pattern (supports path parameters)
//   - h: Standard http.Handler that processes the request
//
// Middleware Processing:
//
//	All middlewares registered via Use() will be applied in order before the handler.
//	The middleware chain is built using the Next() method.
//
// Differences from HandleFunc:
//   - Accepts standard http.Handler instead of HandleFunc
//   - Handler errors must be handled by the http.Handler itself
//   - Returns nil error by default since http.Handler doesn't return errors
//
// Example:
//
//	okapi.Handle("GET", "/static", http.FileServer(http.Dir("./public")))
func (o *Okapi) Handle(method, path string, h http.Handler, opts ...RouteOption) {
	path = normalizeRoutePath(path)

	// Wrap http.Handler into HandleFunc signature
	handleFunc := func(ctx Context) error {
		h.ServeHTTP(ctx.Response, ctx.Request)
		return nil
	}

	// Register like in HandleFunc
	route := &Route{
		Name:   handleName(handleFunc),
		Path:   path,
		Method: method,
		Handle: handleFunc,
		chain:  o,
	}
	for _, opt := range opts {
		opt(route)
	}
	o.routes = append(o.routes, route)

	handler := o.Next(handleFunc)

	o.router.mux.StrictSlash(o.strictSlash).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{
			Request:  r,
			Response: &response{writer: w},
			okapi:    o,
		}

		if err := handler(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods(method)

	// Register OPTIONS handler for CORS
	o.registerOptionsHandler(path)
}

// registerOptionsHandler registers OPTIONS handler
func (o *Okapi) registerOptionsHandler(path string) {
	// Register OPTIONS handler only once per path if CORS is enabled
	if o.corsEnabled && !o.optionsRegistered[path] {
		o.optionsRegistered[path] = true

		o.router.mux.StrictSlash(o.strictSlash).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if !allowedOrigin(o.cors.AllowedOrigins, origin) {
				http.Error(w, "", http.StatusMethodNotAllowed)
				return
			}

			header := w.Header()
			header.Set(AccessControlAllowOrigin, origin)

			if o.cors.AllowCredentials {
				header.Set(AccessControlAllowCredentials, "true")
			}

			if len(o.cors.AllowedHeaders) > 0 {
				header.Set(AccessControlAllowHeaders, strings.Join(o.cors.AllowedHeaders, ", "))
			} else if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
				header.Set(AccessControlAllowHeaders, reqHeaders)
			}

			// Dynamically collect allowed methods for this path
			var methods []string
			for _, route := range o.routes {
				if route.Path == path {
					methods = append(methods, route.Method)
				}
			}
			if len(o.cors.AllowMethods) > 0 {
				header.Set(AccessControlAllowMethods, strings.Join(o.cors.AllowMethods, ", "))
			} else if len(methods) > 0 {
				header.Set(AccessControlAllowMethods, strings.Join(methods, ", "))
			}

			if len(o.cors.ExposeHeaders) > 0 {
				header.Set(AccessControlExposeHeaders, strings.Join(o.cors.ExposeHeaders, ", "))
			}

			if o.cors.MaxAge > 0 {
				header.Set(AccessControlMaxAge, strconv.Itoa(o.cors.MaxAge))
			}

			w.WriteHeader(http.StatusNoContent)
		}).Methods(http.MethodOptions)
	}
}

// ServeHTTP implements the http.Handler interface
func (o *Okapi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx := &Context{
		Request:  r,
		Response: &response{writer: w},
		okapi:    o,
	}
	handler := func(c *Context) {
		o.router.mux.ServeHTTP(c.Response, c.Request)
	}
	handler(ctx)
}

// Middlewares returns the list of middlewares
func (o *Okapi) Middlewares() []Middleware {
	return o.middlewares
}

// Next applies the middlewares in correct order
func (o *Okapi) Next(h HandleFunc) HandleFunc {
	// Start with the original handler
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		h = o.middlewares[i](h)
	}
	return h
}
func (o *Okapi) Routes() []Route {
	routes := make([]Route, 0, len(o.routes))
	for _, route := range o.routes {
		routes = append(routes, *route)
	}
	return routes
}

// Group creates a new route group with the specified base path and optional middlewares.
// The group inherits all existing middlewares from the parent Okapi instance.
// Routes registered within the group will have their paths prefixed with the group's path,
// and the group's middlewares will be executed before the route-specific handlers.
//
// Panics if the path is empty, as this would lead to ambiguous routing.
//
// Example:
//
//	api := okapi.Group("/api", AuthMiddleware) // All /api routes require auth
//	api.Get("/users", getUserHandler)          // Handles /api/users
//	api.Post("/users", createUserHandler)      // Handles /api/users
func (o *Okapi) Group(path string, middlewares ...Middleware) *Group {
	if len(path) == 0 {
		panic("Group path cannot be empty")
	}
	group := &Group{
		basePath:    path,
		okapi:       o,
		middlewares: middlewares,
	}
	return group
}

// initConfig initializes a new Okapi instance.
func initConfig(options ...OptionFunc) *Okapi {
	server := &http.Server{
		Addr: DefaultAddr,
	}

	o := &Okapi{
		context: &Context{
			Request:            new(http.Request),
			Response:           &response{},
			MaxMultipartMemory: defaultMaxMemory,
		},
		router:            newRouter(),
		Server:            server,
		TLSServer:         &http.Server{},
		logger:            slog.Default(),
		accessLog:         true,
		middlewares:       []Middleware{handleAccessLog},
		optionsRegistered: make(map[string]bool),
		cors:              Cors{},
		openAPI: &OpenAPI{
			Title:      FrameworkName,
			Version:    "1.0.0",
			PathPrefix: OpenApiDocPrefix,
			Servers:    openapi3.Servers{{URL: OpenApiURL}},
		},
	}

	return o.With(options...)
}

// applyServerConfig sets common server timeout and keep-alive configurations
func (o *Okapi) applyServerConfig(s *http.Server) {
	s.ReadTimeout = secondsToDuration(o.readTimeout)
	s.WriteTimeout = secondsToDuration(o.writeTimeout)
	s.IdleTimeout = secondsToDuration(o.idleTimeout)
}

// apply is a helper method to apply an OptionFunc to the Okapi instance
func (o *Okapi) apply(options ...OptionFunc) *Okapi {
	for _, option := range options {
		option(o)
	}
	return o
}

// handleName returns the name of the handler function.
func handleName(h HandleFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Type == reflect.TypeOf(http.HandlerFunc(nil)) {
				return field.Name
			}
		}
	}
	return t.Name()

}

// handleAccessLog logs the access details of the request
func handleAccessLog(next HandleFunc) HandleFunc {
	return func(c Context) error {
		if c.IsWebSocketUpgrade() || c.IsSSE() || !c.okapi.accessLog {
			// Skip logging for WebSocket upgrades or Server-Sent Events
			return next(c)
		}
		startTime := time.Now()
		err := next(c)

		duration := goutils.FormatDuration(time.Since(startTime), 2)
		c.okapi.logger.Info("[okapi]",
			"method", c.Request.Method,
			"url", c.Request.URL.Path,
			"client_ip", c.RealIP(),
			"status", c.Response.Status(),
			"duration", duration,
			"referer", c.Request.Referer(),
			"user_agent", c.Request.UserAgent(),
		)

		return err
	}
}
