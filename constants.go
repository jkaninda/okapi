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

const (
	defaultMaxMemory       = 32 << 20 // 32 MB
	constContentTypeHeader = "Content-Type"
	constAcceptHeader      = "Accept"
	constLocationHeader    = "Location"
	okapiName              = "Okapi"
	constTRUE              = "true"
	constIndex             = "index.html"

	openApiVersion                     = "3.0.3"
	openApiVersion31                   = "3.1.0"
	openApiDocPrefix                   = "/docs"
	openApiDocPath                     = "/openapi.json"
	openApiYamlPath                    = "/openapi.yaml"
	openApiDocPath30                   = "/openapi-3.0.json"
	openApiYamlPath30                  = "/openapi-3.0.yaml"
	jsonSchemaDialect                  = "https://spec.openapis.org/oas/3.1/dialect/base"
	docSwaggerPath                     = "/swagger"
	docRedocPath                       = "/redoc"
	docScalarPath                      = "/scalar"
	docFaviconPath                     = "/docs/favicon.png"
	constAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	constAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	constAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	constAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	constAccessControlMaxAge           = "Access-Control-Max-Age"
	constAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
)

// HTTP methods
const (
	methodDelete  = http.MethodDelete
	methodGet     = http.MethodGet
	methodHead    = http.MethodHead
	methodOptions = http.MethodOptions
	methodPost    = http.MethodPost
	methodPut     = http.MethodPut
	methodPatch   = http.MethodPatch
)
const (
	// Tag names
	tagRequired      = "required"
	tagDescription   = "description"
	tagDoc           = "doc"
	tagHeader        = "header"
	tagForm          = "form"
	tagQuery         = "query"
	tagCookie        = "cookie"
	tagPath          = "path"
	tagParam         = "param"
	tagJSON          = "json"
	tagMin           = "min"
	tagMax           = "max"
	tagMinLength     = "minLength"
	tagMaxLength     = "maxLength"
	tagDefault       = "default"
	tagFormat        = "format"
	tagPattern       = "pattern"
	tagEnum          = "enum"
	tagDeprecated    = "deprecated"
	tagHidden        = "hidden"
	tagMultipleOf    = "multipleOf"
	tagExample       = "example"
	tagConst         = "const"
	tagMaxItems      = "maxItems"
	tagMinItems      = "minItems"
	tagUniqueItems   = "uniqueItems"
	tagExclusiveMin  = "exclusiveMin"
	tagExclusiveMax  = "exclusiveMax"
	tagMinProperties = "minProperties"
	tagMaxProperties = "maxProperties"

	// extOkapiConst is an internal marker extension used to carry an OpenAPI 3.1
	// `const` value on the version-agnostic base schema. It is promoted to a real
	// `const` keyword when deriving the 3.1 document and stripped from the 3.0 one,
	// so neither served document exposes the marker.
	extOkapiConst = "x-okapi-const"

	// Format types
	formatEmail    = "email"
	formatDateTime = "date-time"
	formatDate     = "date"
	formatTime     = "time"
	formatDuration = "duration"
	formatIPv4     = "ipv4"
	formatIPv6     = "ipv6"
	formatUUID     = "uuid"
	formatRegex    = "regex"
	formatHostname = "hostname"
	formatUri      = "uri"
	// formats
	formatURL          = "url"
	formatURIReference = "uri-reference"
	formatByte         = "byte"
	formatBase64       = "base64"
	formatMAC          = "mac"
	formatCIDR         = "cidr"
	formatE164         = "e164"
	formatPhone        = "phone"
	formatCreditCard   = "credit-card"
	formatSemver       = "semver"
	formatJSONPointer  = "json-pointer"
	formatULID         = "ulid"
	// string content formats
	formatAlpha        = "alpha"
	formatAlphanumeric = "alphanumeric"
	formatNumeric      = "numeric"
	formatASCII        = "ascii"
	formatLowercase    = "lowercase"
	formatUppercase    = "uppercase"
	formatSlug         = "slug"
	formatHexColor     = "hexcolor"
	// Special values
	bodyValue = "body"
	bodyField = "Body"

	// Parameter locations
	paramHeader = "header"
	paramQuery  = "query"
	paramCookie = "cookie"

	// Default HTTP status
	defaultStatus    = 200
	constLocalhost   = "localhost"
	constDevelopment = "development"

	requestIDHeader = "X-Request-ID"
)
