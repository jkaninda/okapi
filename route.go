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
	"strings"
)

type RouteDefinition struct {
	// Method is the HTTP method for the route (e.g., GET, POST, PUT, DELETE, etc.)
	Method string
	// Path is the URL path for the route, relative to the base path of the Okapi instance or group
	Path string
	// Handler is the function that will handle requests to this route
	Handler HandleFunc
	// RouteOption registers one or more OpenAPI Doc and middleware functions to the Route. // Optional
	Options []RouteOption
	// Middleware registers one or more middleware functions to the Route. // Optional
	Middlewares []Middleware
	// Security defines the security requirements for the route, such as authentication schemes // Optional
	// It can be also applied at Group level.
	Security []map[string][]string
	// Group attach Route to a Group // Optional
	Group *Group
}

// RegisterRoutes registers a slice of RouteDefinition with the given Okapi instance.
//
// For each route definition, this function determines whether to register the route
// on the root Okapi instance or within a specific route group (if provided).
//
// It supports all standard HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT)
// and applies any associated RouteOptions, such as documentation annotations or middleware settings.
//
// If the Group field in the RouteDefinition is nil, the route is registered on the root Okapi instance.
// Otherwise, it is registered within the specified group. If the group's Okapi reference is unset,
// it is automatically assigned from the root instance.
//
// The function panics if an unsupported HTTP method is encountered.
//
// Note: Use the `Use()` method on either the Okapi instance or a Group to apply middleware.
//
// Example:
//
//	routes := []RouteDefinition{
//	    {
//	        Method:  "GET",
//	        Path:    "/example",
//	        Handler: exampleHandler,
//	        Options: []okapi.RouteOption{
//	            okapi.DocSummary("Example GET request"),
//	        },
//	        Group:   &okapi.Group{Prefix: "/api/v1", Tags: []string{"Example"}},
//	    },
//	    {
//	        Method:  "POST",
//	        Path:    "/example",
//	        Handler: exampleHandler,
//			Middlewares: []okapi.Middleware{customMiddleware}
//	        Options: []okapi.RouteOption{
//	            okapi.DocSummary("Example POST request"),
//	        },
//	    	Security: Security: []map[string][]string{
//				{
//					"bearerAuth": {},
//				},
//			},
//	    },
//	}
//	// Create a new Okapi instance
//	app := okapi.New()
//	okapi.RegisterRoutes(app, routes)
func RegisterRoutes(o *Okapi, routes []RouteDefinition) {
	for _, r := range routes {
		group := r.Group
		for _, mid := range r.Middlewares {
			r.Options = append(r.Options, UseMiddleware(mid))
		}
		if len(r.Security) > 0 {
			r.Options = append(r.Options, withSecurity(r.Security))
		}
		if group == nil {
			// Create on root Okapi instance
			switch strings.ToUpper(r.Method) {
			case GET:
				o.Get(r.Path, r.Handler, r.Options...)
			case POST:
				o.Post(r.Path, r.Handler, r.Options...)
			case PUT:
				o.Put(r.Path, r.Handler, r.Options...)
			case DELETE:
				o.Delete(r.Path, r.Handler, r.Options...)
			case PATCH:
				o.Patch(r.Path, r.Handler, r.Options...)
			case HEAD:
				o.Head(r.Path, r.Handler, r.Options...)
			case OPTIONS:
				o.Options(r.Path, r.Handler, r.Options...)
			case TRACE:
				o.Trace(r.Path, r.Handler, r.Options...)
			case CONNECT:
				o.Connect(r.Path, r.Handler, r.Options...)
			default:
				panic("unsupported method: " + r.Method)
			}
			continue
		}
		if group.okapi == nil {
			group.okapi = o
		}
		switch strings.ToUpper(r.Method) {
		case GET:
			group.Get(r.Path, r.Handler, r.Options...)
		case POST:
			group.Post(r.Path, r.Handler, r.Options...)
		case PUT:
			group.Put(r.Path, r.Handler, r.Options...)
		case DELETE:
			group.Delete(r.Path, r.Handler, r.Options...)
		case PATCH:
			group.Patch(r.Path, r.Handler, r.Options...)
		case HEAD:
			group.Head(r.Path, r.Handler, r.Options...)
		case OPTIONS:
			group.Options(r.Path, r.Handler, r.Options...)
		case TRACE:
			group.Trace(r.Path, r.Handler, r.Options...)
		case CONNECT:
			group.Connect(r.Path, r.Handler, r.Options...)
		default:
			panic("unsupported method: " + r.Method)
		}
	}
}
