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
	defaultWriter      io.Writer = os.Stdout
	defaultErrorWriter io.Writer = os.Stderr
	defaultPort                  = 8080
	defaultAddr                  = ":8080"
)

type (
	// Okapi represents the core application structure of the framework,
	// holding configuration, routers, middleware, server settings, and documentation components.
	Okapi struct {
		context            *Context
		router             *Router
		middlewares        []Middleware
		server             *http.Server
		tlsServer          *http.Server
		tlsConfig          *tls.Config
		tlsServerConfig    *tls.Config
		withTlsServer      bool
		tlsAddr            string
		routes             []*Route
		debug              bool
		accessLog          bool
		strictSlash        bool
		logger             *slog.Logger
		renderer           Renderer
		corsEnabled        bool
		cors               Cors
		writeTimeout       int
		readTimeout        int
		idleTimeout        int
		optionsRegistered  map[string]bool
		openapiSpec        *openapi3.T
		openAPI            *OpenAPI
		openApiEnabled     bool
		maxMultipartMemory int64 // Maximum memory for multipart forms
		noRoute            HandleFunc
		noMethod           HandleFunc
	}

	Router struct {
		mux *mux.Router
	}
	OptionFunc func(*Okapi)

	// M is shortcut of map[string]any
	M map[string]any

	// Route defines the structure of a registered HTTP route in the framework.
	// It includes metadata used for routing, OpenAPI documentation, and middleware handling.
	Route struct {
		Name            string
		Path            string
		Method          string
		chain           chain
		tags            []string
		summary         string
		request         *openapi3.SchemaRef
		pathParams      []*openapi3.ParameterRef
		queryParams     []*openapi3.ParameterRef
		headers         []*openapi3.ParameterRef
		middlewares     []Middleware
		responseHeaders map[string]*openapi3.HeaderRef
		requiresAuth    bool
		deprecated      bool
		requestExample  map[string]interface{}
		responses       map[int]*openapi3.SchemaRef
		description     string
		disabled        bool
		handle          HandleFunc
		handler         HandleFunc
	}

	// Response interface defines the methods for writing HTTP responses.
	Response interface {
		http.ResponseWriter
		StatusCode() int
		Close()
		Hijack() (net.Conn, *bufio.ReadWriter, error)
	}
	// HandleFunc is a function type that takes a Context and returns an error.
	HandleFunc func(c Context) error

	Middleware func(next HandleFunc) HandleFunc
)

// chain is an interface that defines a method for chaining handlers.
type chain interface {
	next(hf HandleFunc) HandleFunc
}

// Disable marks the Route as disabled, causing it to return 404 Not Found.
// Returns the Route to allow method chaining.
func (r *Route) Disable() *Route {
	r.disabled = true
	return r
}

// Enable marks the Route as enabled, allowing it to handle requests normally.
// Returns the Route to allow method chaining.
func (r *Route) Enable() *Route {
	r.disabled = false
	return r
}

// SetDisabled sets the disabled state of the Route.
// When disabled is true, the route returns 404 Not Found.
// Returns the Route to allow method chaining.
func (r *Route) SetDisabled(disabled bool) *Route {
	r.disabled = disabled
	return r
}

// Deprecated marks the Route as deprecated.
// Returns the Route to allow method chaining.
func (r *Route) Deprecated() *Route {
	r.deprecated = true
	return r
}

// UseMiddleware registers one or more middleware functions to the Route.
func UseMiddleware(m ...Middleware) RouteOption {
	return func(r *Route) {
		if len(m) == 0 {
			return
		}
		r.middlewares = append(r.middlewares, m...)
	}
}

// Use registers one or more middleware functions to the Route.
func (r *Route) Use(m ...Middleware) {
	if len(m) == 0 {
		return
	}
	r.middlewares = append(r.middlewares, m...)
	r.handler = r.next(r.handle)
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

// WithMuxRouter sets the router for the Okapi instance
func WithMuxRouter(router *mux.Router) OptionFunc {
	return func(o *Okapi) {
		if router != nil {
			o.router.mux = router
		}
	}
}

// WithServer sets the HTTP server for the Okapi instance
func WithServer(server *http.Server) OptionFunc {
	return func(o *Okapi) {
		if server != nil {
			o.server = server
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
		o.server.WriteTimeout = secondsToDuration(t)
	}
}

// WithReadTimeout returns an OptionFunc that sets the read timeout
func WithReadTimeout(t int) OptionFunc {
	return func(o *Okapi) {
		o.readTimeout = t
		o.server.ReadTimeout = secondsToDuration(t)
	}
}

// WithIdleTimeout returns an OptionFunc that sets the idle timeout
func WithIdleTimeout(t int) OptionFunc {
	return func(o *Okapi) {
		o.idleTimeout = t
		o.server.IdleTimeout = secondsToDuration(t)
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
			port = defaultPort
		}
		host, _, err := net.SplitHostPort(o.server.Addr)
		if err != nil || host == "" {
			host = ""
		}
		o.server.Addr = net.JoinHostPort(host, strconv.Itoa(port))
	}
}

// WithAddr sets the server address
func WithAddr(addr string) OptionFunc {
	return func(o *Okapi) {
		if strings.TrimSpace(addr) == "" || addr == ":" {
			addr = defaultAddr
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			addr = net.JoinHostPort(addr, strconv.Itoa(defaultPort))
		}
		o.server.Addr = addr
	}
}

// WithRenderer sets a custom Renderer for the server.
//
// This allows you to define how templates or views are rendered in response handlers.
// You can implement the Renderer interface on your own type, or use the built-in
// RendererFunc adapter to provide an inline function.
//
// Example using a custom Renderer type:
//
//	type Template struct {
//		templates *template.Template
//	}
//
//	func (t *Template) Render(w io.Writer, name string, data interface{}, c okapi.Context) error {
//		return t.templates.ExecuteTemplate(w, name, data)
//	}
//
// o := okapi.New().WithRenderer(&Template{templates: template.Must(template.ParseGlob("public/views/*.html"))})
//
// Example using RendererFunc:
//
//	o := okapi.New().WithRenderer(okapi.RendererFunc(func(w io.Writer,
//	name string, data interface{}, c *okapi.Context) error {
//		tmpl, err := template.ParseFiles("public/views/" + name + ".html")
//		if err != nil {
//			return err
//		}
//		return tmpl.ExecuteTemplate(w, name, data)
//	}))
func WithRenderer(renderer Renderer) OptionFunc {
	return func(o *Okapi) {
		if renderer != nil {
			o.renderer = renderer
		}
	}
}

// WithOpenAPIDisabled disabled OpenAPI Docs
func WithOpenAPIDisabled() OptionFunc {
	return func(o *Okapi) {
		o.openApiEnabled = false
	}
}

// WithMaxMultipartMemory Maximum memory for multipart forms
func WithMaxMultipartMemory(max int64) OptionFunc {
	return func(o *Okapi) {
		o.maxMultipartMemory = max
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

// WithOpenAPIDisabled disabled OpenAPI Docs
func (o *Okapi) WithOpenAPIDisabled() *Okapi {
	return o.apply(WithOpenAPIDisabled())

}

// WithRenderer sets a custom Renderer for the server.
//
// This allows you to define how templates or views are rendered in response handlers.
// You can implement the Renderer interface on your own type, or use the built-in
// RendererFunc adapter to provide an inline function.
//
// Example using a custom Renderer type:
//
//	type Template struct {
//		templates *template.Template
//	}
//
//	func (t *Template) Render(w io.Writer, name string, data interface{}, c okapi.Context) error {
//		return t.templates.ExecuteTemplate(w, name, data)
//	}
//
// o := okapi.New().WithRenderer(&Template{templates: template.Must(template.ParseGlob("public/views/*.html"))})
//
// Example using RendererFunc:
//
//	o := okapi.New().WithRenderer(okapi.RendererFunc(func(w io.Writer,
//	name string, data interface{}, c *okapi.Context) error {
//		tmpl, err := template.ParseFiles("public/views/" + name + ".html")
//		if err != nil {
//			return err
//		}
//		return tmpl.ExecuteTemplate(w, name, data)
//	}))
func (o *Okapi) WithRenderer(renderer Renderer) *Okapi {
	return o.apply(WithRenderer(renderer))
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
func (o *Okapi) WithMaxMultipartMemory(max int64) *Okapi {
	return o.apply(WithMaxMultipartMemory(max))
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
		o.openAPI.License = config.License
		o.openAPI.Contact = config.Contact

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

// StatusCode returns the HTTP status code of the response writer.
func (r *response) StatusCode() int {
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

	o.applyServerConfig(o.server)

	if o.tlsServerConfig != nil {
		o.tlsServer.TLSConfig = o.tlsServerConfig
		o.tlsServer.Addr = o.tlsAddr
		o.applyServerConfig(o.tlsServer)
	}
	if o.openApiEnabled {
		o.WithOpenAPIDocs()
	}
	return o
}

// Start starts the Okapi server
func (o *Okapi) Start() error {
	return o.StartServer(o.server)
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
	if len(middlewares) == 0 {
		return
	}
	o.middlewares = append(o.middlewares, middlewares...)
}

// UseMiddleware registers a standard HTTP middleware function and integrates
// it into Okapi's middleware chain.
//
// This enables compatibility with existing middleware libraries that use the
// func(http.Handler) http.Handler pattern.
//
// Example:
//
//	okapi.UseMiddleware(func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.request) {
//	        w.Header().Set("X-Powered-By", "Okapi")
//	        next.ServeHTTP(w, r)
//	    })
//	})
//
// Internally, Okapi converts between http.Handler and HandleFunc to allow smooth interop.
func (o *Okapi) UseMiddleware(mw func(http.Handler) http.Handler) {
	o.Use(func(next HandleFunc) HandleFunc {
		// Convert HandleFunc to http.Handler
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := Context{
				request:  r,
				response: &response{writer: w},
				okapi:    o,
			}
			if err := next(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		// Apply standard middleware
		wrapped := mw(h)

		// Convert back to HandleFunc
		return func(ctx Context) error {
			wrapped.ServeHTTP(ctx.response, ctx.request)
			return nil
		}
	})
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
	printBanner()
	o.server = server
	server.Handler = o
	o.router.mux.StrictSlash(o.strictSlash)
	o.context.okapi = o
	o.applyCommon()
	_, _ = fmt.Fprintf(defaultWriter, "Starting HTTP server at %s\n", o.server.Addr)
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

		o.tlsServer.Handler = o
		_, _ = fmt.Fprintf(defaultWriter, "Starting HTTP server at %s\n", o.tlsServer.Addr)
		return o.tlsServer.ListenAndServeTLS("", "")
	}

	// Default HTTP only
	return server.ListenAndServe()
}

// Stop gracefully shuts down the Okapi HTTP and HTTPS server(s)
func (o *Okapi) Stop() error {
	if o.server != nil {
		_, _ = fmt.Fprintf(defaultWriter, "[Okapi] Gracefully shutting down HTTP server at %s\n", o.server.Addr)
		if err := o.Shutdown(o.server); err != nil {
			return fmt.Errorf("HTTP shutdown error at %s: %w", o.server.Addr, err)
		}
		o.server = nil
	}

	if o.withTlsServer && o.tlsServerConfig != nil && o.tlsServer != nil {
		_, _ = fmt.Fprintf(defaultWriter, "[Okapi] Gracefully shutting down HTTPS server at %s\n", o.tlsServer.Addr)
		if err := o.Shutdown(o.tlsServer); err != nil {
			return fmt.Errorf("HTTPS shutdown error at %s: %w", o.tlsServer.Addr, err)
		}
		o.tlsServer = nil
	}

	return nil
}

// Shutdown performs graceful shutdown of the provided server using a background context
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
	return o.addRoute(GET, path, nil, h, opts...)
}

// Post registers a new POST route with the given path and handler function.
// Returns the created *Route for possible chaining or modification.
func (o *Okapi) Post(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(POST, path, nil, h, opts...)
}

// Put registers a new PUT route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Put(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(PUT, path, nil, h, opts...)
}

// Delete registers a new DELETE route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Delete(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodDelete, path, nil, h, opts...)
}

// Patch registers a new PATCH route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Patch(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(PATCH, path, nil, h, opts...)
}

// Options registers a new OPTIONS route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Options(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodOptions, path, nil, h, opts...)
}

// Head registers a new HEAD route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Head(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(HEAD, path, nil, h, opts...)
}

// Connect registers a new CONNECT route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Connect(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(http.MethodConnect, path, nil, h, opts...)
}

// Trace registers a new TRACE route with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Trace(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute(TRACE, path, nil, h, opts...)
}

// Any registers a route that matches any HTTP method with the given path and handler function.
// Returns the created *Route
func (o *Okapi) Any(path string, h HandleFunc, opts ...RouteOption) *Route {
	return o.addRoute("", path, nil, h, opts...)
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
func (o *Okapi) addRoute(method, path string, tags []string, h HandleFunc, opts ...RouteOption) *Route {
	if path == "" {
		panic("Path cannot be empty")
	}
	if len(tags) == 0 {
		tags = []string{"default"}
	}
	path = normalizeRoutePath(path)
	route := &Route{
		Name:      handleName(h),
		Path:      path,
		Method:    method,
		tags:      tags,
		handle:    h,
		chain:     o,
		responses: make(map[int]*openapi3.SchemaRef),
	}
	for _, opt := range opts {
		opt(route)
	}
	o.routes = append(o.routes, route)
	route.handler = route.next(h)
	// Main handler
	o.router.mux.StrictSlash(o.strictSlash).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{
			request:  r,
			response: &response{writer: w},
			okapi:    o,
		}
		if route.disabled {
			http.Error(w, "404 Not Found", http.StatusNotFound)
			return
		}
		if err := route.handler(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}).Methods(method)
	// Register OPTIONS handler only once per path if CORS is enabled
	o.registerOptionsHandler(path)
	return route
}

// Handle registers a new route with the given HTTP method, path, and Okapi-style handler function.
//
// It performs the following steps:
//  1. Normalizes the route path
//  2. Creates and configures a new Route instance
//  3. Applies all registered middleware to the handler
//  4. Registers the route with the underlying router (with method filtering)
//  5. Adds centralized error handling for the route
//
// Parameters:
//   - method: HTTP method (e.g., "GET", "POST", "PUT")
//   - path:   URL path pattern (supports path parameters, e.g., /users/:id)
//   - h:      A handler function using Okapi's Context abstraction
//   - opts:   Optional route metadata (e.g., OpenAPI summary, description, tags)
//
// Middleware:
//
//	All middleware registered via Use() or UseMiddleware() will be applied in order,
//	wrapping around the handler.
//
// Error Handling:
//
//	Any non-nil error returned by the handler will automatically result in a
//	500 Internal Server Error response.
//
// Example:
//
//	okapi.Handle("GET", "/users/:id", func(c Context) error {
//	    id := c.Param("id")
//	    // process request...
//	    return nil
//	})
func (o *Okapi) Handle(method, path string, h HandleFunc, opts ...RouteOption) {
	o.addRoute(method, path, nil, h, opts...)
}

// HandleHTTP registers a new route using a standard http.Handler.
//
// It wraps the provided http.Handler into Okapi's internal HandleFunc signature
// and processes it as if it were registered via Handle.
//
// Parameters:
//   - method: HTTP method (e.g., "GET", "POST", "DELETE")
//   - path:   URL path pattern (supports dynamic segments)
//   - h:      A standard http.Handler (or http.HandlerFunc)
//   - opts:   Optional route metadata (e.g., OpenAPI summary, description, tags)
//
// Differences from Handle:
//   - Uses the standard http.Handler interface
//   - Middleware is still applied
//   - Errors must be handled inside the handler itself (Okapi will not capture them)
//
// Example:
//
//	okapi.HandleHTTP("GET", "/static", http.FileServer(http.Dir("./public")))
func (o *Okapi) HandleHTTP(method, path string, h http.Handler, opts ...RouteOption) {
	o.addRoute(method, path, nil, o.wrapHTTPHandler(h), opts...)
}

// HandleStd is a convenience method for registering handlers using the standard
// http.HandlerFunc signature (func(http.ResponseWriter, *http.Request)).
//
// Internally, it wraps the handler into http.Handler and delegates to HandleHTTP.
//
// Example:
//
//	okapi.HandleStd("GET", "/greet", func(w http.ResponseWriter, r *http.Request) {
//	    w.Write([]byte("Hello from Okapi!"))
//	})
//
// This handler will still benefit from:
//   - All registered middleware
//   - Automatic route and CORS registration
func (o *Okapi) HandleStd(method, path string, h func(http.ResponseWriter, *http.Request), opts ...RouteOption) {
	o.HandleHTTP(method, path, http.HandlerFunc(h), opts...)
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
		request:  r,
		response: &response{writer: w},
		okapi:    o,
	}
	handler := func(c *Context) {
		o.router.mux.ServeHTTP(c.response, c.request)
	}
	handler(ctx)
}

// next applies the middlewares in correct order
func (o *Okapi) next(h HandleFunc) HandleFunc {
	// Start with the original handler
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		h = o.middlewares[i](h)
	}
	return h
}

// next applies the middlewares in correct order for a specific route
func (r *Route) next(h HandleFunc) HandleFunc {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return r.chain.next(h)
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
func (o *Okapi) Group(prefix string, middlewares ...Middleware) *Group {
	if len(prefix) == 0 {
		panic("Group prefix cannot be empty")
	}
	group := &Group{
		Prefix:      prefix,
		okapi:       o,
		middlewares: middlewares,
	}
	return group
}

// initConfig initializes a new Okapi instance.
func initConfig(options ...OptionFunc) *Okapi {
	server := &http.Server{
		Addr: defaultAddr,
	}

	o := &Okapi{
		context: &Context{
			request:  new(http.Request),
			response: &response{},
			store:    newStoreData(),
		},
		router:             newRouter(),
		server:             server,
		tlsServer:          &http.Server{},
		logger:             slog.Default(),
		accessLog:          true,
		middlewares:        []Middleware{handleAccessLog},
		optionsRegistered:  make(map[string]bool),
		maxMultipartMemory: defaultMaxMemory,
		cors:               Cors{},
		openAPI: &OpenAPI{
			Title:      okapiName,
			Version:    "1.0.0",
			PathPrefix: openApiDocPrefix,
			Servers:    Servers{{}},
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
func (o *Okapi) applyCommon() {
	if o.noRoute != nil {
		o.router.mux.NotFoundHandler = o.wrapHandleFunc(o.noRoute)
	}
	if o.noMethod != nil {
		o.router.mux.MethodNotAllowedHandler = o.wrapHandleFunc(o.noMethod)
	}
}

// NoRoute sets a custom handler to be executed when no matching route is found.
//
// This function allows you to define a fallback handler for unmatched routes (404).
// It is useful for returning custom error pages or JSON responses when a route doesn't exist.
//
// Example:
//
//	o.NoRoute(func(c okapi.Context) error {
//		return c.AbortNotFound("Custom 404 - Not found")
//	})
func (o *Okapi) NoRoute(h HandleFunc) {
	o.noRoute = h
}

// NoMethod sets a custom handler to be executed when the HTTP method is not allowed.
//
// This function is triggered when the request path exists but the method (e.g., POST, GET) is not allowed.
// It enables you to define a consistent response for unsupported HTTP methods (405).
//
// Example:
//
//	o.NoMethod(func(c okapi.Context) error {
//	 return c.AbortMethodNotAllowed("Custom 405 - Method Not Allowed")
//	})
func (o *Okapi) NoMethod(h HandleFunc) {
	o.noMethod = h
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
			return next(c)
		}
		startTime := time.Now()
		err := next(c)
		status := c.response.StatusCode()
		duration := goutils.FormatDuration(time.Since(startTime), 2)

		logger := c.okapi.logger
		args := []any{
			"method", c.request.Method,
			"url", c.request.URL.Path,
			"ip", c.RealIP(),
			"host", c.request.Host,
			"status", status,
			"duration", duration,
			"referer", c.request.Referer(),
			"user_agent", c.request.UserAgent(),
		}
		switch {
		case status >= 500:
			logger.Error("[okapi] Incoming request", args...)
		case status >= 400:
			logger.Warn("[okapi] Incoming request", args...)
		default:
			logger.Info("[okapi] Incoming request", args...)
		}
		return err
	}
}

func printBanner() {
	fmt.Println(banner)
}
func (o *Okapi) wrapHandleFunc(h HandleFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := Context{
			request:  r,
			response: &response{writer: w},
			okapi:    o,
		}
		if err := h(ctx); err != nil {
			o.logger.Error("handler error", slog.String("error", err.Error()))
			http.Error(w, err.Error(), http.StatusInternalServerError)

		}
	})
}
func (o *Okapi) wrapHTTPHandler(h http.Handler) HandleFunc {
	return func(ctx Context) error {
		h.ServeHTTP(ctx.response, ctx.request)
		return nil
	}
}

// Register registers a list of RouteDefinition to the Okapi instance.
// This method allows you to define multiple routes in a single call, which can be useful for
// organizing your routes in a more structured way.
// Example:
//
//	routes := []okapi.RouteDefinition{
//	    {
//	        Method:  "GET",
//	        Path:    "/example",
//	        Handler: exampleHandler,
//			Middlewares: []okapi.Middleware{customMiddleware}
//	        Options: []okapi.RouteOption{
//	            okapi.DocSummary("Example GET request"),
//	        },
//	    },
//	    {
//	        Method:  "POST",
//	        Path:    "/example",
//	        Handler: exampleHandler,
//	        Options: []okapi.RouteOption{
//	            okapi.DocSummary("Example POST request"),
//	        },
//	    },
//	}
//	// Create a new Okapi instance
//	app := okapi.New()
//	app.Register(routes...)
func (o *Okapi) Register(routes ...RouteDefinition) {
	RegisterRoutes(o, routes)
}
