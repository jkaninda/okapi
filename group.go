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

import "net/http"

type Group struct {
	// Prefix is the base path for all routes in this group.
	Prefix string
	// Tags is an optional tag for the group, used for documentation purposes.
	Tags        []string
	disabled    bool
	bearerAuth  bool
	basicAuth   bool
	deprecated  bool
	middlewares []Middleware
	okapi       *Okapi
	security    []map[string][]string
}

// newGroup creates a new route group with the specified base path, Okapi reference,
// and optional middlewares.
func newGroup(basePath string, disabled bool, okapi *Okapi, middlewares ...Middleware) *Group {
	mws := append([]Middleware{}, middlewares...)
	return &Group{
		Prefix:      basePath,
		middlewares: mws,
		okapi:       okapi,
		disabled:    disabled,
	}
}
func NewGroup(basePath string, okapi *Okapi, middlewares ...Middleware) *Group {
	if okapi == nil {
		panic("okapi instance cannot be nil")
	}
	if basePath == "" {
		panic("Prefix cannot be empty")
	}
	return newGroup(basePath, false, okapi, middlewares...)
}

// Disable marks the Group as disabled, causing all routes within it to return 404 Not Found.
// Returns the Group to allow method chaining.
func (g *Group) Disable() *Group {
	g.disabled = true
	return g
}

// Enable marks the Group as enabled, allowing all routes within it to handle requests normally.
// Returns the Group to allow method chaining.
func (g *Group) Enable() *Group {
	g.disabled = false
	return g
}

// setDisabled sets the unregistered state of the Group.
// When unregistered is true, all routes in the group return 404 Not Found.
// Returns the Group to allow method chaining.
func (g *Group) setDisabled(disabled bool) *Group {
	g.disabled = disabled
	return g
}

// WithBearerAuth marks the Group as requiring Bearer authentication for its routes.
// Returns the Group to allow method chaining.
func (g *Group) WithBearerAuth() *Group {
	g.bearerAuth = true
	return g
}
func (g *Group) WithBasicAuth() *Group {
	g.basicAuth = true
	return g
}

// WithTags sets the tags for the Group, which can be used for documentation purposes.
func (g *Group) WithTags(tags []string) *Group {
	g.Tags = tags
	return g
}

// Deprecated marks the Group as deprecated for its routes.
// Returns the Group to allow method chaining.
func (g *Group) Deprecated() *Group {
	g.deprecated = true
	return g
}

// WithSecurity sets the security requirements for the Group's routes.
func (g *Group) WithSecurity(security []map[string][]string) *Group {
	g.security = security
	return g
}

// Okapi returns the parent Okapi instance associated with this group.
func (g *Group) Okapi() *Okapi {
	return g.okapi
}

// Use adds one or more middlewares to the group's middleware chain.
// These middlewares will be executed in the order they are added,
// before the route handler for all routes within this group.
// Middlewares are inherited by any subgroups created from this group.
func (g *Group) Use(m ...Middleware) {
	if len(m) == 0 {
		return
	}
	g.middlewares = append(g.middlewares, m...)
}

// add is an internal method that handles route registration with the combined
// middlewares from both the group and parent Okapi instance.
func (g *Group) add(method, path string, h HandlerFunc, opts ...RouteOption) *Route {
	if g.okapi == nil {
		panic("okapi instance is nil, cannot register route")
	}
	fullPath := joinPaths(g.Prefix, path)
	// Wrap handler with combined middlewares
	finalHandler := h
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		finalHandler = g.middlewares[i](finalHandler)
	}
	tags := g.Tags
	if len(tags) == 0 {
		tags = []string{g.Prefix}
	}
	// Register the route with the joined base path and route path
	return g.okapi.addRoute(method, fullPath, tags, finalHandler, opts...).setDisabled(g.disabled)
}

// handle is a helper method that delegates to add with the given HTTP method.
func (g *Group) handle(method, path string, h HandlerFunc, opts ...RouteOption) *Route {
	if g.bearerAuth {
		opts = append(opts, DocBearerAuth())
	}
	if g.basicAuth {
		opts = append(opts, DocBasicAuth())
	}
	if g.deprecated {
		opts = append(opts, DocDeprecated())
	}
	if len(g.Tags) != 0 {
		for _, tag := range g.Tags {
			if tag == "" {
				continue // Skip empty tags
			}
			opts = append(opts, DocTag(tag))
		}
	}
	if len(g.security) > 0 {
		opts = append(opts, withSecurity(g.security))
	}
	return g.add(method, path, h, opts...)
}

// Get registers a GET route within the group with the given path and handler.
func (g *Group) Get(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodGet, path, h, opts...)
}

// Post registers a POST route within the group with the given path and handler.
func (g *Group) Post(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodPost, path, h, opts...)
}

// Put registers a PUT route within the group with the given path and handler.
func (g *Group) Put(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodPut, path, h, opts...)
}

// Delete registers a DELETE route within the group with the given path and handler.
func (g *Group) Delete(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodDelete, path, h, opts...)
}

// Patch registers a PATCH route within the group with the given path and handler.
func (g *Group) Patch(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodPatch, path, h, opts...)
}

// Options registers an OPTIONS route within the group with the given path and handler.
func (g *Group) Options(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodOptions, path, h, opts...)
}

// Head registers a HEAD route within the group with the given path and handler.
func (g *Group) Head(path string, h HandlerFunc, opts ...RouteOption) *Route {
	return g.handle(methodHead, path, h, opts...)
}

// Group creates a nested subgroup with an additional path segment and optional middlewares.
// The new group inherits all middlewares from its parent group.
func (g *Group) Group(path string, middlewares ...Middleware) *Group {
	return newGroup(
		// Combine paths
		joinPaths(g.Prefix, path),
		g.disabled,
		// Share the same Okapi instance
		g.okapi,
		// Combine middlewares
		append(g.middlewares, middlewares...)...)
}

// HandleStd registers a standard http.HandlerFunc and wraps it with the group's middleware chain.
func (g *Group) HandleStd(method, path string, h func(http.ResponseWriter, *http.Request), opts ...RouteOption) {
	// Convert standard handler to HandlerFunc
	converted := func(c *Context) error {
		h(c.response, c.request)
		return nil
	}
	// Apply group middleware
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		converted = g.middlewares[i](converted)
	}
	tags := g.Tags
	if len(tags) == 0 {
		tags = []string{g.Prefix}
	}
	// Register route
	g.okapi.addRoute(method, joinPaths(g.Prefix, path), tags, converted, opts...).setDisabled(g.disabled)
}

// HandleHTTP registers a standard http.Handler and wraps it with the group's middleware chain.
func (g *Group) HandleHTTP(method, path string, h http.Handler, opts ...RouteOption) {
	// Convert standard handler to HandlerFunc
	converted := g.okapi.wrapHTTPHandler(h)
	// Apply group middleware
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		converted = g.middlewares[i](converted)
	}
	tags := g.Tags
	if len(tags) == 0 {
		tags = []string{g.Prefix}
	}
	// Register route
	g.okapi.addRoute(method, joinPaths(g.Prefix, path), tags, converted, opts...).setDisabled(g.disabled)
}

// UseMiddleware registers a standard HTTP middleware function and integrates
// it into Okapi's middleware chain.
//
// This enables compatibility with existing middleware libraries that use the
// func(http.Handler) http.Handler pattern.
func (g *Group) UseMiddleware(mw func(http.Handler) http.Handler) {
	g.Use(func(next HandlerFunc) HandlerFunc {
		// Convert HandlerFunc to http.Handler
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := NewContext(g.okapi, w, r)
			if err := next(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		// Apply standard middleware
		wrapped := mw(h)

		// Convert back to HandlerFunc
		return func(ctx *Context) error {
			wrapped.ServeHTTP(ctx.response, ctx.request)
			return nil
		}
	})
}

// Register registers a slice of RouteDefinition with the group.
// It ensures that each route is associated with the group and its Okapi instance.
// If a route's Group field is nil, it assigns the current group to it.
// If the route's Group's Okapi reference is nil, it assigns the group's Okapi instance to it.
// This method is useful for bulk registering routes defined in a controller or similar structure.
//
// Example:
//
//	routes := []okapi.RouteDefinition{
//	    {
//	        Method:  "GET",
//	        Path:    "/example",
//	        Handler: exampleHandler,
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
//
// api:= app.Group("/api")
//
// api.Register(routes...)
func (g *Group) Register(routes ...RouteDefinition) {
	for _, r := range routes {
		if r.Group == nil {
			r.Group = g
		} else if r.Group.okapi == nil {
			r.Group.okapi = g.okapi
		}
		tags := r.Group.Tags
		if len(tags) == 0 {
			tags = []string{g.Prefix}
		}
		for _, mid := range r.Middlewares {
			r.Options = append(r.Options, UseMiddleware(mid))
		}
		g.okapi.addRoute(r.Method, joinPaths(g.Prefix, r.Path), tags, r.Handler, r.Options...).setDisabled(g.disabled)
	}
}
