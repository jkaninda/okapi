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

// newGroup creates a new route group with the specified base path and Okapi reference.
func newGroup(basePath string, okapi *Okapi, middlewares ...Middleware) *Group {
	mws := append([]Middleware{LoggerMiddleware}, middlewares...)
	return &Group{
		basePath:    basePath,
		middlewares: mws,
		okapi:       okapi,
	}
}

func (g *Group) BasePath() string {
	return g.basePath
}

func (g *Group) SetBasePath(basePath string) {
	g.basePath = basePath
}

func (g *Group) Okapi() *Okapi {
	return g.okapi
}

func (g *Group) Use(m Middleware) {
	g.middlewares = append(g.middlewares, m)
}

func (g *Group) add(method, path string, h HandleFunc) *Route {
	tempOkapi := &Okapi{
		context:     g.okapi.context,
		router:      g.okapi.router,
		middlewares: append(g.okapi.middlewares, g.middlewares...),
		Server:      g.okapi.Server,
		TLSServer:   g.okapi.TLSServer,
		debug:       g.okapi.debug,
		logger:      g.okapi.logger,
	}
	return tempOkapi.addRoute(method, joinPaths(g.basePath, path), h)
}

func (g *Group) handle(method, path string, h HandleFunc) *Route {
	return g.add(method, path, h)
}
func (g *Group) Get(path string, h HandleFunc) *Route    { return g.handle(GET, path, h) }
func (g *Group) Post(path string, h HandleFunc) *Route   { return g.handle(POST, path, h) }
func (g *Group) Put(path string, h HandleFunc) *Route    { return g.handle(PUT, path, h) }
func (g *Group) Delete(path string, h HandleFunc) *Route { return g.handle(DELETE, path, h) }
func (g *Group) Patch(path string, h HandleFunc) *Route  { return g.handle(PATCH, path, h) }
func (g *Group) Options(path string, h HandleFunc) *Route {
	return g.handle(OPTIONS, path, h)
}
func (g *Group) Head(path string, h HandleFunc) *Route  { return g.handle(HEAD, path, h) }
func (g *Group) Trace(path string, h HandleFunc) *Route { return g.handle(TRACE, path, h) }

// Group creates a nested subgroup with an additional path segment
func (g *Group) Group(path string, middlewares ...Middleware) *Group {
	return newGroup(joinPaths(g.basePath, path), g.okapi, append(g.middlewares, middlewares...)...)
}
