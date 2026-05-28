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
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
	goutils "github.com/jkaninda/go-utils"
)

const (
	constInt      = "int"
	constUint     = "uint"
	constInt64    = "int64"
	constInt32    = "int32"
	constFloat    = "float"
	constFloat64  = "float64"
	constDouble   = "double"
	constDateTime = "date-time"
	constDate     = "date"
	constUUID     = "uuid"
	constBool     = "bool"
	constString   = "string"
	constEnum     = "enum"
)

// RouteOption defines a function type that modifies a Route's documentation properties
type RouteOption func(*Route)

// OpenAPI contains configuration for generating OpenAPI/Swagger documentation.
// It includes metadata about the API and its documentation.
type OpenAPI struct {
	Title          string // Title of the API
	Summary        string // OpenAPI >=3.1
	Description    string
	TermsOfService string
	Version        string  // Version of the API
	Servers        Servers // List of server URLs where the API is hosted
	License        License // License information for the API
	Contact        Contact // Contact information for the API maintainers
	// SecuritySchemes defines security schemes for the OpenAPI specification.
	SecuritySchemes  SecuritySchemes
	ExternalDocs     *ExternalDocs
	ComponentSchemas map[string]*SchemaInfo

	// Okapi: UI selects the interactive documentation UI rendered at /docs.
	// Valid values: SwaggerUI (default), RedocUI, ScalarUI.
	// By default each UI is also reachable at its own route (/swagger, /redoc
	// and /scalar) regardless of this setting; set StrictDocUI to serve only /docs.
	UI DocUI
	// Okapi: StrictDocUI restricts documentation serving to /docs only. When false
	// (default), /swagger, /redoc and /scalar are all registered alongside /docs
	// so each UI stays reachable directly. When true, only /docs is served and
	// every per-UI route returns 404.
	StrictDocUI bool
	// Favicon is the URL of the favicon used by the documentation UIs.
	Favicon string
}
type SecuritySchemes []SecurityScheme

type SecurityScheme struct {
	Extensions map[string]any `json:"-" yaml:"-"`
	Origin     *Origin        `json:"__origin__,omitempty" yaml:"__origin__,omitempty"`
	Name       string
	// Type string // "http", "oauth2", "apiKey"
	Type string
	// Scheme string // "basic", "bearer", etc.
	Scheme       string
	BearerFormat string
	// In string // "header", "query", "cookie"
	In          string
	Flows       *OAuthFlows
	Description string
}
type ExternalDocs struct {
	Extensions map[string]any `json:"-" yaml:"-"`
	Origin     *Origin        `json:"__origin__,omitempty" yaml:"__origin__,omitempty"`

	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	URL         string `json:"url,omitempty" yaml:"url,omitempty"`
}

type Origin struct {
	Key    *Location           `json:"key,omitempty" yaml:"key,omitempty"`
	Fields map[string]Location `json:"fields,omitempty" yaml:"fields,omitempty"`
}

type Location struct {
	Line   int `json:"line,omitempty" yaml:"line,omitempty"`
	Column int `json:"column,omitempty" yaml:"column,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string
	TokenURL         string
	RefreshURL       string
	Scopes           map[string]string
}
type OAuthFlows struct {
	Implicit          *OAuthFlow
	Password          *OAuthFlow
	ClientCredentials *OAuthFlow
	AuthorizationCode *OAuthFlow
}
type SecurityRequirement map[string][]string // SchemeName -> Scopes

// License contains license information for the API.
// It follows the OpenAPI specification format.
type License struct {
	Extensions map[string]any `json:"-" yaml:"-"`                         // Custom extensions not part of OpenAPI spec
	Name       string         `json:"name" yaml:"name"`                   // Required license name (e.g., "MIT")
	URL        string         `json:"url,omitempty" yaml:"url,omitempty"` // Optional URL to the license
	// Identifier is an SPDX license expression (e.g. "MIT", "Apache-2.0").
	// It is an OpenAPI 3.1-only field and is mutually exclusive with URL: when
	// set, it is emitted only on the default 3.1 document (/openapi.json) and URL
	// is dropped there. It never appears on the 3.0 document (/openapi-3.0.json).
	Identifier string `json:"identifier,omitempty" yaml:"identifier,omitempty"`
}

// Servers is a list of Server objects representing API server locations
type Servers []Server

// Server represents an API server location where the API is hosted
type Server struct {
	Extensions map[string]any `json:"-" yaml:"-"`
	// Server URL (e.g., "https://api.example.com/v1")
	URL string `json:"url" yaml:"url"`
	// Optional server description
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Contact contains contact information for the API maintainers
type Contact struct {
	Extensions map[string]any `json:"-" yaml:"-"`                             // Custom extensions not part of OpenAPI spec
	Name       string         `json:"name,omitempty" yaml:"name,omitempty"`   // Optional contact name
	URL        string         `json:"url,omitempty" yaml:"url,omitempty"`     // Optional contact URL
	Email      string         `json:"email,omitempty" yaml:"email,omitempty"` // Optional contact email
}

// fieldInfo holds information about a struct field
type fieldInfo struct {
	field       reflect.StructField
	required    bool
	description string
}

// ToOpenAPI converts License to openapi3.License.
// It transforms the custom License type to the format expected by the openapi3 package.
func (l License) ToOpenAPI() *openapi3.License {
	license := &openapi3.License{
		Name: l.Name,
		URL:  l.URL,
	}
	// Copy any extensions to the target license object
	for k, v := range l.Extensions {
		license.Extensions[k] = v
	}
	return license
}

// ToOpenAPI converts Servers to openapi3.Servers.
// It transforms the custom Servers type to the format expected by the openapi3 package.
func (s Servers) ToOpenAPI() openapi3.Servers {
	servers := make(openapi3.Servers, 0, len(s))
	for _, srv := range s {
		server := &openapi3.Server{
			URL:         srv.URL,
			Description: srv.Description,
		}
		if len(srv.Extensions) > 0 {
			for k, v := range srv.Extensions {
				server.Extensions[k] = v
			}
		}
		servers = append(servers, server)
	}
	return servers
}

func (ss SecuritySchemes) ToOpenAPI() openapi3.SecuritySchemes {
	result := make(openapi3.SecuritySchemes)
	for _, s := range ss {
		result[s.Name] = &openapi3.SecuritySchemeRef{
			Value: &openapi3.SecurityScheme{
				Extensions:   s.Extensions,
				Origin:       s.Origin.ToOpenAPI(),
				Type:         s.Type,
				Name:         s.Name,
				Scheme:       s.Scheme,
				BearerFormat: s.BearerFormat,
				Flows:        s.Flows.ToOpenAPI(),
				In:           s.In,
				Description:  s.Description,
			},
		}
	}
	return result
}
func (l *Location) ToOpenAPI() openapi3.Location {
	if l == nil {
		return openapi3.Location{}
	}
	return openapi3.Location{
		Line:   l.Line,
		Column: l.Column,
	}

}
func (o *Origin) ToOpenAPI() *openapi3.Origin {
	if o == nil {
		return nil
	}
	origin := &openapi3.Origin{}
	if o.Key != nil {
		origin.Key = &openapi3.Location{
			Line:   o.Key.Line,
			Column: o.Key.Column,
		}
	}
	if len(o.Fields) > 0 {
		origin.Fields = make(map[string]openapi3.Location)
		for k, v := range o.Fields {
			origin.Fields[k] = v.ToOpenAPI()
		}
	}
	return origin
}

func (e *ExternalDocs) ToOpenAPI() *openapi3.ExternalDocs {
	if e == nil {
		return nil
	}
	doc := &openapi3.ExternalDocs{
		Description: e.Description,
		URL:         e.URL,
	}
	for k, v := range e.Extensions {
		doc.Extensions[k] = v
	}
	return doc
}

func (f *OAuthFlow) ToOpenAPI() *openapi3.OAuthFlow {
	if f == nil {
		return nil
	}
	return &openapi3.OAuthFlow{
		AuthorizationURL: f.AuthorizationURL,
		TokenURL:         f.TokenURL,
		RefreshURL:       f.RefreshURL,
		Scopes:           f.Scopes,
	}
}
func (flows *OAuthFlows) ToOpenAPI() *openapi3.OAuthFlows {
	if flows == nil {
		return nil
	}
	return &openapi3.OAuthFlows{
		Implicit:          flows.Implicit.ToOpenAPI(),
		Password:          flows.Password.ToOpenAPI(),
		ClientCredentials: flows.ClientCredentials.ToOpenAPI(),
		AuthorizationCode: flows.AuthorizationCode.ToOpenAPI(),
	}
}

// ToOpenAPI converts Contact to openapi3.Contact.
// It transforms the custom Contact type to the format expected by the openapi3 package.
func (c Contact) ToOpenAPI() *openapi3.Contact {
	contact := &openapi3.Contact{
		Name:  c.Name,
		URL:   c.URL,
		Email: c.Email,
	}
	for k, v := range c.Extensions {
		contact.Extensions[k] = v
	}
	return contact
}

// SchemaInfo holds additional information about a schema for better naming.
// It's used when generating OpenAPI schemas from Go types.
type SchemaInfo struct {
	Schema   *openapi3.SchemaRef
	TypeName string
	Package  string
}

// Doc creates and returns a new DocBuilder instance for chaining documentation options.
func Doc() *DocBuilder {
	return &DocBuilder{}
}

// DocBuilder helps construct a list of RouteOption functions in a fluent, chainable way.
type DocBuilder struct {
	options []RouteOption
}

// RequestBody adds a request body schema to the route documentation using the provided value.
func (b *DocBuilder) RequestBody(v any) *DocBuilder {
	b.options = append(b.options, DocRequestBody(v))
	return b
}

// Response registers a response schema for the route's OpenAPI documentation.
// It can be used in two ways:
//  1. DocResponse(status int, value any) - Defines a response schema for the specified HTTP status code (e.g., 200, 201, 400).
//  2. DocResponse(value any) - Shorthand for DocResponse(200, value).
//
// Examples:
//
//	DocResponse(201, CreatedResponse{})   // Response for 201 Created
//	DocResponse(400, ErrorResponse{})     // Response for 400 Bad Request
//	DocResponse(Response{})               // Response: assumes status 200
func (b *DocBuilder) Response(statusOrValue any, vOptional ...any) *DocBuilder {
	b.options = append(b.options, DocResponse(statusOrValue, vOptional...))
	return b
}

// ErrorResponse defines an error response schema for a specific HTTP status code
// in the route's OpenAPI documentation.
// Deprecated: This function is deprecated in favor of Response(status, v).
//
// Parameters:
//   - status: the HTTP status code (e.g., 400, 404, 500).
//   - v: a Go value (e.g., a struct instance) whose type will be used to generate
//     the OpenAPI schema for the error response.
func (b *DocBuilder) ErrorResponse(status int, v any) *DocBuilder {
	b.options = append(b.options, DocResponse(status, v))
	return b
}

// Summary adds a short summary description to the route documentation.
func (b *DocBuilder) Summary(summary string) *DocBuilder {
	b.options = append(b.options, Summary(summary))
	return b
}

// OperationId sets a unique identifier for the operation in the OpenAPI documentation.
func (b *DocBuilder) OperationId(operationId string) *DocBuilder {
	b.options = append(b.options, OperationId(operationId))
	return b
}

// Description adds a description to the route documentation.
func (b *DocBuilder) Description(description string) *DocBuilder {
	b.options = append(b.options, Description(description))
	return b
}

// Tags adds one or more tags to the route documentation for categorization.
func (b *DocBuilder) Tags(tags ...string) *DocBuilder {
	b.options = append(b.options, Tags(tags...))
	return b
}

// BearerAuth marks the route as requiring Bearer token authentication.
func (b *DocBuilder) BearerAuth() *DocBuilder {
	b.options = append(b.options, DocBearerAuth())
	return b
}

// Deprecated marks the route as deprecated
func (b *DocBuilder) Deprecated() *DocBuilder {
	b.options = append(b.options, Deprecated())
	return b
}

// PathParam adds a documented path parameter to the route.
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
func (b *DocBuilder) PathParam(name, typ, desc string) *DocBuilder {
	b.options = append(b.options, DocPathParam(name, typ, desc))
	return b
}

// PathParamWithDefault adds a documented path parameter to the route.
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
// defvalue: default value to use
func (b *DocBuilder) PathParamWithDefault(name, typ, desc string, defvalue any) *DocBuilder {
	b.options = append(b.options, DocPathParamWithDefault(name, typ, desc, defvalue))
	return b
}

// QueryParam adds a documented query parameter to the route.
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
// required: whether the parameter is required
func (b *DocBuilder) QueryParam(name, typ, desc string, required bool) *DocBuilder {
	b.options = append(b.options, DocQueryParam(name, typ, desc, required))
	return b
}

// QueryParamWithDefault adds a documented query parameter to the route with default.
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
// required: whether the parameter is required
// defvalue: default value to use
func (b *DocBuilder) QueryParamWithDefault(name, typ, desc string, required bool, defvalue any) *DocBuilder {
	b.options = append(b.options, DocQueryParamWithDefault(name, typ, desc, required, defvalue))
	return b
}

// Header adds a documented header to the route.
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description
// required: whether the header is required
func (b *DocBuilder) Header(name, typ, desc string, required bool) *DocBuilder {
	b.options = append(b.options, DocHeader(name, typ, desc, required))
	return b
}

// HeaderWithDefault adds a documented header to the route with default.
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description
// required: whether the header is required
// defvalue: default value to use
func (b *DocBuilder) HeaderWithDefault(name, typ, desc string, required bool, defvalue any) *DocBuilder {
	b.options = append(b.options, DocHeaderWithDefault(name, typ, desc, required, defvalue))
	return b
}

// ResponseHeader adds a response header to the route documentation
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description, optional
func (b *DocBuilder) ResponseHeader(name, typ string, desc ...string) *DocBuilder {
	b.options = append(b.options, DocResponseHeader(name, typ, desc...))
	return b
}

// Hide marks the route to be excluded from OpenAPI documentation.
func (b *DocBuilder) Hide() *DocBuilder {
	b.options = append(b.options, Hide())
	return b
}

// Build returns a single RouteOption composed of all accumulated documentation options.
// This method is intended to be passed directly to route registration functions.
//
// Example:
//
//	okapi.Get("/books", handler, okapi.Doc().response(Book{}).Summary("List books").Build())
func (b *DocBuilder) Build() RouteOption {
	return b.AsOption()
}

// AsOption returns a single RouteOption by merging all accumulated documentation options.
// This is functionally equivalent to Build(), and exists for naming flexibility and readability.
//
// You can use either Build() or AsOption(), depending on what best fits your code style.
//
// Example:
//
//	okapi.Get("/books", handler, okapi.Doc().response(Book{}).AsOption())
func (b *DocBuilder) AsOption() RouteOption {
	return func(r *Route) {
		for _, opt := range b.options {
			opt(r)
		}
	}
}

// ptr is a helper function that returns a pointer to any value
func ptr[T any](v T) *T { return &v }

// DocSummary sets a short summary description for the route
func DocSummary(summary string) RouteOption {
	return Summary(summary)
}

// DocHide marks the route to be excluded from OpenAPI documentation.
func DocHide() RouteOption {
	return Hide()
}
func DocOperationId(operationId string) RouteOption {
	return OperationId(operationId)
}

// DocDescription sets a description for the route
func DocDescription(description string) RouteOption {
	return Description(description)
}

// Hide marks the route to be excluded from OpenAPI documentation.
func Hide() RouteOption {
	return func(r *Route) {
		r.hidden = true
	}
}

// OperationId sets a unique identifier for the operation in the OpenAPI documentation.
func OperationId(operationId string) RouteOption {
	return func(r *Route) {
		r.operationId = operationId
	}
}

// Summary sets a short summary description for the route
func Summary(summary string) RouteOption {
	return func(r *Route) {
		r.summary = summary
	}
}

// Description adds a description to the route documentation.
func Description(description string) RouteOption {
	return func(route *Route) {
		route.description = description
	}
}

// DocPathParam adds a path parameter to the route documentation
// name: parameter name
// typ: parameter type (e.g., "string", "int", "uuid")
// desc: parameter description
func DocPathParam(name, typ, desc string) RouteOption {
	return DocPathParamWithDefault(name, typ, desc, nil)
}

// DocPathParamWithDefault adds a path parameter to the route documentation
// name: parameter name
// typ: parameter type (e.g., "string", "int", "uuid")
// desc: parameter description
// defvalue: default value to use.
func DocPathParamWithDefault(name, typ, desc string, defvalue any) RouteOption {
	return func(r *Route) {

		var schema *openapi3.SchemaRef
		// accept custom schema
		if sch, ok := defvalue.(*openapi3.SchemaRef); ok {
			schema = sch
		} else {
			schema = getSchemaForType(typ)
			if defvalue != nil {
				// special handling for enum default
				dv := reflect.ValueOf(defvalue)
				if strings.ToLower(typ) == constEnum && dv.Kind() == reflect.Slice {
					enumvals := make([]any, dv.Len())
					for i := 0; i < dv.Len(); i++ {
						enumvals[i] = dv.Index(i).Interface()
					}
					schema.Value.Enum = enumvals
				} else {
					schema.Value.Default = defvalue
				}
			}
		}

		r.pathParams = append(r.pathParams, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        name,
				In:          "path",
				Required:    true,
				Schema:      schema,
				Description: desc,
			},
		})
	}
}

// docAutoPathParams automatically extracts path parameters from the route path
// and adds them to the documentation.
// It skips parameters that are already defined.
func docAutoPathParams() RouteOption {
	return func(r *Route) {
		pathParams := extractPathParams(r.docPath)
		for _, param := range pathParams {
			// Check if parameter already exists to avoid duplicates
			exists := false
			for _, existing := range r.pathParams {
				if existing.Value.Name == param.Value.Name {
					exists = true
					break
				}
			}
			if !exists {
				r.pathParams = append(r.pathParams, param)
			}
		}
	}
}

// DocQueryParam adds a query parameter to the route documentation
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
// required: whether the parameter is required
func DocQueryParam(name, typ, desc string, required bool) RouteOption {
	return DocQueryParamWithDefault(name, typ, desc, required, nil)
}

// DocQueryParamWithDefault adds a query parameter to the route documentation (with default if provided)
// name: parameter name
// typ: parameter type (e.g., "string", "int")
// desc: parameter description
// required: whether the parameter is required
// defvalue: default value to use
func DocQueryParamWithDefault(name, typ, desc string, required bool, defvalue any) RouteOption {
	return func(r *Route) {
		var schema *openapi3.SchemaRef
		// accept custom schema
		if sch, ok := defvalue.(*openapi3.SchemaRef); ok {
			schema = sch
		} else {
			schema = getSchemaForType(typ)
			if defvalue != nil {
				// special handling for enum default
				dv := reflect.ValueOf(defvalue)
				if strings.ToLower(typ) == constEnum && dv.Kind() == reflect.Slice {
					enumvals := make([]any, dv.Len())
					for i := 0; i < dv.Len(); i++ {
						enumvals[i] = dv.Index(i).Interface()
					}
					schema.Value.Enum = enumvals
				} else {
					schema.Value.Default = defvalue
				}
			}
		}
		r.queryParams = append(r.queryParams, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        name,
				In:          "query",
				Required:    required,
				Schema:      schema,
				Description: desc,
			},
		})
	}
}

// DocHeader adds a header parameter to the route documentation
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description
// required: whether the header is required
func DocHeader(name, typ, desc string, required bool) RouteOption {
	return DocHeaderWithDefault(name, typ, desc, required, nil)
}

// DocHeaderWithDefault adds a header parameter to the route documentation with default (if provided)
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description
// required: whether the header is required
// defvalue: default value to use
func DocHeaderWithDefault(name, typ, desc string, required bool, defvalue any) RouteOption {
	return func(r *Route) {
		var schema *openapi3.SchemaRef
		// accept custom schema
		if sch, ok := defvalue.(*openapi3.SchemaRef); ok {
			schema = sch
		} else {
			schema = getSchemaForType(typ)
			if defvalue != nil {
				// special handling for enum default
				dv := reflect.ValueOf(defvalue)
				if strings.ToLower(typ) == constEnum && dv.Kind() == reflect.Slice {
					enumvals := make([]any, dv.Len())
					for i := 0; i < dv.Len(); i++ {
						enumvals[i] = dv.Index(i).Interface()
					}
					schema.Value.Enum = enumvals
				} else {
					schema.Value.Default = defvalue
				}
			}
		}
		r.headers = append(r.headers, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        name,
				In:          "header",
				Required:    required,
				Schema:      schema,
				Description: desc,
			},
		})
	}
}

// DocTag adds a single tag to categorize the route
func DocTag(tag string) RouteOption {
	return Tag(tag)
}

// DocTags adds multiple tags to categorize the route
func DocTags(tags ...string) RouteOption {
	return Tags(tags...)
}

// DocResponseHeader adds a response header to the route documentation
// name: header name
// typ: header value type (e.g., "string", "int")
// desc: header description, optional
func DocResponseHeader(name, typ string, desc ...string) RouteOption {
	return func(r *Route) {
		schema := getSchemaForType(typ)
		description := ""
		// Initialize responseHeaders map if it doesn't exist
		if r.responseHeaders == nil {
			r.responseHeaders = make(map[string]*openapi3.HeaderRef)
		}
		if len(desc) != 0 {
			description = desc[0]
		}
		r.responseHeaders[name] = &openapi3.HeaderRef{
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: description,
					Required:    true,
					Schema:      schema,
				},
			},
		}
	}
}

// DocResponse registers a response schema for the route's OpenAPI documentation.
// It can be used in two ways:
//  1. DocResponse(status int, value any) - Defines a response schema for the specified HTTP status code (e.g., 200, 201, 400).
//  2. DocResponse(value any) - Shorthand for DocResponse(200, value).
//
// Examples:
//
//	DocResponse(201, CreatedResponse{})   // response for 201 Created
//	DocResponse(400, ErrorResponse{})     // response for 400 Bad request
//	DocResponse(response{})               // response: assumes status 200
func DocResponse(statusOrValue any, vOptional ...any) RouteOption {
	return func(doc *Route) {
		switch val := statusOrValue.(type) {
		case int:
			// usage: DocResponse(200, value)
			if len(vOptional) == 0 || vOptional[0] == nil {
				return
			}
			doc.responses[val] = reflectToSchemaWithInfo(vOptional[0]).Schema

		default:
			// usage: DocResponse(value)
			if val == nil {
				return
			}
			doc.responses[200] = reflectToSchemaWithInfo(val).Schema
		}
	}
}

// DocErrorResponse defines an error response schema for a specific HTTP status code
// in the route's OpenAPI documentation.
// Deprecated: This function is deprecated in favor of DocResponse(status, v).
//
// Parameters:
//   - status: the HTTP status code (e.g., 400, 404, 500).
//   - v: a Go value (e.g., a struct instance) whose type will be used to generate
//     the OpenAPI schema for the error response.
//
// Returns:
//   - A RouteOption function that adds the error schema to the route's documentation.
func DocErrorResponse(status int, v any) RouteOption {
	return func(doc *Route) {
		if v == nil {
			return
		}
		// Generate a schema from the provided Go value and assign it to the error response
		doc.responses[status] = reflectToSchemaWithInfo(v).Schema
	}
}

// DocRequestBody defines the request body schema for the route
// v: a Go value whose type will be used to generate the request schema
func DocRequestBody(v any) RouteOption {
	return func(doc *Route) {
		if v == nil {
			return
		}
		doc.request = reflectToSchemaWithInfo(v).Schema
	}
}

// Tag adds a single tag to categorize the route
func Tag(tag string) RouteOption {
	return func(r *Route) {
		r.tags = append(r.tags, tag)
	}
}

// Tags adds multiple tags to categorize the route
func Tags(tags ...string) RouteOption {
	return func(doc *Route) {
		doc.tags = append(doc.tags, tags...)
	}
}

// Request registers the request schema for a route.
// The provided value must be a struct or a pointer to a struct.
//
// This schema is used for both OpenAPI documentation and request validation.
//
// Field mapping rules:
//   - Request body: A field named `Body`, or a field tagged with `json:"body"`, is treated as the request body.
//   - Path parameters: Fields tagged with `path:"name"` or `param:"name"` are treated as path parameters.
//   - Query parameters: Fields tagged with `query:"name"` are treated as query parameters.
//   - Headers: Fields tagged with `header:"name"` are treated as HTTP headers.
//   - Cookies: Fields tagged with `cookie:"name"` are treated as HTTP cookies.
//   - Any remaining fields are treated as general request metadata or ignored if not applicable.
func Request(v any) RouteOption {
	return func(r *Route) {
		if v != nil {
			r.generateRequestSchema(v)
		}
	}
}

// Response registers the response schema for a route.
// The provided value must be a struct or a pointer to a struct.
//
// This schema is used for OpenAPI documentation and response representation.
//
// Field mapping rules:
//   - Status code: A field named `Status` is interpreted as the HTTP status code (default: 200 if omitted).
//   - Response body: A field named `Body`, or a field tagged with `json:",inline"`, is treated as the response body.
//   - Headers: Fields tagged with `header:"name"` are treated as HTTP response headers.
//   - Cookies: Fields tagged with `cookie:"name"` are treated as HTTP cookies.
//   - Any remaining fields are treated as general response metadata or ignored if not applicable.
//
// Example:
//
//	type CreateUserResponse struct {
//	    Status int 			`json:"status"`
//	    Body User 			`json:"body"`
//	    Trace string 		`header:"X-Trace-ID"`
//	    SessionId string 	`cookie:"session_id"`
//	}
func Response(v any) RouteOption {
	return func(r *Route) {
		if v != nil {
			r.generateResponseSchema(v)
		}
	}
}

// WithIO registers both request and response schemas for a route in one call.
// It is a convenience helper that combines Request and Response.
func WithIO(req any, res any) RouteOption {
	return func(r *Route) {
		if req != nil {
			r.generateRequestSchema(req)
		}
		if res != nil {
			r.generateResponseSchema(res)
		}
	}
}

// DocBearerAuth marks the route as requiring Bearer token authentication
func DocBearerAuth() RouteOption {
	return func(doc *Route) {
		doc.bearerAuth = true
	}
}

// DocBasicAuth marks the route as requiring Basic authentication
func DocBasicAuth() RouteOption {
	return func(doc *Route) {
		doc.basicAuth = true
	}
}

// DocDeprecated marks the route as deprecated
func DocDeprecated() RouteOption {
	return Deprecated()
}

// Deprecated marks the route as deprecated
func Deprecated() RouteOption {
	return func(doc *Route) {
		doc.deprecated = true
	}
}

func withSecurity(security []map[string][]string) RouteOption {
	return func(r *Route) {
		r.security = security
	}
}

// buildOpenAPISpec constructs the complete OpenAPI specification documents by
// aggregating all route documentation. It first builds the OpenAPI 3.0 base
// spec, then derives the OpenAPI 3.1 spec from it (see deriveSpec31). The 3.1
// document is the default served at /openapi.json; both remain reachable at
// their version-pinned routes.
func (o *Okapi) buildOpenAPISpec() {
	spec := &openapi3.T{
		OpenAPI: openApiVersion,
		Info: &openapi3.Info{
			Title:          o.openAPI.Title,
			Version:        o.openAPI.Version,
			Summary:        o.openAPI.Summary,
			Description:    o.openAPI.Description,
			TermsOfService: o.openAPI.TermsOfService,
			License:        o.openAPI.License.ToOpenAPI(),
			Contact:        o.openAPI.Contact.ToOpenAPI(),
		},
		Paths:   &openapi3.Paths{},
		Servers: o.openAPI.Servers.ToOpenAPI(),
		Components: &openapi3.Components{
			SecuritySchemes: o.openAPI.SecuritySchemes.ToOpenAPI(),
			Schemas:         make(openapi3.Schemas),
		},
		ExternalDocs: o.openAPI.ExternalDocs.ToOpenAPI(),
	}
	if len(o.openAPI.SecuritySchemes) == 0 && o.hasBearerAuth() {
		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{
			"BearerAuth": &openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:         "http",
					Scheme:       "bearer",
					BearerFormat: "JWT",
				},
			},
		}
	}
	if len(o.openAPI.SecuritySchemes) == 0 && o.hasBasicAuth() {
		spec.Components.SecuritySchemes = openapi3.SecuritySchemes{
			"BasicAuth": &openapi3.SecuritySchemeRef{
				Value: &openapi3.SecurityScheme{
					Type:   "http",
					Scheme: "basic",
				},
			},
		}
	}
	// Initialize schema registry for reusable components
	schemaRegistry := make(map[string]*SchemaInfo)

	// Start with registered ones first
	for name, sinfo := range o.openAPI.ComponentSchemas {
		schemaRegistry[name] = sinfo
		spec.Components.Schemas[name] = sinfo.Schema
	}

	// Process all registered routes
	for _, r := range o.routes {
		// If route is disabled ignore it
		if r.disabled || r.hidden {
			continue
		}
		// Auto-extract path parameters if none are defined
		if len(r.pathParams) == 0 {
			docAutoPathParams()(r)
		}
		if len(r.operationId) == 0 {
			if len(r.summary) != 0 {
				r.operationId = goutils.Slug(r.summary)
			}
		}
		item := spec.Paths.Value(r.Path)
		if item == nil {
			item = &openapi3.PathItem{}
			spec.Paths.Set(r.Path, item)
		}

		op := o.buildOperation(spec, r, schemaRegistry)

		// Assign operation to correct HTTP verb
		switch r.Method {
		case methodGet:
			item.Get = op
		case methodPost:
			item.Post = op
		case methodPut:
			item.Put = op
		case methodDelete:
			item.Delete = op
		case methodPatch:
			item.Patch = op
		case methodHead:
			item.Head = op
		case methodOptions:
			item.Options = op
		}
	}

	spec.Tags = o.collectRootTags()

	// Derive the OpenAPI 3.1 document from the 3.0 base before the base is
	// cleaned of internal markers (the derivation deep-copies the base).
	o.openapiSpec31 = o.deriveSpec31(spec)
	// Remove internal markers so the 3.0 document stays clean and valid.
	stripConstMarkers(spec)
	o.openapiSpec = spec
}

// buildOperation builds an OpenAPI operation from a route's documentation
// metadata. It is shared by route generation and webhook generation, and it
// registers reusable object schemas as components on spec via schemaRegistry.
func (o *Okapi) buildOperation(spec *openapi3.T, r *Route, schemaRegistry map[string]*SchemaInfo) *openapi3.Operation {
	op := &openapi3.Operation{
		OperationID: r.operationId,
		Summary:     r.summary,
		Description: r.description,
		Tags:        goutils.RemoveDuplicates(r.tags), // Remove duplicates in tags
		Parameters:  append(append(r.pathParams, r.queryParams...), r.headers...),
		Responses:   &openapi3.Responses{},
		Deprecated:  r.deprecated,
	}

	addSecurity(spec, op, r)
	// Handle request body
	if r.request != nil {
		// Generate reusable schema component if it's a complex type
		schemaRef := o.getOrCreateSchemaComponent(r.request, schemaRegistry, spec.Components.Schemas)

		requestBody := &openapi3.RequestBody{
			Content:  openapi3.NewContentWithJSONSchemaRef(schemaRef),
			Required: true,
		}

		// Add example if available
		if r.requestExample != nil {
			requestBody.Content[constJSON].Example = r.requestExample
		}

		op.RequestBody = &openapi3.RequestBodyRef{Value: requestBody}
	}
	if len(r.responses) != 0 {
		for key, resp := range r.responses {
			schemaRef := o.getOrCreateSchemaComponent(resp, schemaRegistry, spec.Components.Schemas)
			apiResponse := &openapi3.Response{
				Description: ptr(http.StatusText(key)),
				Content:     openapi3.NewContentWithJSONSchemaRef(schemaRef),
				Headers:     r.responseHeaders,
			}
			op.Responses.Set(strconv.Itoa(key), &openapi3.ResponseRef{
				Value: apiResponse,
			})
		}
	}
	// Add default responses
	op.Responses.Set("500", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Description: ptr("Internal Server Error"),
		},
	})
	return op
}

// deriveSpec31 produces an OpenAPI 3.1 document from the 3.0 base spec.
//
// The base is deep-copied via a JSON round-trip so the 3.0 document served at
// /openapi.json is never mutated. kin-openapi marshals whatever fields are set
// regardless of document version, so all 3.1-only adjustments (type-array
// nullability, jsonSchemaDialect, SPDX license identifier, examples, const,
// webhooks) are applied here and only here.
func (o *Okapi) deriveSpec31(base *openapi3.T) *openapi3.T {
	clone := &openapi3.T{}
	data, err := base.MarshalJSON()
	if err != nil {
		o.logger.Error("openapi: failed to marshal base spec for 3.1 derivation", "error", err)
		return clone
	}
	if err := clone.UnmarshalJSON(data); err != nil {
		o.logger.Error("openapi: failed to derive 3.1 spec", "error", err)
		return clone
	}

	clone.OpenAPI = openApiVersion31
	clone.JSONSchemaDialect = jsonSchemaDialect

	// SPDX license identifier (3.1-only). identifier and url are mutually
	// exclusive, so url is dropped when an identifier is provided.
	if id := o.openAPI.License.Identifier; id != "" && clone.Info != nil {
		if clone.Info.License == nil {
			clone.Info.License = &openapi3.License{Name: o.openAPI.License.Name}
		}
		clone.Info.License.Identifier = id
		clone.Info.License.URL = ""
	}

	// Webhooks (3.1-only). Built before the schema transform so webhook schemas
	// are converted to 3.1 idioms as well.
	o.buildWebhooks(clone)

	// Convert every schema in the document to OpenAPI 3.1 / JSON Schema 2020-12.
	transformSpecTo31(clone)
	return clone
}

// buildWebhooks populates spec.Webhooks from the registered webhook routes.
func (o *Okapi) buildWebhooks(spec *openapi3.T) {
	if len(o.webhooks) == 0 {
		return
	}
	if spec.Components == nil {
		spec.Components = &openapi3.Components{Schemas: make(openapi3.Schemas)}
	}
	if spec.Components.Schemas == nil {
		spec.Components.Schemas = make(openapi3.Schemas)
	}
	// Seed the schema registry with components that already exist on the spec so
	// webhook payloads reuse them instead of creating duplicates.
	registry := make(map[string]*SchemaInfo, len(spec.Components.Schemas))
	for name, ref := range spec.Components.Schemas {
		registry[name] = &SchemaInfo{Schema: ref, TypeName: name}
	}

	spec.Webhooks = make(map[string]*openapi3.PathItem, len(o.webhooks))
	for _, r := range o.webhooks {
		item := spec.Webhooks[r.Name]
		if item == nil {
			item = &openapi3.PathItem{}
			spec.Webhooks[r.Name] = item
		}
		op := o.buildOperation(spec, r, registry)
		switch r.Method {
		case methodGet:
			item.Get = op
		case methodPost:
			item.Post = op
		case methodPut:
			item.Put = op
		case methodDelete:
			item.Delete = op
		case methodPatch:
			item.Patch = op
		case methodHead:
			item.Head = op
		case methodOptions:
			item.Options = op
		default:
			item.Post = op
		}
	}
}

// transformSpecTo31 walks every schema reachable from spec and rewrites it to
// OpenAPI 3.1 idioms (see transformSchemaTo31).
func transformSpecTo31(spec *openapi3.T) {
	walkAllSchemas(spec, transformSchemaTo31)
}

// transformSchemaTo31 rewrites a single schema from the version-agnostic base
// form to OpenAPI 3.1 / JSON Schema 2020-12:
//   - nullable: true        -> type becomes ["<type>", "null"]
//   - example               -> examples: [example]
//   - x-okapi-const marker  -> const
func transformSchemaTo31(s *openapi3.Schema) {
	if s == nil {
		return
	}
	// Nullable -> type array
	if s.Nullable {
		if s.Type != nil && len(s.Type.Slice()) > 0 && !s.Type.Includes(openapi3.TypeNull) {
			types := openapi3.Types(append(s.Type.Slice(), openapi3.TypeNull))
			s.Type = &types
		}
		s.Nullable = false
	}
	// example -> examples (3.1 prefers the array form)
	if s.Example != nil && len(s.Examples) == 0 {
		s.Examples = []any{s.Example}
		s.Example = nil
	}
	// const marker -> const keyword
	if s.Extensions != nil {
		if v, ok := s.Extensions[extOkapiConst]; ok {
			s.Const = v
			delete(s.Extensions, extOkapiConst)
		}
	}
	// exclusiveMinimum/Maximum: boolean modifier (3.0) -> numeric bound (3.1).
	if s.ExclusiveMin.IsTrue() && s.Min != nil {
		s.ExclusiveMin = openapi3.ExclusiveBound{Value: s.Min}
		s.Min = nil
	}
	if s.ExclusiveMax.IsTrue() && s.Max != nil {
		s.ExclusiveMax = openapi3.ExclusiveBound{Value: s.Max}
		s.Max = nil
	}
}

// stripConstMarkers removes the internal const marker extension from every
// schema in spec so the 3.0 document never exposes it.
func stripConstMarkers(spec *openapi3.T) {
	walkAllSchemas(spec, func(s *openapi3.Schema) {
		if s.Extensions != nil {
			delete(s.Extensions, extOkapiConst)
		}
	})
}

// walkAllSchemas applies fn exactly once to every schema reachable from the
// document: component schemas, and the parameter/request/response schemas of
// every path and webhook operation.
func walkAllSchemas(spec *openapi3.T, fn func(*openapi3.Schema)) {
	seen := make(map[*openapi3.Schema]bool)
	if spec.Components != nil {
		for _, ref := range spec.Components.Schemas {
			walkSchemaRef(ref, seen, fn)
		}
	}
	if spec.Paths != nil {
		for _, item := range spec.Paths.Map() {
			walkPathItemSchemas(item, seen, fn)
		}
	}
	for _, item := range spec.Webhooks {
		walkPathItemSchemas(item, seen, fn)
	}
}

// walkPathItemSchemas applies fn to the schemas of every operation on item.
func walkPathItemSchemas(item *openapi3.PathItem, seen map[*openapi3.Schema]bool, fn func(*openapi3.Schema)) {
	if item == nil {
		return
	}
	for _, op := range item.Operations() {
		if op == nil {
			continue
		}
		for _, p := range op.Parameters {
			if p != nil && p.Value != nil {
				walkSchemaRef(p.Value.Schema, seen, fn)
			}
		}
		if op.RequestBody != nil && op.RequestBody.Value != nil {
			for _, mt := range op.RequestBody.Value.Content {
				if mt != nil {
					walkSchemaRef(mt.Schema, seen, fn)
				}
			}
		}
		if op.Responses != nil {
			for _, resp := range op.Responses.Map() {
				if resp == nil || resp.Value == nil {
					continue
				}
				for _, mt := range resp.Value.Content {
					if mt != nil {
						walkSchemaRef(mt.Schema, seen, fn)
					}
				}
			}
		}
	}
}

// walkSchemaRef applies fn to a schema and all of its descendants once.
func walkSchemaRef(ref *openapi3.SchemaRef, seen map[*openapi3.Schema]bool, fn func(*openapi3.Schema)) {
	if ref == nil || ref.Value == nil {
		return
	}
	s := ref.Value
	if seen[s] {
		return
	}
	seen[s] = true
	fn(s)
	for _, child := range s.Properties {
		walkSchemaRef(child, seen, fn)
	}
	walkSchemaRef(s.Items, seen, fn)
	if s.AdditionalProperties.Schema != nil {
		walkSchemaRef(s.AdditionalProperties.Schema, seen, fn)
	}
	for _, child := range s.AllOf {
		walkSchemaRef(child, seen, fn)
	}
	for _, child := range s.AnyOf {
		walkSchemaRef(child, seen, fn)
	}
	for _, child := range s.OneOf {
		walkSchemaRef(child, seen, fn)
	}
	walkSchemaRef(s.Not, seen, fn)
}

// collectRootTags aggregates GroupTag entries from every route
func (o *Okapi) collectRootTags() openapi3.Tags {
	seen := make(map[string]*openapi3.Tag)
	for _, r := range o.routes {
		if r.disabled || r.hidden {
			continue
		}
		for _, t := range r.tagInfos {
			if t.Name == "" {
				continue
			}
			if _, ok := seen[t.Name]; ok {
				continue
			}
			seen[t.Name] = &openapi3.Tag{
				Name:         t.Name,
				Description:  t.Description,
				ExternalDocs: t.ExternalDocs.ToOpenAPI(),
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	tags := make(openapi3.Tags, 0, len(names))
	for _, name := range names {
		tags = append(tags, seen[name])
	}
	return tags
}
func (o *Okapi) hasBearerAuth() bool {
	// Check if any route requires Bearer authentication
	for _, r := range o.routes {
		if r.bearerAuth {
			return true
		}
	}
	return false
}
func (o *Okapi) hasBasicAuth() bool {
	// Check if any route requires Basic authentication
	for _, r := range o.routes {
		if r.basicAuth {
			return true
		}
	}
	return false
}

// getOrCreateSchemaComponent creates reusable schema components for complex types
func (o *Okapi) getOrCreateSchemaComponent(schema *openapi3.SchemaRef,
	registry map[string]*SchemaInfo,
	components openapi3.Schemas) *openapi3.SchemaRef {
	if schema == nil || schema.Value == nil {
		return schema
	}

	// Only create components for object schemas (structs)
	if schema.Value.Type == nil || !schema.Value.Type.Is("object") || len(schema.Value.Properties) == 0 {
		return schema
	}

	// Try to find existing schema info in registry by comparing schema structure
	for componentName, schemaInfo := range registry {
		if o.schemasEqual(schema, schemaInfo.Schema) {
			return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", componentName)}
		}
	}

	// Generate a component name based on the schema title or structure
	componentName := o.generateComponentName(schema)

	// Ensure uniqueness
	originalName := componentName
	counter := 1
	for _, exists := registry[componentName]; exists; _, exists = registry[componentName] {
		componentName = fmt.Sprintf("%s%d", originalName, counter)
		counter++
	}

	// Register the schema as a component
	schemaInfo := &SchemaInfo{
		Schema:   schema,
		TypeName: schema.Value.Title,
	}
	registry[componentName] = schemaInfo
	components[componentName] = schema

	// Return a reference to the component
	return &openapi3.SchemaRef{Ref: fmt.Sprintf("#/components/schemas/%s", componentName)}
}

// schemasEqual compares two schemas for structural equality
func (o *Okapi) schemasEqual(a, b *openapi3.SchemaRef) bool {
	if a == nil || b == nil || a.Value == nil || b.Value == nil {
		return a == b
	}

	// Compare basic properties
	if a.Value.Title != b.Value.Title {
		return false
	}

	// Compare type
	if (a.Value.Type == nil) != (b.Value.Type == nil) {
		return false
	}
	if a.Value.Type != nil && b.Value.Type != nil && !a.Value.Type.Is(b.Value.Type.Slice()[0]) {
		return false
	}

	// Compare properties count
	if len(a.Value.Properties) != len(b.Value.Properties) {
		return false
	}

	// Compare required fields
	if len(a.Value.Required) != len(b.Value.Required) {
		return false
	}

	// Simple structural comparison - you might want to make this more sophisticated
	for name := range a.Value.Properties {
		if _, exists := b.Value.Properties[name]; !exists {
			return false
		}
	}

	return true
}

// generateComponentName generates a meaningful name for a schema component
func (o *Okapi) generateComponentName(schema *openapi3.SchemaRef) string {
	if schema == nil || schema.Value == nil {
		return "UnknownSchema"
	}

	if schema.Value.Title != "" {
		return o.sanitizeComponentName(schema.Value.Title)
	}

	// Handle nil Type
	if schema.Value.Type == nil {
		if len(schema.Value.Properties) > 0 {
			return "AnonymousObject"
		}
		return "UnknownSchema"
	}

	// Fallback: create name based on properties
	if len(schema.Value.Properties) > 0 {
		properties := make([]string, 0, len(schema.Value.Properties))
		for propName := range schema.Value.Properties {
			properties = append(properties, propName)
		}

		// Sort for consistency
		sort.Strings(properties)

		// Create a hash-based name if we have many properties
		if len(properties) > 3 {
			hash := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(properties, ","))))
			return fmt.Sprintf("Schema_%s", hash[:8])
		}

		// Use property names for simpler schemas, but make it cleaner
		name := strings.Join(properties, "")
		return o.sanitizeComponentName(name + "Object")
	}

	return "EmptySchema"
}

// sanitizeComponentName ensures the component name follows OpenAPI naming conventions
func (o *Okapi) sanitizeComponentName(name string) string {
	// Remove any non-alphanumeric characters except underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	name = reg.ReplaceAllString(name, "")

	// Ensure it starts with a letter
	if len(name) > 0 && !unicode.IsLetter(rune(name[0])) {
		name = "Schema_" + name
	}

	// Ensure it's not empty
	if name == "" {
		name = "Schema"
	}

	return name
}

// reflectToSchemaWithInfo uses SchemaRef or converts a Go type to an OpenAPI schema with type information
func reflectToSchemaWithInfo(v any) *SchemaInfo {
	// 1. if v is schemaRef or *SchemaRef use it.
	switch sr := v.(type) {
	case *openapi3.SchemaRef:
		return &SchemaInfo{
			Schema: sr,
		}
	case openapi3.SchemaRef:
		return &SchemaInfo{
			Schema: &sr,
		}
	}

	// 2. inspect the struct
	t := reflect.TypeOf(v)

	// Handle pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := typeToSchemaWithInfo(t)

	return &SchemaInfo{
		Schema:   schema,
		TypeName: t.Name(),
		Package:  t.PkgPath(),
	}
}

// typeToSchemaWithInfo converts a reflect.Type to an OpenAPI SchemaRef with proper naming
func typeToSchemaWithInfo(t reflect.Type) *openapi3.SchemaRef {
	switch t.Kind() {
	case reflect.String:
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema := openapi3.NewIntegerSchema()
		if t.Kind() == reflect.Int64 {
			schema.Format = constInt64
		} else {
			schema.Format = constInt32
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema := openapi3.NewIntegerSchema()
		schema.Min = ptr(float64(0))
		if t.Kind() == reflect.Uint64 {
			schema.Format = constInt64
		} else {
			schema.Format = constInt32
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Float32, reflect.Float64:
		schema := openapi3.NewFloat64Schema()
		if t.Kind() == reflect.Float32 {
			schema.Format = constFloat
		} else {
			schema.Format = constDouble
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Bool:
		return openapi3.NewSchemaRef("", openapi3.NewBoolSchema())

	case reflect.Slice, reflect.Array:
		elemSchema := typeToSchemaWithInfo(t.Elem())
		schema := openapi3.NewArraySchema()
		schema.Items = elemSchema
		return openapi3.NewSchemaRef("", schema)

	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			valueSchema := typeToSchemaWithInfo(t.Elem())
			schema := openapi3.NewObjectSchema()
			schema.AdditionalProperties = openapi3.AdditionalProperties{
				Schema: valueSchema,
			}
			return openapi3.NewSchemaRef("", schema)
		}
		return openapi3.NewSchemaRef("", openapi3.NewObjectSchema())

	case reflect.Struct:
		return structToSchemaWithInfo(t)

	case reflect.Interface:
		return openapi3.NewSchemaRef("", &openapi3.Schema{})

	default:
		return openapi3.NewSchemaRef("", openapi3.NewObjectSchema())
	}
}

// structToSchemaWithInfo converts a struct type to an OpenAPI schema with proper naming
func structToSchemaWithInfo(t reflect.Type) *openapi3.SchemaRef {
	// Handle time.Time
	if t == reflect.TypeOf(time.Time{}) {
		schema := openapi3.NewStringSchema()
		schema.Format = constDateTime
		return openapi3.NewSchemaRef("", schema)
	}

	// Dereference pointers
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	schema := openapi3.NewObjectSchema()
	if t.Name() != "" {
		schema.Title = t.Name()
	}
	required := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded (anonymous) fields
		if field.Anonymous {
			embeddedType := field.Type

			// Deref pointer embedded
			for embeddedType.Kind() == reflect.Pointer {
				embeddedType = embeddedType.Elem()
			}

			if embeddedType.Kind() == reflect.Struct {
				embeddedRef := structToSchemaWithInfo(embeddedType)
				if embedded := embeddedRef.Value; embedded != nil && embedded.Properties != nil {
					// Copy properties
					for propName, propSchema := range embedded.Properties {
						// Prevent overwriting parent fields
						if _, exists := schema.Properties[propName]; !exists {
							schema.WithProperty(propName, propSchema.Value)
						}
					}
					// Merge required
					if len(embedded.Required) > 0 {
						required = append(required, embedded.Required...)
					}
				}
			}
			continue
		}

		// Normal named field
		jsonName := getJSONFieldName(field)
		if jsonName == "-" {
			continue
		}
		if hidden := field.Tag.Get(tagHidden); hidden == constTRUE {
			continue
		}

		isPointer := field.Type.Kind() == reflect.Ptr
		fieldType := field.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Create schema for the field type
		fieldSchema := typeToSchemaWithInfo(fieldType)

		// Pointer fields are nullable. Recorded here as the version-agnostic
		// `nullable` flag (valid in 3.0); converted to a `["...","null"]` type
		// array when the 3.1 document is derived.
		if isPointer && fieldSchema.Value != nil {
			fieldSchema.Value.Nullable = true
		}

		// Apply validation tags from the field
		applyValidationTags(fieldSchema.Value, field.Tag)

		// Description
		if desc := field.Tag.Get(tagDescription); desc != "" {
			fieldSchema.Value.Description = desc
		}
		if desc := field.Tag.Get(tagDoc); desc != "" {
			fieldSchema.Value.Description = desc
		}

		// Deprecated
		if deprecated := field.Tag.Get(tagDeprecated); deprecated == constTRUE {
			fieldSchema.Value.Deprecated = true
		}

		schema.WithProperty(jsonName, fieldSchema.Value)

		// Required, check both the required tag and standard logic
		if isRequiredFieldWithTag(field) {
			required = append(required, jsonName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return openapi3.NewSchemaRef("", schema)
}

// getJSONFieldName extracts the JSON field name from struct tags
func getJSONFieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name
	}

	parts := strings.Split(jsonTag, ",")
	name := parts[0]

	if name == "" {
		return field.Name
	}

	return name
}

// isRequiredFieldWithTag determines if a struct field is required
func isRequiredFieldWithTag(field reflect.StructField) bool {
	// First check explicit "required" tag
	if requiredTag := field.Tag.Get("required"); requiredTag == constTRUE {
		return true
	}

	jsonTag := field.Tag.Get("json")
	validateTag := field.Tag.Get("validate")

	// Check if omitempty is present in json tag
	if strings.Contains(jsonTag, "omitempty") {
		return false
	}

	// Check if required is present in validate tag
	if strings.Contains(validateTag, "required") {
		return true
	}

	return false
}

// applyValidationTags applies struct tag validations to a schema
func applyValidationTags(schema *openapi3.Schema, tag reflect.StructTag) {
	// Description
	if desc := tag.Get(tagDescription); desc != "" {
		schema.Description = desc
	}
	if desc := tag.Get(tagDoc); desc != "" {
		schema.Description = desc
	}

	applyStringSchemaTags(schema, tag)
	applyNumericSchemaTags(schema, tag)
	applyArraySchemaTags(schema, tag)

	// Enum validation
	if enum := tag.Get(tagEnum); enum != "" {
		values := strings.Split(enum, ",")
		schema.Enum = make([]interface{}, len(values))
		for i, v := range values {
			schema.Enum[i] = strings.TrimSpace(v)
		}
	}
	// Example
	if example := tag.Get(tagExample); example != "" {
		schema.Example = example
	}
	// Const (OpenAPI 3.1). Stored as a marker extension on the version-agnostic
	// schema; promoted to a real `const` for 3.1 and stripped for 3.0.
	if constVal := tag.Get(tagConst); constVal != "" {
		if schema.Extensions == nil {
			schema.Extensions = make(map[string]any)
		}
		schema.Extensions[extOkapiConst] = constVal
	}
}

// applyStringSchemaTags applies minLength, maxLength, pattern, and format.
func applyStringSchemaTags(schema *openapi3.Schema, tag reflect.StructTag) {
	if maxLen := tag.Get(tagMaxLength); maxLen != "" {
		if val, err := strconv.ParseUint(maxLen, 10, 64); err == nil {
			schema.MaxLength = ptr(val)
		}
	}
	if minLen := tag.Get(tagMinLength); minLen != "" {
		if val, err := strconv.ParseUint(minLen, 10, 64); err == nil {
			schema.MinLength = val
		}
	}
	if pattern := tag.Get(tagPattern); pattern != "" {
		schema.Pattern = pattern
	}
	if format := tag.Get(tagFormat); format != "" {
		schema.Format = format
	}
}

// applyNumericSchemaTags applies min, max, exclusive bounds, and multipleOf.
func applyNumericSchemaTags(schema *openapi3.Schema, tag reflect.StructTag) {
	if maxTag := tag.Get(tagMax); maxTag != "" {
		if val, err := strconv.ParseFloat(maxTag, 64); err == nil {
			schema.Max = ptr(val)
		}
	}
	if minTag := tag.Get(tagMin); minTag != "" {
		if val, err := strconv.ParseFloat(minTag, 64); err == nil {
			schema.Min = ptr(val)
		}
	}
	// Exclusive bounds. The base document is OpenAPI 3.0, where these are
	// booleans modifying minimum/maximum; the bound goes in Min/Max with the
	// flag set. transformSchemaTo31 rewrites this to the 3.1 numeric form.
	if exclusiveMin := tag.Get(tagExclusiveMin); exclusiveMin != "" {
		if val, err := strconv.ParseFloat(exclusiveMin, 64); err == nil {
			schema.Min = ptr(val)
			schema.ExclusiveMin = openapi3.ExclusiveBound{Bool: ptr(true)}
		}
	}
	if exclusiveMax := tag.Get(tagExclusiveMax); exclusiveMax != "" {
		if val, err := strconv.ParseFloat(exclusiveMax, 64); err == nil {
			schema.Max = ptr(val)
			schema.ExclusiveMax = openapi3.ExclusiveBound{Bool: ptr(true)}
		}
	}
	if multipleOf := tag.Get(tagMultipleOf); multipleOf != "" {
		if val, err := strconv.ParseFloat(multipleOf, 64); err == nil {
			schema.MultipleOf = ptr(val)
		}
	}
}

// applyArraySchemaTags applies slice item constraints and object property counts.
func applyArraySchemaTags(schema *openapi3.Schema, tag reflect.StructTag) {
	if maxItems := tag.Get(tagMaxItems); maxItems != "" {
		if val, err := strconv.ParseUint(maxItems, 10, 64); err == nil {
			schema.MaxItems = ptr(val)
		}
	}
	if minItems := tag.Get(tagMinItems); minItems != "" {
		if val, err := strconv.ParseUint(minItems, 10, 64); err == nil {
			schema.MinItems = val
		}
	}
	if uniqueItems := tag.Get(tagUniqueItems); uniqueItems == constTRUE {
		schema.UniqueItems = true
	}
	if maxProps := tag.Get(tagMaxProperties); maxProps != "" {
		if val, err := strconv.ParseUint(maxProps, 10, 64); err == nil {
			schema.MaxProps = ptr(val)
		}
	}
	if minProps := tag.Get(tagMinProperties); minProps != "" {
		if val, err := strconv.ParseUint(minProps, 10, 64); err == nil {
			schema.MinProps = val
		}
	}
}

// extractPathParams extracts path parameters from a route path
// Supports patterns like:
// - /users/{id} -> id (string)
// - /users/:id -> id (string)
// - /users/{user_id} -> user_id (string)
// - /users/{id:int} -> id (int)
// - /users/:id:int -> id (int)
// - /users/{user_id:uuid} -> user_id (uuid)
func extractPathParams(path string) []*openapi3.ParameterRef {
	params := []*openapi3.ParameterRef{}
	seen := map[string]struct{}{}

	// {id} or {id:type}
	braceRe := regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]+))?\}`)

	// :id or :id:type
	colonRe := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)(?::([^/]+))?`)

	// 1 Extract { } params
	braceMatches := braceRe.FindAllStringSubmatch(path, -1)
	for _, match := range braceMatches {
		name := match[1]
		typ := ""

		if len(match) > 2 && match[2] != "" {
			typ = normalizeType(match[2])
		} else {
			typ = inferTypeFromParamName(name)
		}

		seen[name] = struct{}{}
		params = append(params, buildPathParam(name, typ))
	}

	// 2 Remove { } segments before scanning for :params
	cleanPath := braceRe.ReplaceAllString(path, "")

	// 3 Extract :params safely
	colonMatches := colonRe.FindAllStringSubmatch(cleanPath, -1)
	for _, match := range colonMatches {
		name := match[1]
		if _, exists := seen[name]; exists {
			continue
		}

		typ := ""
		if len(match) > 2 && match[2] != "" {
			typ = normalizeType(match[2])
		} else {
			typ = inferTypeFromParamName(name)
		}

		seen[name] = struct{}{}
		params = append(params, buildPathParam(name, typ))
	}

	return params
}
func buildPathParam(name, typ string) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        name,
			In:          "path",
			Required:    true,
			Schema:      getSchemaForType(typ),
			Description: generateParamDescription(name, typ),
		},
	}
}

// ExtractPathParams extracts all struct fields with "path" or "param" tags
func extractPathParamsFromStruct(v any) []*openapi3.ParameterRef {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	typ := val.Type()

	// Handle pointer types
	if typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
		typ = val.Type()
	}

	// Must be a struct
	if typ.Kind() != reflect.Struct {
		return nil
	}

	var params []*openapi3.ParameterRef

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check for "path" or "param" tag
		paramName := getPathParamName(field)
		if paramName == "" {
			continue
		}

		// Get the field type as string
		fieldType := getFieldTypeName(field.Type)

		param := buildPathParam(paramName, fieldType)
		params = append(params, param)
	}
	return params
}

// getPathParamName extracts the parameter name from "path" or "param" struct tags.
func getPathParamName(field reflect.StructField) string {
	// Check "path" tag first
	if tag := field.Tag.Get(tagPath); tag != "" {
		return parseTagName(tag)
	}

	// Check "param" tag
	if tag := field.Tag.Get(tagParam); tag != "" {
		return parseTagName(tag)
	}

	return ""
}

// parseTagName extracts the name from a tag value
func parseTagName(tag string) string {
	if tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return ""
	}

	return name
}

// getFieldTypeName returns the string representation of a reflect.Type
func getFieldTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return constString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return constInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return constUint
	case reflect.Float32, reflect.Float64:
		return constFloat
	case reflect.Bool:
		return constBool
	case reflect.Ptr:
		return getFieldTypeName(t.Elem())
	default:
		return t.String()
	}
}

func normalizeType(t string) string {
	switch strings.ToLower(t) {
	case constInt, "integer":
		return constInt
	case constInt64:
		return constInt64
	case constFloat, "float32":
		return constFloat
	case constFloat64, "double":
		return constFloat64
	case constBool, "boolean":
		return constBool
	case constUUID:
		return constUUID
	case constDate:
		return constDate
	case "datetime", "date-time":
		return constDateTime
	case "string":
		return constString
	default:
		return t
	}
}

// inferTypeFromParamName attempts to infer the parameter type from its name
func inferTypeFromParamName(name string) string {
	n := strings.ToLower(name)
	//  Explicit identifier patterns (highest priority)
	// id, user_id, order_id, etc.
	if n == "id" || strings.HasSuffix(n, "_id") {
		return constUUID
	}
	if strings.HasSuffix(n, "id") {
		return constString
	}

	// Pagination & numeric counters
	if strings.Contains(n, "count") ||
		strings.Contains(n, "total") ||
		strings.Contains(n, "limit") ||
		strings.Contains(n, "offset") ||
		strings.Contains(n, "page") ||
		strings.Contains(n, "size") ||
		strings.Contains(n, "number") ||
		strings.Contains(n, "index") {
		return constInt
	}

	// Date & time (timestamps)
	if strings.Contains(n, "created_at") ||
		strings.Contains(n, "updated_at") ||
		strings.Contains(n, "deleted_at") ||
		strings.HasSuffix(n, "_at") ||
		strings.Contains(n, "timestamp") {
		return constDateTime
	}

	// Pure date (not time)
	if strings.Contains(n, "date") ||
		strings.HasSuffix(n, "_on") {
		return constDate
	}

	// Boolean flags
	if strings.HasPrefix(n, "is_") ||
		strings.HasPrefix(n, "has_") ||
		strings.HasPrefix(n, "can_") ||
		strings.HasPrefix(n, "should_") ||
		strings.HasPrefix(n, "enable") ||
		strings.HasPrefix(n, "disable") ||
		strings.HasPrefix(n, "active") {
		return constBool
	}
	return constString
}

// generateParamDescription generates a human-readable description for a parameter
func generateParamDescription(name, typ string) string {
	// Convert snake_case to human readable
	words := strings.Split(strings.ReplaceAll(name, "_", " "), " ")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	readable := strings.Join(words, " ")

	switch typ {
	case "uuid":
		return fmt.Sprintf("%s identifier", readable)
	case "int":
		return fmt.Sprintf("%s (integer)", readable)
	case "bool":
		return fmt.Sprintf("%s (boolean)", readable)
	case "date":
		return fmt.Sprintf("%s (date)", readable)
	case "date-time":
		return fmt.Sprintf("%s (date-time)", readable)
	default:
		return readable
	}
}

func getSchemaForType(typ string) *openapi3.SchemaRef {
	switch strings.ToLower(typ) {
	case "string":
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	case "int", "integer":
		return openapi3.NewSchemaRef("", openapi3.NewInt32Schema())
	case "int64":
		return openapi3.NewSchemaRef("", openapi3.NewInt64Schema())
	case "float", "float32":
		schema := openapi3.NewFloat64Schema()
		schema.Format = "float"
		return openapi3.NewSchemaRef("", schema)
	case "float64", "double":
		return openapi3.NewSchemaRef("", openapi3.NewFloat64Schema())
	case "bool", "boolean":
		return openapi3.NewSchemaRef("", openapi3.NewBoolSchema())
	case "uuid":
		schema := openapi3.NewStringSchema()
		schema.Format = constUUID
		return openapi3.NewSchemaRef("", schema)
	case "date":
		schema := openapi3.NewStringSchema()
		schema.Format = constDate
		return openapi3.NewSchemaRef("", schema)
	case "datetime", constDateTime:
		schema := openapi3.NewStringSchema()
		schema.Format = constDateTime
		return openapi3.NewSchemaRef("", schema)
	default:
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	}
}
func addSecurity(spec *openapi3.T, op *openapi3.Operation, r *Route) {
	if r.bearerAuth {
		op.Security = &openapi3.SecurityRequirements{
			openapi3.SecurityRequirement{
				"BearerAuth": {},
			},
		}
	}
	if r.basicAuth {
		if op.Security == nil {
			op.Security = &openapi3.SecurityRequirements{}
		}
		*op.Security = append(*op.Security, openapi3.SecurityRequirement{
			"BasicAuth": {},
		})
	}
	if len(r.security) != 0 {
		// Initialize an empty slice for security requirements
		op.Security = &openapi3.SecurityRequirements{}
		for _, sec := range r.security {
			valid := true
			for scheme := range sec {
				if _, exists := spec.Components.SecuritySchemes[scheme]; !exists {
					slog.Warn("Security scheme not defined in OpenAPI spec", "scheme", scheme)
					valid = false
					break
				}
			}
			if valid {
				*op.Security = append(*op.Security, sec)
			}
		}
	}

}

// normalizeToStructPointer ensures the input is a pointer to a struct.
// It accepts both struct values and struct pointers, auto-converting
// structs to pointers when needed.
func normalizeToStructPointer(input any, inputType string) reflect.Value {
	v := reflect.ValueOf(input)

	// If a struct was passed, wrap it into a pointer
	if v.Kind() == reflect.Struct {
		_ptr := reflect.New(v.Type())
		_ptr.Elem().Set(v)
		v = _ptr
	}

	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic(fmt.Sprintf(
			"Invalid %s: expected struct or non-nil pointer to struct, but got %T. "+
				"Example: My%s{} or &My%s{}",
			inputType, input, capitalize(inputType), capitalize(inputType),
		))
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		panic(fmt.Sprintf(
			"Invalid %s: expected struct or non-nil pointer to struct, but got %T",
			inputType, input,
		))
	}

	return elem
}

// extractFieldInfo extracts common field information
func extractFieldInfo(field reflect.StructField) fieldInfo {
	description := field.Tag.Get(tagDescription)
	if description == "" {
		description = field.Tag.Get(tagDoc)
	}
	return fieldInfo{
		field:       field,
		required:    field.Tag.Get(tagRequired) == constTRUE,
		description: description,
	}
}

// createParameter creates an OpenAPI parameter
func createParameter(name, location string, info fieldInfo) *openapi3.ParameterRef {
	return &openapi3.ParameterRef{
		Value: &openapi3.Parameter{
			Name:        name,
			In:          location,
			Required:    info.required,
			Schema:      getSchemaForType(info.field.Type.Name()),
			Description: info.description,
		},
	}
}

// createHeader creates an OpenAPI Response Header
func createHeader(name string, info fieldInfo) *openapi3.HeaderRef {
	return &openapi3.HeaderRef{
		Value: &openapi3.Header{
			Parameter: openapi3.Parameter{
				Name:        name,
				Required:    info.required,
				Schema:      getSchemaForType(info.field.Type.Name()),
				Description: info.description,
			},
		},
	}
}

// processField processes a single struct field for parameter extraction
func (r *Route) processField(info fieldInfo, isRequest bool) bool {
	sf := info.field

	// Header parameter
	if isRequest {
		if key := sf.Tag.Get(tagHeader); key != "" {
			param := createParameter(key, paramHeader, info)
			r.headers = append(r.headers, param)
			return true
		}
	} else {
		// Response Header
		// Initialize responseHeaders map if it doesn't exist
		if r.responseHeaders == nil {
			r.responseHeaders = make(map[string]*openapi3.HeaderRef)
		}
		if key := sf.Tag.Get(tagHeader); key != "" {
			header := createHeader(key, info)
			r.responseHeaders[key] = header
			return true
		}
	}

	// Cookie parameter
	if key := sf.Tag.Get(tagCookie); key != "" {
		param := createParameter(key, paramCookie, info)
		r.cookies = append(r.cookies, param)
		return true
	}

	// Query parameter (request only)
	if isRequest {
		if key := sf.Tag.Get(tagQuery); key != "" {
			param := createParameter(key, paramQuery, info)
			r.queryParams = append(r.queryParams, param)
			return true
		}

		// Path parameter (request only)
		if key := sf.Tag.Get(tagPath); key != "" {
			// Path params are handled elsewhere
			return true
		}
		// Path parameter (request only)
		if key := sf.Tag.Get(tagParam); key != "" {
			return true
		}
	}

	// Body field
	if sf.Tag.Get(tagJSON) == bodyValue || sf.Name == bodyField {
		r.processBodyField(sf, isRequest)
		return true
	}

	return false
}

// processBodyField processes a body field
func (r *Route) processBodyField(field reflect.StructField, isRequest bool) {
	bodyPtr := reflect.New(field.Type)
	schema := reflectToSchemaWithInfo(bodyPtr.Interface()).Schema

	if isRequest {
		r.request = schema
	} else {
		r.responses[defaultStatus] = schema
	}
}

// processFields processes all fields in a struct
func (r *Route) processFields(v reflect.Value, t reflect.Type, isRequest bool) bool {
	hasExplicitBinding := false

	for i := 0; i < v.NumField(); i++ {
		fInfo := extractFieldInfo(t.Field(i))
		if r.processField(fInfo, isRequest) {
			hasExplicitBinding = true
		}
	}

	return hasExplicitBinding
}

// getResponseStatus extracts the HTTP status code from response struct
func getResponseStatus(v reflect.Value) int {
	if statusField := v.FieldByName("Status"); statusField.IsValid() && statusField.Kind() == reflect.Int && int(statusField.Int()) > 0 {
		return int(statusField.Int())
	}
	return defaultStatus
}

func (r *Route) generateResponseSchema(input any) {
	v := normalizeToStructPointer(input, "response")
	t := v.Type()
	status := getResponseStatus(v)

	hasExplicitBinding := r.processFields(v, t, false)

	// Fallback: if no explicit binding, use whole struct as body
	if !hasExplicitBinding {
		r.responses[status] = reflectToSchemaWithInfo(input).Schema
	}
}

func (r *Route) generateRequestSchema(input any) {
	v := normalizeToStructPointer(input, "request")
	t := v.Type()

	hasExplicitBinding := r.processFields(v, t, true)

	// Fallback: if no explicit binding, use whole struct as body
	if !hasExplicitBinding {
		r.request = reflectToSchemaWithInfo(input).Schema
	}
	r.pathParams = extractPathParamsFromStruct(input)
}
