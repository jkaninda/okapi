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
	"strconv"
	"strings"
)

func (c *Context) bindStruct(input any) error {
	v := reflect.ValueOf(input).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		// Handle data extraction and assignment
		if err := c.extractAndSetField(field, sf); err != nil {
			return err
		}

		// Handle validations
		if err := c.validateField(field, sf); err != nil {
			return err
		}
	}

	return nil
}

// extractAndSetField extracts a field's value from request sources (headers, query, cookies, params, body)
// and assigns it to the struct field.
func (c *Context) extractAndSetField(field reflect.Value, sf reflect.StructField) error {
	var raw string
	var rawSlice []string

	// Header
	if key := sf.Tag.Get(tagHeader); key != "" {
		raw = c.Header(key)
	}

	// Query - supports slices and comma-separated values
	if key := sf.Tag.Get(tagQuery); key != "" {
		if field.Kind() == reflect.Slice {
			rawSlice = c.QueryArray(key)
			if len(rawSlice) == 1 && strings.Contains(rawSlice[0], ",") {
				rawSlice = strings.Split(rawSlice[0], ",")
			}
		} else {
			raw = c.Query(key)
		}
	}

	// Cookie
	if key := sf.Tag.Get(tagCookie); key != "" {
		if cookie, err := c.Cookie(key); err == nil {
			raw = cookie
		}
	}

	// Path / Param
	if key := sf.Tag.Get(tagPath); key != "" {
		raw = c.Param(key)
	}
	if key := sf.Tag.Get(tagParam); key != "" {
		raw = c.Param(key)
	}

	// Body binding (special case)
	if sf.Tag.Get(tagJSON) == bodyValue || sf.Name == bodyField {
		bodyPtr := reflect.New(sf.Type)
		if err := c.Bind(bodyPtr.Interface()); err != nil {
			return fmt.Errorf("failed to bind body: %w", err)
		}
		field.Set(bodyPtr.Elem())

		// Validate nested struct fields
		if err := c.validateStruct(bodyPtr.Elem(), sf); err != nil {
			return err
		}
		return nil
	}

	// Default values
	if raw == "" && len(rawSlice) == 0 {
		if def := sf.Tag.Get(tagDefault); def != "" {
			if field.Kind() == reflect.Slice {
				rawSlice = strings.Split(def, ",")
			} else {
				raw = def
			}
		}
	}

	// Set field value
	if field.CanSet() {
		if field.Kind() == reflect.Slice && len(rawSlice) > 0 {
			if err := setSliceWithType(field, rawSlice); err != nil {
				return fmt.Errorf("cannot set field %s: %w", sf.Name, err)
			}
		} else if raw != "" {
			if err := setWithType(field, raw); err != nil {
				return fmt.Errorf("cannot set field %s: %w", sf.Name, err)
			}
		}
	}

	return nil
}

// validateField performs tag-based validations: required, min/max, length constraints.
func (c *Context) validateField(field reflect.Value, sf reflect.StructField) error {
	// Required
	if sf.Tag.Get(tagRequired) == TRUE && isEmptyValue(field) {
		return fmt.Errorf("field %s is required", sf.Name)
	}

	// Numeric min/max
	if minTag := sf.Tag.Get(tagMin); minTag != "" {
		if err := checkMin(field, minTag); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}
	if maxTag := sf.Tag.Get(tagMax); maxTag != "" {
		if err := checkMax(field, maxTag); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}

	// String length validation
	if minLen := sf.Tag.Get(tagMinLength); minLen != "" {
		if err := checkMinLength(field, minLen); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}
	if maxLen := sf.Tag.Get(tagMaxLength); maxLen != "" {
		if err := checkMaxLength(field, maxLen); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}

	return nil
}

// validateStruct validates nested struct fields using their struct tags
func (c *Context) validateStruct(v reflect.Value, parentField reflect.StructField) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		// Required validation
		if sf.Tag.Get(tagRequired) == TRUE && isEmptyValue(field) {
			return fmt.Errorf("field %s.%s is required", parentField.Name, sf.Name)
		}

		// Numeric min/max
		if minTag := sf.Tag.Get(tagMin); minTag != "" {
			if err := checkMin(field, minTag); err != nil {
				return fmt.Errorf("field %s.%s: %w", parentField.Name, sf.Name, err)
			}
		}
		if maxTag := sf.Tag.Get(tagMax); maxTag != "" {
			if err := checkMax(field, maxTag); err != nil {
				return fmt.Errorf("field %s.%s: %w", parentField.Name, sf.Name, err)
			}
		}

		// String minLength/maxLength
		if minLenTag := sf.Tag.Get(tagMinLength); minLenTag != "" {
			if err := checkMinLength(field, minLenTag); err != nil {
				return fmt.Errorf("field %s.%s: %w", parentField.Name, sf.Name, err)
			}
		}
		if maxLenTag := sf.Tag.Get(tagMaxLength); maxLenTag != "" {
			if err := checkMaxLength(field, maxLenTag); err != nil {
				return fmt.Errorf("field %s.%s: %w", parentField.Name, sf.Name, err)
			}
		}
	}

	return nil
}

func setWithType(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value '%s': %w", raw, err)
		}
		if field.OverflowInt(i) {
			return fmt.Errorf("integer value '%s' overflows %s", raw, field.Type())
		}
		field.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value '%s': %w", raw, err)
		}
		if field.OverflowUint(u) {
			return fmt.Errorf("unsigned integer value '%s' overflows %s", raw, field.Type())
		}
		field.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("invalid float value '%s': %w", raw, err)
		}
		if field.OverflowFloat(f) {
			return fmt.Errorf("float value '%s' overflows %s", raw, field.Type())
		}
		field.SetFloat(f)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid boolean value '%s': %w", raw, err)
		}
		field.SetBool(b)
		return nil
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setWithType(field.Elem(), raw)
	case reflect.Slice:
		// Handle comma-separated values for slices
		values := strings.Split(raw, ",")
		return setSliceWithType(field, values)
	default:
		return fmt.Errorf("unsupported field type %s", field.Kind())
	}
}

func setSliceWithType(field reflect.Value, rawSlice []string) error {
	elemType := field.Type().Elem()
	slice := reflect.MakeSlice(field.Type(), len(rawSlice), len(rawSlice))

	for i, raw := range rawSlice {
		elem := slice.Index(i)
		switch elemType.Kind() {
		case reflect.String:
			elem.SetString(strings.TrimSpace(raw))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val, err := strconv.Atoi(strings.TrimSpace(raw))
			if err != nil {
				return fmt.Errorf("invalid integer value '%s': %w", raw, err)
			}
			elem.SetInt(int64(val))
		case reflect.Bool:
			val, err := strconv.ParseBool(strings.TrimSpace(raw))
			if err != nil {
				return fmt.Errorf("invalid boolean value '%s': %w", raw, err)
			}
			elem.SetBool(val)
		default:
			return fmt.Errorf("unsupported slice element type: %s", elemType.Kind())
		}
	}

	field.Set(slice)
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	return v.IsZero()
}

func checkMin(field reflect.Value, minTag string) error {
	minValue, err := strconv.Atoi(minTag)
	if err != nil {
		return fmt.Errorf("invalid min value: %s", minTag)
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() < int64(minValue) {
			return fmt.Errorf("value %d must be >= %d", field.Int(), minValue)
		}
	}
	return nil
}

func checkMax(field reflect.Value, maxTag string) error {
	maxValue, err := strconv.Atoi(maxTag)
	if err != nil {
		return fmt.Errorf("invalid max value: %s", maxTag)
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Int() > int64(maxValue) {
			return fmt.Errorf("value %d must be <= %d", field.Int(), maxValue)
		}
	}
	return nil
}

func checkMinLength(field reflect.Value, minTag string) error {
	minValue, err := strconv.Atoi(minTag)
	if err != nil {
		return fmt.Errorf("invalid minLength value: %s", minTag)
	}

	if field.Kind() == reflect.String {
		if len(field.String()) < minValue {
			return fmt.Errorf("string length %d must be at least %d characters", len(field.String()), minValue)
		}
	}
	return nil
}

func checkMaxLength(field reflect.Value, maxTag string) error {
	maxValue, err := strconv.Atoi(maxTag)
	if err != nil {
		return fmt.Errorf("invalid maxLength value: %s", maxTag)
	}

	if field.Kind() == reflect.String {
		if len(field.String()) > maxValue {
			return fmt.Errorf("string length %d must be at most %d characters", len(field.String()), maxValue)
		}
	}
	return nil
}
