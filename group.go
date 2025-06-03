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

type Group struct {
	basePath    string
	middlewares []Middleware
	okapi       *Okapi
}

// newGroup creates a new route group with the specified base path, Okapi reference,
// and optional middlewares.
func newGroup(basePath string, okapi *Okapi, middlewares ...Middleware) *Group {
	mws := append([]Middleware{}, middlewares...)
	return &Group{
		basePath:    basePath,
		middlewares: mws,
		okapi:       okapi,
	}
}

// BasePath returns the group's base path that prefixes all routes in this group.
func (g *Group) BasePath() string {
	return g.basePath
}

// SetBasePath updates the group's base path.
// Note: Use with caution as it affects all routes in the group.
func (g *Group) SetBasePath(basePath string) {
	g.basePath = basePath
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
	g.middlewares = append(g.middlewares, m...)
}

// add is an internal method that handles route registration with the combined
// middlewares from both the group and parent Okapi instance.
func (g *Group) add(method, path string, h HandleFunc, opts ...RouteOption) *Route {
	fullPath := joinPaths(g.basePath, path)
	// Wrap handler with combined middlewares
	finalHandler := h
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		finalHandler = g.middlewares[i](finalHandler)
	}
	// Register the route with the joined base path and route path
	return g.okapi.addRoute(method, fullPath, g.basePath, finalHandler, opts...)
}

// handle is a helper method that delegates to add with the given HTTP method.
func (g *Group) handle(method, path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.add(method, path, h, opts...)
}

// Get registers a GET route within the group with the given path and handler.
func (g *Group) Get(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(GET, path, h, opts...)
}

// Post registers a POST route within the group with the given path and handler.
func (g *Group) Post(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(POST, path, h, opts...)
}

// Put registers a PUT route within the group with the given path and handler.
func (g *Group) Put(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(PUT, path, h, opts...)
}

// Delete registers a DELETE route within the group with the given path and handler.
func (g *Group) Delete(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(DELETE, path, h, opts...)
}

// Patch registers a PATCH route within the group with the given path and handler.
func (g *Group) Patch(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(PATCH, path, h, opts...)
}

// Options registers an OPTIONS route within the group with the given path and handler.
func (g *Group) Options(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(OPTIONS, path, h, opts...)
}

// Head registers a HEAD route within the group with the given path and handler.
func (g *Group) Head(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(HEAD, path, h, opts...)
}

// Trace registers a TRACE route within the group with the given path and handler.
func (g *Group) Trace(path string, h HandleFunc, opts ...RouteOption) *Route {
	return g.handle(TRACE, path, h, opts...)
}

// Group creates a nested subgroup with an additional path segment and optional middlewares.
// The new group inherits all middlewares from its parent group.
func (g *Group) Group(path string, middlewares ...Middleware) *Group {
	return newGroup(
		joinPaths(g.basePath, path),              // Combine paths
		g.okapi,                                  // Share the same Okapi instance
		append(g.middlewares, middlewares...)...) // Combine middlewares
}
