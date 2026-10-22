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
	"reflect"
	"regexp"
)

var routeParamRegex = regexp.MustCompile(`\{([^}.]+)(?:\.\.\.)?\}`)

// checkRouteInput logs the fields of a route's input type that can never bind,
// so the mistake shows when the server starts rather than as a 400 on every request.
func (o *Okapi) checkRouteInput(route *Route) {
	input, ok := bindingInput(route.handle)
	if !ok || o.logger == nil {
		return
	}
	for _, problem := range inputProblems(input, route.Path) {
		o.logger.Error("route input can never bind",
			"method", route.Method,
			"path", route.Path,
			"input", input.String(),
			"problem", problem,
			"component", "okapi",
		)
	}
}

// inputProblems lists the fields of input that can never bind on a route at path.
func inputProblems(input reflect.Type, path string) []string {
	if input.Kind() != reflect.Struct {
		return nil
	}
	params := map[string]bool{}
	for _, m := range routeParamRegex.FindAllStringSubmatch(path, -1) {
		params[m[1]] = true
	}

	var problems []string
	for i := 0; i < input.NumField(); i++ {
		sf := input.Field(i)
		if !sf.IsExported() {
			continue
		}
		if sf.Name == bodyField {
			if !decodable(sf.Type) {
				problems = append(problems, fmt.Sprintf("field Body: %s cannot be decoded from a request body", sf.Type))
			}
			continue
		}
		// Required fields only: one input is often shared by a create route and its /{id} update route.
		for _, tag := range []string{tagParam, tagPath} {
			if key := sf.Tag.Get(tag); key != "" && !params[key] && sf.Tag.Get(tagRequired) == constTRUE {
				problems = append(problems, fmt.Sprintf("field %s: required path parameter %q is not in the route", sf.Name, key))
			}
		}
		for _, tag := range []string{tagParam, tagPath, tagQuery, tagHeader, tagCookie} {
			if sf.Tag.Get(tag) != "" && !parsable(sf.Type) {
				problems = append(problems, fmt.Sprintf("field %s: %s cannot be parsed from a %s value", sf.Name, sf.Type, tag))
				break
			}
		}
	}
	return problems
}

// parsable reports whether setWithType can fill a field of type t from a string.
func parsable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Pointer:
		return parsable(t.Elem())
	case reflect.Slice:
		switch t.Elem().Kind() {
		case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return true
		}
	}
	return false
}

// decodable reports whether a request body decoder can fill a value of type t.
func decodable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
		return false
	case reflect.Interface:
		return t.NumMethod() == 0
	case reflect.Pointer:
		return decodable(t.Elem())
	}
	return true
}
