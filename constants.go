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

	openApiVersion                     = "3.0.0"
	openApiDocPrefix                   = "/docs"
	openApiDocPath                     = "/openapi.json"
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
	methodConnect = http.MethodConnect
	methodTrace   = http.MethodTrace
)
const (
	// Tag names
	tagRequired    = "required"
	tagDescription = "description"
	tagDoc         = "doc"
	tagHeader      = "header"
	tagForm        = "form"
	tagQuery       = "query"
	tagCookie      = "cookie"
	tagPath        = "path"
	tagParam       = "param"
	tagJSON        = "json"
	tagMin         = "min"
	tagMax         = "max"
	tagMinLength   = "minLength"
	tagMaxLength   = "maxLength"
	tagDefault     = "default"
	tagFormat      = "format"
	tagPattern     = "pattern"
	tagEnum        = "enum"
	tagDeprecated  = "deprecated"
	tagHidden      = "hidden"

	// Format types
	formatEmail    = "email"
	formatDateTime = "date-time"
	formatDate     = "date"
	formatDuration = "duration"
	formatIPv4     = "ipv4"
	formatIPv6     = "ipv6"
	formatUUID     = "uuid"
	formatRegex    = "regex"
	// Special values
	bodyValue = "body"
	bodyField = "Body"

	// Parameter locations
	paramHeader = "header"
	paramQuery  = "query"
	paramCookie = "cookie"

	// Default HTTP status
	defaultStatus = 200
)
const banner = `:::::::::::::: 🦒 ::::::::::::::
    	Okapi Web Framework 
::::::::::::::::::::::::::::::::`
