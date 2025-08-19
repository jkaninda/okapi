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
	defaultMaxMemory  = 32 << 20 // 32 MB
	ContentTypeHeader = "Content-Type"
	AcceptHeader      = "Accept"
	LocationHeader    = "Location"
	okapiName         = "Okapi"
	TRUE              = "true"

	openApiVersion                = "3.0.0"
	openApiDocPrefix              = "/docs"
	openApiDocPath                = "/openapi.json"
	AccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	AccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	AccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	AccessControlAllowMethods     = "Access-Control-Allow-Methods"
	AccessControlMaxAge           = "Access-Control-Max-Age"
	AccessControlAllowCredentials = "Access-Control-Allow-Credentials"
)

// Constants for HTTP StatusCode Codes
const (
	StatusNotFound            = http.StatusNotFound
	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusInternalServerError = http.StatusInternalServerError
)

// HTTP methods
const (
	DELETE  = http.MethodDelete
	GET     = http.MethodGet
	HEAD    = http.MethodHead
	OPTIONS = http.MethodOptions
	POST    = http.MethodPost
	PUT     = http.MethodPut
	PATCH   = http.MethodPatch
	CONNECT = http.MethodConnect
	TRACE   = http.MethodTrace
)
const banner = `
   ___  _               _ 
  / _ \| | ____ _ _ __ (_)
 | | | | |/ / _` + "`" + ` | '_ \| |
 | |_| |   < (_| | |_) | |
  \___/|_|\_\__,_| .__/|_|
    		 	 |_|`
