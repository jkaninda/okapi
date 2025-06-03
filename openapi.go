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
	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const (
	Int      = "int"
	Int64    = "int64"
	Float    = "float"
	DateTime = "date-time"
	Date     = "date"
	UUID     = "uuid"
	Bool     = "bool"
	String   = "string"
)

// RouteOption defines a function type that modifies a Route's documentation properties
type RouteOption func(*Route)

// OpenAPI contains configuration for generating OpenAPI/Swagger documentation
type OpenAPI struct {
	Title   string // Title of the API
	Version string // Version of the API
	// PathPrefix is the URL prefix for accessing the documentation
	PathPrefix string           // e.g., "/docs" (default)
	Servers    openapi3.Servers // List of server URLs where the API is hosted
}

// ptr is a helper function that returns a pointer to any value
func ptr[T any](v T) *T { return &v }

// DocSummary sets a short summary description for the route
func DocSummary(summary string) RouteOption {
	return func(doc *Route) {
		doc.Summary = summary
	}
}

// DocPathParam adds a path parameter to the route documentation
// name: parameter name
// typ: parameter type (e.g., "string", "int", "uuid")
// desc: parameter description
func DocPathParam(name, typ, desc string) RouteOption {
	return func(doc *Route) {
		schema := getSchemaForType(typ)
		doc.PathParams = append(doc.PathParams, &openapi3.ParameterRef{
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

// DocAutoPathParams automatically extracts path parameters from the route path
// and adds them to the documentation.
// It skips parameters that are already defined.
func DocAutoPathParams() RouteOption {
	return func(doc *Route) {
		pathParams := extractPathParams(doc.Path)
		for _, param := range pathParams {
			// Check if parameter already exists to avoid duplicates
			exists := false
			for _, existing := range doc.PathParams {
				if existing.Value.Name == param.Value.Name {
					exists = true
					break
				}
			}
			if !exists {
				doc.PathParams = append(doc.PathParams, param)
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
	return func(doc *Route) {
		schema := getSchemaForType(typ)
		doc.QueryParams = append(doc.QueryParams, &openapi3.ParameterRef{
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
	return func(doc *Route) {
		schema := getSchemaForType(typ)
		doc.Headers = append(doc.Headers, &openapi3.ParameterRef{
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
	return func(doc *Route) {
		doc.Tags = append(doc.Tags, tag)
	}
}

// DocTags adds multiple tags to categorize the route
func DocTags(tags ...string) RouteOption {
	return func(doc *Route) {
		doc.Tags = append(doc.Tags, tags...)
	}
}

// DocResponse defines the response schema for the route
// v: a Go value whose type will be used to generate the response schema
func DocResponse(v any) RouteOption {
	return func(doc *Route) {
		if v == nil {
			return
		}
		doc.Response = reflectToSchema(v)
	}
}

// DocRequest defines the request body schema for the route
// v: a Go value whose type will be used to generate the request schema
func DocRequest(v any) RouteOption {
	return func(doc *Route) {
		if v == nil {
			return
		}
		doc.Request = reflectToSchema(v)
	}
}

// DocBearerAuth marks the route as requiring Bearer token authentication
func DocBearerAuth() RouteOption {
	return func(doc *Route) {
		doc.RequiresAuth = true
	}
}

// buildOpenAPISpec constructs the complete OpenAPI specification document
// by aggregating all the route documentation into a single OpenAPI 3.0 spec
func (o *Okapi) buildOpenAPISpec() {
	spec := &openapi3.T{
		OpenAPI: OpenApiVersion,
		Info: &openapi3.Info{
			Title:   o.openAPi.Title,
			Version: o.openAPi.Version,
		},
		Paths:   &openapi3.Paths{},
		Servers: o.openAPi.Servers,
		Components: &openapi3.Components{
			SecuritySchemes: openapi3.SecuritySchemes{
				"BearerAuth": &openapi3.SecuritySchemeRef{
					Value: &openapi3.SecurityScheme{
						Type:         "http",
						Scheme:       "bearer",
						BearerFormat: "JWT",
					},
				},
			},
		},
	}

	// Process all registered routes
	for _, r := range o.routes {
		// Auto-extract path parameters if none are defined
		if len(r.PathParams) == 0 {
			DocAutoPathParams()(r)
		}
		tags := r.Tags
		if tags == nil {
			tags = append(tags, r.GroupPath)
		}
		item := spec.Paths.Value(r.Path)
		if item == nil {
			item = &openapi3.PathItem{}
			spec.Paths.Set(r.Path, item)
		}

		op := &openapi3.Operation{
			Summary:    r.Summary,
			Tags:       tags,
			Parameters: append(append(r.PathParams, r.QueryParams...), r.Headers...),
			Responses:  &openapi3.Responses{},
		}

		if r.RequiresAuth {
			op.Security = &openapi3.SecurityRequirements{
				openapi3.SecurityRequirement{
					"BearerAuth": {},
				},
			}
		}

		if r.Request != nil {
			op.RequestBody = &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.NewContentWithJSONSchemaRef(r.Request),
				},
			}
		}

		if r.Response != nil {
			op.Responses.Set("200", &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: ptr("Success"),
					Content:     openapi3.NewContentWithJSONSchemaRef(r.Response),
				},
			})
		}

		// Add default error responses
		op.Responses.Set("400", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: ptr("Bad Request"),
			},
		})

		if r.RequiresAuth {
			op.Responses.Set("401", &openapi3.ResponseRef{
				Value: &openapi3.Response{
					Description: ptr("Unauthorized"),
				},
			})
		}

		op.Responses.Set("500", &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Description: ptr("Internal Server Error"),
			},
		})

		// Assign operation to correct HTTP verb
		switch r.Method {
		case "GET":
			item.Get = op
		case "POST":
			item.Post = op
		case "PUT":
			item.Put = op
		case "DELETE":
			item.Delete = op
		case "PATCH":
			item.Patch = op
		case "HEAD":
			item.Head = op
		case "OPTIONS":
			item.Options = op
		}
	}

	o.openapiSpec = spec
}

// reflectToSchema converts a Go type to an OpenAPI schema using reflection
func reflectToSchema(v any) *openapi3.SchemaRef {
	t := reflect.TypeOf(v)

	// Handle pointers
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return typeToSchema(t)
}

// typeToSchema converts a reflect.Type to an OpenAPI SchemaRef
func typeToSchema(t reflect.Type) *openapi3.SchemaRef {
	switch t.Kind() {
	case reflect.String:
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema := openapi3.NewIntegerSchema()
		if t.Kind() == reflect.Int64 {
			schema.Format = Int64
		} else {
			schema.Format = "int32"
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema := openapi3.NewIntegerSchema()
		schema.Min = ptr(float64(0))
		if t.Kind() == reflect.Uint64 {
			schema.Format = "int64"
		} else {
			schema.Format = "int32"
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Float32, reflect.Float64:
		schema := openapi3.NewFloat64Schema()
		if t.Kind() == reflect.Float32 {
			schema.Format = Float
		} else {
			schema.Format = "double"
		}
		return openapi3.NewSchemaRef("", schema)

	case reflect.Bool:
		return openapi3.NewSchemaRef("", openapi3.NewBoolSchema())

	case reflect.Slice, reflect.Array:
		elemSchema := typeToSchema(t.Elem())
		schema := openapi3.NewArraySchema()
		schema.Items = elemSchema
		return openapi3.NewSchemaRef("", schema)

	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			valueSchema := typeToSchema(t.Elem())
			schema := openapi3.NewObjectSchema()
			schema.AdditionalProperties = openapi3.AdditionalProperties{
				Schema: valueSchema,
			}
			return openapi3.NewSchemaRef("", schema)
		}
		// For non-string keys, treat as generic object
		return openapi3.NewSchemaRef("", openapi3.NewObjectSchema())

	case reflect.Struct:
		return structToSchema(t)

	case reflect.Interface:
		// For interface{}, return a generic schema
		return openapi3.NewSchemaRef("", &openapi3.Schema{})

	default:
		// Fallback for unsupported types
		return openapi3.NewSchemaRef("", openapi3.NewObjectSchema())
	}
}

// structToSchema converts a struct type to an OpenAPI schema
func structToSchema(t reflect.Type) *openapi3.SchemaRef {
	// Handle special types
	if t == reflect.TypeOf(time.Time{}) {
		schema := openapi3.NewStringSchema()
		schema.Format = "date-time"
		return openapi3.NewSchemaRef("", schema)
	}

	schema := openapi3.NewObjectSchema()
	required := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fieldName := getJSONFieldName(field)
		if fieldName == "-" {
			continue
		}

		fieldSchema := typeToSchema(field.Type)

		// Add description from comments or tags
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Value.Description = desc
		}

		schema.WithProperty(fieldName, fieldSchema.Value)

		// Check if field is required
		if isRequiredField(field) {
			required = append(required, fieldName)
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

// isRequiredField determines if a struct field is required
func isRequiredField(field reflect.StructField) bool {
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

	// Check if field is a pointer (usually optional)
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	// Default to required for non-pointer fields without omitempty
	return !strings.Contains(jsonTag, "omitempty")
}

// extractPathParams extracts path parameters from a route path
// Supports patterns like:
// - /users/{id} -> id (string)
// - /users/{user_id} -> user_id (string)
// - /users/{id:int} -> id (int)
// - /users/{user_id:uuid} -> user_id (uuid)
func extractPathParams(path string) []*openapi3.ParameterRef {
	params := []*openapi3.ParameterRef{}

	// Find all parameters in curly braces
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		paramDef := match[1]
		var name, typ, description string

		// Check if type is specified (e.g., {id:int} or {user_id:uuid})
		if strings.Contains(paramDef, ":") {
			parts := strings.SplitN(paramDef, ":", 2)
			name = parts[0]
			typ = parts[1]
		} else {
			name = paramDef
			typ = inferTypeFromParamName(name)
		}

		description = generateParamDescription(name, typ)
		schema := getSchemaForType(typ)

		params = append(params, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:        name,
				In:          "path",
				Required:    true,
				Schema:      schema,
				Description: description,
			},
		})
	}

	return params
}

// inferTypeFromParamName attempts to infer the parameter type from its name
func inferTypeFromParamName(name string) string {
	name = strings.ToLower(name)

	// Common ID patterns
	if strings.HasSuffix(name, "_id") || name == "id" {
		return UUID // Assume UUIDs for IDs
	}

	// Numeric patterns
	if strings.Contains(name, "count") || strings.Contains(name, "limit") ||
		strings.Contains(name, "offset") || strings.Contains(name, "page") ||
		strings.Contains(name, "size") || strings.Contains(name, "number") {
		return Int
	}

	// Date patterns
	if strings.Contains(name, "date") || strings.Contains(name, "time") {
		return Date
	}

	// Boolean patterns
	if strings.HasPrefix(name, "is_") || strings.HasPrefix(name, "has_") ||
		strings.HasPrefix(name, "can_") || strings.HasPrefix(name, "should_") {
		return Bool
	}

	// Default to string
	return String
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
		schema.Format = UUID
		return openapi3.NewSchemaRef("", schema)
	case "date":
		schema := openapi3.NewStringSchema()
		schema.Format = Date
		return openapi3.NewSchemaRef("", schema)
	case "datetime", DateTime:
		schema := openapi3.NewStringSchema()
		schema.Format = DateTime
		return openapi3.NewSchemaRef("", schema)
	default:
		return openapi3.NewSchemaRef("", openapi3.NewStringSchema())
	}
}
