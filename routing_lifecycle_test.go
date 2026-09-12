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
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// serveTestRequest dispatches a request through the app without a listener.
// headers are name/value pairs.
func serveTestRequest(o *Okapi, method, target string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	rec := httptest.NewRecorder()
	o.ServeHTTP(rec, req)
	return rec
}

func okHandler(c *Context) error {
	return c.JSON(http.StatusOK, M{"ok": true})
}

func denyMiddleware(c *Context) error {
	return c.Error(http.StatusUnauthorized, "denied")
}

// TestGroupUseAppliesToEarlierRoutes guards against group middleware being
// copied into routes at registration, which left routes registered before Use
// unprotected.
func TestGroupUseAppliesToEarlierRoutes(t *testing.T) {
	o := New()
	admin := o.Group("/admin")
	admin.Get("/users", okHandler)
	admin.Use(denyMiddleware)
	admin.Get("/other", okHandler)

	for _, path := range []string{"/admin/users", "/admin/other"} {
		if rec := serveTestRequest(o, http.MethodGet, path); rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s = %d, want 401", path, rec.Code)
		}
	}
}

func TestSubgroupInheritsLaterParentMiddleware(t *testing.T) {
	o := New()
	api := o.Group("/api")
	v1 := api.Group("/v1")
	v1.Get("/orders", okHandler)
	api.Use(denyMiddleware)

	if rec := serveTestRequest(o, http.MethodGet, "/api/v1/orders"); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/orders = %d, want 401", rec.Code)
	}
}

func TestGroupDisableAfterRegistration(t *testing.T) {
	o := New()
	admin := o.Group("/admin")
	admin.Get("/x", okHandler)
	nested := admin.Group("/nested")
	nested.Get("/y", okHandler)

	admin.Disable()
	for _, path := range []string{"/admin/x", "/admin/nested/y"} {
		if rec := serveTestRequest(o, http.MethodGet, path); rec.Code != http.StatusNotFound {
			t.Errorf("disabled: GET %s = %d, want 404", path, rec.Code)
		}
	}

	admin.Enable()
	if rec := serveTestRequest(o, http.MethodGet, "/admin/nested/y"); rec.Code != http.StatusOK {
		t.Errorf("re-enabled: GET /admin/nested/y = %d, want 200", rec.Code)
	}
}

// TestGroupsFromSharedSliceDoNotAlias guards against Okapi.Group keeping the
// caller's slice, which let Use on one group write into another's middlewares.
func TestGroupsFromSharedSliceDoNotAlias(t *testing.T) {
	mark := func(name string) Middleware {
		return func(c *Context) error {
			c.response.Header().Add("X-Mw", name)
			return c.Next()
		}
	}
	common := make([]Middleware, 1, 4)
	common[0] = mark("common")

	o := New()
	a := o.Group("/a", common...)
	b := o.Group("/b", common...)
	a.Use(mark("a"))
	b.Use(mark("b"))
	a.Get("/x", okHandler)
	b.Get("/x", okHandler)

	got := strings.Join(serveTestRequest(o, http.MethodGet, "/a/x").Header().Values("X-Mw"), ",")
	if got != "common,a" {
		t.Errorf("GET /a/x ran middlewares %q, want %q", got, "common,a")
	}
}

// TestGroupRegisterAppliesGroupMiddleware guards against Group.Register calling
// addRoute directly, which skipped the group's middlewares and the definition's
// documentation fields.
func TestGroupRegisterAppliesGroupMiddleware(t *testing.T) {
	o := New()
	api := o.Group("/api", denyMiddleware)
	defs := []RouteDefinition{{Method: http.MethodGet, Path: "/secret", Handler: okHandler, Summary: "Secret"}}
	api.Register(defs...)

	if rec := serveTestRequest(o, http.MethodGet, "/api/secret"); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/secret = %d, want 401", rec.Code)
	}
	if defs[0].Group != nil {
		t.Error("Register modified the caller's route definition")
	}
	for _, r := range o.routes {
		if r.Path == "/api/secret" && r.summary != "Secret" {
			t.Errorf("route summary = %q, want %q", r.summary, "Secret")
		}
	}
}

func TestAnyMatchesEveryMethod(t *testing.T) {
	o := New()
	o.Any("/health", okHandler)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete, "PROPFIND"} {
		if rec := serveTestRequest(o, method, "/health"); rec.Code != http.StatusOK {
			t.Errorf("%s /health = %d, want 200", method, rec.Code)
		}
	}
}

// TestExplicitOptionsRouteWithCors guards against the automatic preflight
// handler and a user-registered OPTIONS route colliding in the router.
func TestExplicitOptionsRouteWithCors(t *testing.T) {
	const (
		origin = "https://options.example"
		marker = "user-options-handler"
	)
	custom := func(c *Context) error {
		return c.JSON(http.StatusOK, M{"handler": marker})
	}
	o := New(WithCors(Cors{AllowedOrigins: []string{origin}}))
	o.Get("/x", okHandler)
	o.Options("/x", custom)
	o.Options("/y", custom)

	rec := serveTestRequest(o, http.MethodOptions, "/x")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), marker) {
		t.Errorf("plain OPTIONS /x = %d %q, want the user handler", rec.Code, rec.Body.String())
	}

	rec = serveTestRequest(o, http.MethodOptions, "/x",
		"Origin", origin,
		"Access-Control-Request-Method", http.MethodGet)
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight OPTIONS /x = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != origin {
		t.Errorf("preflight missing Access-Control-Allow-Origin: %v", rec.Header())
	}

	if rec := serveTestRequest(o, http.MethodOptions, "/y"); rec.Code != http.StatusOK {
		t.Errorf("OPTIONS /y = %d, want 200", rec.Code)
	}
}

func TestWithTLSAppliesToServer(t *testing.T) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	o := New(WithTLS(cfg))
	if o.server.TLSConfig != cfg {
		t.Error("WithTLS did not set the HTTP server's TLSConfig; the server would serve plain HTTP")
	}
}

// TestWithServerKeepsTimeouts guards against With overwriting the timeouts of a
// server passed through WithServer.
func TestWithServerKeepsTimeouts(t *testing.T) {
	s := &http.Server{ReadTimeout: 5 * time.Second, WriteTimeout: 7 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	o := New(WithServer(s))
	if o.server.ReadTimeout != 5*time.Second || o.server.WriteTimeout != 7*time.Second || o.server.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("timeouts = read %v, write %v, readHeader %v; want 5s, 7s, 2s",
			o.server.ReadTimeout, o.server.WriteTimeout, o.server.ReadHeaderTimeout)
	}
	if o.server.IdleTimeout == 0 {
		t.Error("IdleTimeout left unset; the default should fill a zero field")
	}

	o = New(WithServer(&http.Server{}), WithReadTimeout(3))
	if o.server.ReadTimeout != 3*time.Second {
		t.Errorf("ReadTimeout = %v, want 3s from WithReadTimeout", o.server.ReadTimeout)
	}
}

// TestHandlerErrorUsesErrorHandler guards against returned errors being written
// as plain text containing err.Error(), bypassing the configured ErrorHandler.
func TestHandlerErrorUsesErrorHandler(t *testing.T) {
	secret := "pq: password authentication failed for user admin"
	failing := func(c *Context) error { return errors.New(secret) }

	o := New()
	o.Get("/fail", failing)
	rec := serveTestRequest(o, http.MethodGet, "/fail")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("response leaks the handler error: %q", rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want the error handler's JSON", rec.Header().Get("Content-Type"))
	}

	called := false
	o = New()
	o.errorHandler = func(c *Context, code int, message string, err error) error {
		called = true
		return c.Error(code, message)
	}
	o.Get("/fail", failing)
	if rec := serveTestRequest(o, http.MethodGet, "/fail"); rec.Code != http.StatusInternalServerError || !called {
		t.Errorf("custom error handler called = %v, status = %d", called, rec.Code)
	}

	o = New()
	o.Get("/late", func(c *Context) error {
		_ = c.JSON(http.StatusOK, M{"ok": true})
		return errors.New(secret)
	})
	rec = serveTestRequest(o, http.MethodGet, "/late")
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "password") {
		t.Errorf("error after write: %d %q, want the original 200 response untouched", rec.Code, rec.Body.String())
	}
}

func TestStaticFSDoesNotListDirectories(t *testing.T) {
	o := New()
	o.StaticFS("/assets", http.FS(fstest.MapFS{
		"secret/backup.sql": {Data: []byte("DROP TABLE users;")},
	}))

	rec := serveTestRequest(o, http.MethodGet, "/assets/secret/")
	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "backup.sql") {
		t.Errorf("GET /assets/secret/ = %d %q, want no directory listing", rec.Code, rec.Body.String())
	}
}

// TestStopDrainsInFlightRequests guards against shutdown cancelling request
// contexts before the server has drained, which failed every in-flight request.
// Run with -race: Start and Stop touch the server from different goroutines.
func TestStopDrainsInFlightRequests(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	o := New(WithAddr("127.0.0.1:" + strconv.Itoa(port)))
	started := make(chan struct{})
	o.Get("/slow", func(c *Context) error {
		close(started)
		select {
		case <-c.Request().Context().Done():
			return c.Error(http.StatusServiceUnavailable, "cancelled")
		case <-time.After(300 * time.Millisecond):
			return c.JSON(http.StatusOK, M{"ok": true})
		}
	})

	go func() { _ = o.Start() }()
	if o.WaitForServer(2*time.Second) == "" {
		t.Fatal("server did not start")
	}

	type result struct {
		code int
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/slow")
		if err != nil {
			done <- result{err: err}
			return
		}
		_ = resp.Body.Close()
		done <- result{code: resp.StatusCode}
	}()

	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := o.StopWithContext(ctx); err != nil {
		t.Fatalf("StopWithContext: %v", err)
	}

	res := <-done
	if res.err != nil || res.code != http.StatusOK {
		t.Errorf("in-flight request = %d, %v; want 200", res.code, res.err)
	}
}

// TestSSEStreamEndsOnShutdown guards the counterpart of draining: a stream never
// finishes on its own, so it must end when shutdown begins rather than hold
// StopWithContext until its deadline.
func TestSSEStreamEndsOnShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	o := New(WithAddr("127.0.0.1:" + strconv.Itoa(port)))
	streaming := make(chan struct{})
	streamErr := make(chan error, 1)
	o.Get("/events", func(c *Context) error {
		close(streaming)
		err := c.SSEStream(c.Request().Context(), make(chan Message))
		streamErr <- err
		return err
	})

	go func() { _ = o.Start() }()
	if o.WaitForServer(2*time.Second) == "" {
		t.Fatal("server did not start")
	}

	go func() {
		resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/events")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	<-streaming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	if err := o.StopWithContext(ctx); err != nil {
		t.Fatalf("StopWithContext: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("shutdown took %v; the open stream held it", elapsed)
	}
	if err := <-streamErr; err != nil {
		t.Errorf("SSEStream returned %v, want nil on shutdown", err)
	}
}

// TestRouteDefinitionDisabled covers RouteDefinition.Disabled on the root
// instance and through a group: the route answers 404 and is left out of the
// OpenAPI document, as Route.Disable does.
func TestRouteDefinitionDisabled(t *testing.T) {
	const off = "/off"
	o := New()
	RegisterRoutes(o, []RouteDefinition{
		{Method: http.MethodGet, Path: off, Handler: okHandler, Disabled: true},
		{Method: http.MethodGet, Path: "/on", Handler: okHandler},
	})
	o.Group("/api").Register(RouteDefinition{Method: http.MethodGet, Path: off, Handler: okHandler, Disabled: true})

	for path, want := range map[string]int{off: http.StatusNotFound, "/on": http.StatusOK, "/api" + off: http.StatusNotFound} {
		if rec := serveTestRequest(o, http.MethodGet, path); rec.Code != want {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, want)
		}
	}

	o.WithOpenAPIDocs()
	for _, path := range []string{off, "/api" + off} {
		if o.openapiSpec.Paths.Find(path) != nil {
			t.Errorf("disabled route %s is documented", path)
		}
	}
	if o.openapiSpec.Paths.Find("/on") == nil {
		t.Error("enabled route /on is missing from the OpenAPI document")
	}
}

// TestAnyRouteWithCors guards plain OPTIONS requests on an Any route, which the
// CORS preflight handler answered with 204 instead of dispatching to the route.
func TestAnyRouteWithCors(t *testing.T) {
	const origin = "https://any.example"
	o := New(WithCors(Cors{AllowedOrigins: []string{origin}}))
	o.Any("/proxy", func(c *Context) error {
		return c.JSON(http.StatusOK, M{"method": c.Request().Method})
	})

	rec := serveTestRequest(o, http.MethodOptions, "/proxy")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), http.MethodOptions) {
		t.Errorf("plain OPTIONS /proxy = %d %q, want the Any handler", rec.Code, rec.Body.String())
	}

	rec = serveTestRequest(o, http.MethodOptions, "/proxy",
		"Origin", origin,
		"Access-Control-Request-Method", http.MethodPut)
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight OPTIONS /proxy = %d, want 204", rec.Code)
	}
	if allow := rec.Header().Get("Access-Control-Allow-Methods"); strings.Contains(allow, "*") || !strings.Contains(allow, http.MethodPut) {
		t.Errorf("Access-Control-Allow-Methods = %q, want method names including PUT", allow)
	}
}

// TestRegisterRoutesAnyMethod covers ANY and "*" in RouteDefinition.Method, which
// RegisterRoutes rejected as an unsupported method.
func TestRegisterRoutesAnyMethod(t *testing.T) {
	o := New()
	api := o.Group("/api")
	RegisterRoutes(o, []RouteDefinition{
		{Method: "any", Path: "/a", Handler: okHandler},
		{Method: "*", Path: "/b", Handler: okHandler, Group: api},
	})

	for _, path := range []string{"/a", "/api/b"} {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			if rec := serveTestRequest(o, method, path); rec.Code != http.StatusOK {
				t.Errorf("%s %s = %d, want 200", method, path, rec.Code)
			}
		}
	}
}
