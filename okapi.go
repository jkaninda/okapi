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
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	goutils "github.com/jkaninda/go-utils"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"
)

// Constants for Error Handling
var (
	ErrNotFound               = errors.New("not found")
	ErrFailedToParseMultiPart = errors.New("failed to parse multipart data")
	ErrInvalidMultiPartData   = errors.New("invalid multipart data")
)

var (
	DefaultWriter      io.Writer = os.Stdout
	DefaultErrorWriter io.Writer = os.Stderr
	DefaultPort        int       = 8080
	DefaultAddr        string    = ":8080"
)

type (
	Okapi struct {
		context     *Context
		router      *Router
		middlewares []Middleware
		Server      *http.Server
		TLSServer   *http.Server
		routes      []*Route
		debug       bool
		accessLog   bool
		strictSlash bool
		logger      *slog.Logger
		Renderer    Renderer
	}
	Router struct {
		mux *mux.Router
	}
	OptionFunc func(*Okapi)

	// M is shortcut of map[string]any
	M map[string]any

	Route struct {
		Name   string
		Path   string
		Method string
		Handle HandleFunc
		chain  chain
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
		o.router.mux = mux
	}
}

// WithTLSServer sets the TLS server for the Okapi instance
func WithTLSServer(server *http.Server) OptionFunc {
	return func(o *Okapi) {
		o.TLSServer = server
	}
}

// WithServer sets the HTTP server for the Okapi instance
func WithServer(server *http.Server) OptionFunc {
	return func(o *Okapi) {
		o.Server = server
	}
}

// WithLogger sets the logger for the Okapi instance
func WithLogger(logger *slog.Logger) OptionFunc {
	return func(o *Okapi) {
		o.logger = logger
	}
}

// WithStrictSlash sets the strict slash mode for the Okapi instance
func WithStrictSlash(strict bool) OptionFunc {
	return func(o *Okapi) {
		o.strictSlash = strict
	}
}

// WithDebug sets the debug mode for the Okapi instance
func WithDebug() OptionFunc {
	return func(o *Okapi) {
		o.debug = true
		o.accessLog = true
	}
}

// WithAccessLogDisabled disables the access log for the Okapi instance
func WithAccessLogDisabled() OptionFunc {
	return func(o *Okapi) {
		o.accessLog = false
	}
}

// WithPort sets the port for the Okapi instance
func WithPort(port int) OptionFunc {
	return func(o *Okapi) {
		if port <= 0 {
			port = DefaultPort
		}
		o.Server.Addr = ":" + strconv.Itoa(port)
	}
}

// WithAddr sets the address for the Okapi instance
func WithAddr(addr string) OptionFunc {
	return func(o *Okapi) {
		if len(addr) == 0 {
			addr = DefaultAddr
		}
		if _, _, err := net.SplitHostPort(addr); err != nil {
			// If no port is specified, use the default port
			if addr == "" {
				addr = DefaultAddr
			} else {
				// Append the default port if no port is specified
				addr = net.JoinHostPort(addr, strconv.Itoa(DefaultPort))
			}
		} else if addr == ":" {
			// If addr is just a colon, set it to the default address
			addr = DefaultAddr
		}
		o.Server.Addr = addr
	}
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

// New creates a new Okapi instance
func New(options ...OptionFunc) (e *Okapi) {
	return Default(options...)

}

// Default creates a new Okapi instance with default settings
func Default(options ...OptionFunc) *Okapi {
	server := &http.Server{
		Addr: DefaultAddr,
	}
	o := &Okapi{
		context: &Context{
			Request:            new(http.Request),
			Response:           &response{},
			MaxMultipartMemory: defaultMaxMemory,
		},
		router:      newRouter(),
		Server:      server,
		logger:      slog.Default(),
		accessLog:   true,
		middlewares: []Middleware{handleAccessLog},
	}
	return o.With(options...)
}

// With applies the provided options to the Okapi instance
func (o *Okapi) With(options ...OptionFunc) *Okapi {
	for _, option := range options {
		option(o)
	}
	if o.debug {
		o.logger = slog.New(slog.NewJSONHandler(DefaultWriter,
			&slog.HandlerOptions{Level: slog.LevelDebug},
		))
	}
	return o
}

// Start starts the Okapi server
func (o *Okapi) Start() error {
	return o.StartServer(o.Server)
}

// Use adds middleware to the Okapi instance
func (o *Okapi) Use(middlewares ...Middleware) {
	o.middlewares = append(o.middlewares, middlewares...)
}

// StartServer starts the Okapi server with the specified HTTP server
func (o *Okapi) StartServer(server *http.Server) error {
	// Validate the server address
	if !ValidateAddr(server.Addr) {
		o.logger.Error("Invalid server address", slog.String("addr", server.Addr))
		panic("Invalid server address")
	}
	o.Server = server
	server.Handler = o
	o.router.mux.StrictSlash(o.strictSlash)
	o.context.okapi = o
	_, _ = fmt.Fprintf(DefaultWriter, "Starting Server on %s\n", o.Server.Addr)
	return server.ListenAndServe()
}

// Stop stops the Okapi server
func (o *Okapi) Stop() {
	o.logger.Info("Stopping Server on...")
	err := o.Server.Shutdown(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	o.Server = nil
}
func (o *Okapi) Shutdown(server *http.Server) error {
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

func (o *Okapi) Get(path string, h HandleFunc) *Route  { return o.addRoute(GET, path, h) }
func (o *Okapi) Post(path string, h HandleFunc) *Route { return o.addRoute(POST, path, h) }
func (o *Okapi) Put(path string, h HandleFunc) *Route  { return o.addRoute(PUT, path, h) }
func (o *Okapi) Delete(path string, h HandleFunc) *Route {
	return o.addRoute(http.MethodDelete, path, h)
}
func (o *Okapi) Patch(path string, h HandleFunc) *Route { return o.addRoute(PATCH, path, h) }
func (o *Okapi) Options(path string, h HandleFunc) *Route {
	return o.addRoute(http.MethodOptions, path, h)
}
func (o *Okapi) Head(path string, h HandleFunc) *Route { return o.addRoute(HEAD, path, h) }
func (o *Okapi) Connect(path string, h HandleFunc) *Route {
	return o.addRoute(http.MethodConnect, path, h)
}
func (o *Okapi) Trace(path string, h HandleFunc) *Route { return o.addRoute(TRACE, path, h) }
func (o *Okapi) Any(path string, h HandleFunc) *Route   { return o.addRoute("", path, h) }

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
func (o *Okapi) addRoute(method, path string, h HandleFunc) *Route {
	if path == "" {
		panic("Path cannot be empty")
	}
	path = normalizeRoutePath(path)
	route := &Route{
		Name:   handleName(h),
		Path:   path,
		Method: method,
		Handle: h,
		chain:  o,
	}
	o.routes = append(o.routes, route)
	handler := o.Next(h)

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

	return route
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
	next := h
	// Apply middlewares in the order they were added
	for _, m := range o.middlewares {
		next = m(next)
	}
	return next
}

// NewRouter

// HandleFunc adds a route with a custom handler function
func (o *Okapi) HandleFunc(method, path string, h HandleFunc) {
	path = normalizeRoutePath(path)
	route := &Route{
		Name:   handleName(h),
		Path:   path,
		Method: method,
		Handle: h,
		chain:  o,
	}
	o.routes = append(o.routes, route)

	// Apply middlewares to the handler
	handler := o.Next(h)

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
}
func (o *Okapi) Routes() []Route {
	routes := make([]Route, 0, len(o.routes))
	for _, route := range o.routes {
		routes = append(routes, *route)
	}
	return routes
}

// Group creates a new group with the specified path
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
		slog.Info("[Okapi]",
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
