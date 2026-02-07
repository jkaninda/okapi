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
	"fmt"
	"strings"
)

type RouteDefinition struct {
	// Method is the HTTP method for the route (e.g., GET, POST, PUT, DELETE, etc.)
	Method string
	// Path is the URL path for the route, relative to the base path of the Okapi instance or group
	Path string
	// Handler is the function that will handle requests to this route
	Handler HandlerFunc
	// Group attach Route to a Group // Optional
	Group *Group
	// OperationId is an optional unique identifier for the route, primarily
	// used in OpenAPI documentation to distinguish operations.
	OperationId string
	// Summary is an optional short description of the route,
	// used for OpenAPI documentation.
	// Example: "Create a new book"
	Summary string
	// Description is an optional detailed description of the route,
	// used for OpenAPI documentation.
	// Example:  "This endpoint allows clients to create a new book in the system by providing the necessary details."
	Description string

	// Request optionally defines the expected input schema for the route.
	// It can be a struct or pointer to a struct with binding tags (query, path, header, cookie, form, body).
	// If provided, Okapi will:
	//   - Bind incoming request data to the struct
	//   - Perform validations based on struct tags (e.g., required, minLength, maxLength, default)
	//   - Generate OpenAPI documentation for the request schema
	//
	// Note: To generate OpenAPI documentation, it is recommended to use a struct or pointer to a struct.
	//
	// Example:
	//	type CreateBookInput struct {
	// 		 Tags []string `query:"tags"`
	//		 XApiKey string `header:"X-API-KEY" required:"true" description:"API Key"`
	//		 Body struct {
	// 		 	Title string `json:"title" required:"true" minLength:"5"  maxLength:"100" description:"Book title"`
	// 		 	Price int    `json:"price" max:"5" min:"2"  yaml:"price" required:"true" description:"Book price"`
	// 		}
	//		}
	//   RouteDefinition{
	//       Method:  "POST",
	//       Path:    "/books",
	//       Request: &CreateBookInput{},
	//   }
	Request any

	// Response optionally defines the output schema for the route.
	// It can be any type (struct, slice, map, etc.). If provided, Okapi will:
	//   - Serialize the value into the response body (e.g., JSON)
	//   - Generate OpenAPI documentation for the response schema
	//
	// Note: To generate OpenAPI documentation, it is recommended to use a struct or pointer to a struct.
	//
	// Example:
	//   type BookResponse struct {
	//       ID    string `json:"id"`
	//       Title string `json:"title"`
	//   }
	//   RouteDefinition{
	//       Method:   "POST",
	//       Path:     "/books",
	//       Request:  &CreateBookInput{},
	//       Response: &BookResponse{},
	//   }
	Response any
	// Security defines the security requirements for the route, such as authentication schemes // Optional
	// It can be also applied at Group level.
	Security []map[string][]string
	// RouteOption registers one or more OpenAPI Doc and middleware functions to the Route. // Optional
	Options []RouteOption
	// Middleware registers one or more middleware functions to the Route. // Optional
	Middlewares []Middleware
}

// RegisterRoutes registers a slice of RouteDefinition with the given Okapi instance.
//
// For each route definition, this function determines whether to register the route
// on the root Okapi instance or within a specific route group (if provided).
//
// It supports all standard HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
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
//	routes := []okapi.RouteDefinition{
//		{
//			Method:  "GET",
//			Path:    "/example",
//			Handler: exampleHandler,
//			OperationId: "get-example",
//			Summary: "Example GET request",
//			Request: nil,
//			Response: &ExampleResponse{},
//			Group:   &okapi.Group{Prefix: "/api/v1", Tags: []string{"Example"}},
//		},
//		{
//			Method:  "POST",
//			Path:    "/example",
//			Handler: exampleHandler,
//			Middlewares: []okapi.Middleware{customMiddleware}
//			Options: []okapi.RouteOption{
//			okapi.DocSummary("Example POST request"),
//			okapi.Request(&ExampleRequest{}),
//			okapi.Response(&ExampleResponse{}),
//		},
//		Security: Security: []map[string][]string{
//			{
//			"bearerAuth": {},
//			},
//		},
//		},
//	}
//
//	// Create a new Okapi instance
//	app := okapi.New()
//	okapi.RegisterRoutes(app, routes)
func RegisterRoutes(o *Okapi, routes []RouteDefinition) {
	for _, r := range routes {
		if r.Path == "" && r.Group == nil {
			panic("okapi: invalid route definition — either Path or Group must be specified")
		}
		if r.Method == "" {
			panic(fmt.Sprintf("okapi: invalid route definition — missing HTTP method for path=%q", r.Path))
		}
		if r.Handler == nil {
			panic(fmt.Sprintf("okapi: invalid route definition — missing handler for method=%q path=%q", r.Method, r.Path))
		}
		group := r.Group
		for _, mid := range r.Middlewares {
			r.Options = append(r.Options, UseMiddleware(mid))
		}
		r.attachDocOptions()
		if group == nil {
			// Create on root Okapi instance
			switch strings.ToUpper(r.Method) {
			case methodGet:
				o.Get(r.Path, r.Handler, r.Options...)
			case methodPost:
				o.Post(r.Path, r.Handler, r.Options...)
			case methodPut:
				o.Put(r.Path, r.Handler, r.Options...)
			case methodDelete:
				o.Delete(r.Path, r.Handler, r.Options...)
			case methodPatch:
				o.Patch(r.Path, r.Handler, r.Options...)
			case methodHead:
				o.Head(r.Path, r.Handler, r.Options...)
			case methodOptions:
				o.Options(r.Path, r.Handler, r.Options...)
			default:
				panic(fmt.Sprintf("okapi: unsupported HTTP method %q for path=%q", r.Method, r.Path))
			}
			continue
		}
		if group.okapi == nil {
			group.okapi = o
		}
		switch strings.ToUpper(r.Method) {
		case methodGet:
			group.Get(r.Path, r.Handler, r.Options...)
		case methodPost:
			group.Post(r.Path, r.Handler, r.Options...)
		case methodPut:
			group.Put(r.Path, r.Handler, r.Options...)
		case methodDelete:
			group.Delete(r.Path, r.Handler, r.Options...)
		case methodPatch:
			group.Patch(r.Path, r.Handler, r.Options...)
		case methodHead:
			group.Head(r.Path, r.Handler, r.Options...)
		case methodOptions:
			group.Options(r.Path, r.Handler, r.Options...)
		default:
			panic(fmt.Sprintf("okapi: unsupported HTTP method %q for path=%q", r.Method, r.Path))
		}
	}
}

// attachDocOptions appends documentation-related RouteOptions to the RouteDefinition
func (r *RouteDefinition) attachDocOptions() {
	if len(r.Security) > 0 {
		r.Options = append(r.Options, withSecurity(r.Security))
	}
	if r.OperationId != "" {
		r.Options = append(r.Options, OperationId(r.OperationId))
	}
	if r.Request != nil {
		r.Options = append(r.Options, Request(r.Request))
	}
	if r.Response != nil {
		r.Options = append(r.Options, Response(r.Response))
	}
	if r.Summary != "" {
		r.Options = append(r.Options, Summary(r.Summary))
	}
	if r.Description != "" {
		r.Options = append(r.Options, Description(r.Description))
	}
}
