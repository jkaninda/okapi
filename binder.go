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
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

func (c *Context) ShouldBind(v any) (bool, error) {
	if err := c.Bind(v); err != nil {
		return false, err
	}
	return true, nil
}

// B is a shortcut for Bind, allowing you to bind request data to a struct.
func (c *Context) B(v any) error {
	if err := c.Bind(v); err != nil {
		return fmt.Errorf("binding error: %w", err)
	}
	return nil
}

// Bind binds the request data to the provided struct based on the content type and tags.
func (c *Context) Bind(out any) error {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("bind target must be a non-nil pointer to a struct")
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return errors.New("bind target must be a pointer to a struct")
	}

	// Decode body content based on content type (if any)
	switch contentType := c.ContentType(); {
	case strings.Contains(contentType, JSON):
		_ = c.BindJSON(out) // ignore error for now
	case strings.Contains(contentType, XML):
		_ = c.BindXML(out)
	case strings.Contains(contentType, YAML),
		strings.Contains(contentType, YamlX),
		strings.Contains(contentType, YamlText):
		_ = c.BindYAML(out)
	case strings.Contains(contentType, PROTOBUF):
		if msg, ok := out.(proto.Message); ok {
			_ = c.BindProtoBuf(msg)
		}
	case strings.Contains(contentType, FormData):
		// Handle multipart form data specially
		return c.BindMultipart(out)
	}

	// Overlay additional values from param, query, and form
	if err := c.bindFromFields(out); err != nil {
		return err
	}

	// Final validation
	return validateStruct(out)
}

func (c *Context) BindMultipart(out any) error {
	if err := c.Request.ParseMultipartForm(c.okapi.maxMultipartMemory); err != nil {
		return fmt.Errorf("invalid multipart form: %w", err)
	}

	v := reflect.ValueOf(out).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		valField := v.Field(i)

		if !valField.CanSet() {
			continue
		}

		if err := c.bindMultipartField(field, valField); err != nil {
			return fmt.Errorf("bind error for field %s: %w", field.Name, err)
		}
	}

	return validateStruct(out)
}

func (c *Context) bindMultipartField(field reflect.StructField, valField reflect.Value) error {
	var wasSet bool
	var err error

	// Handle file uploads (legacy form-file tag)
	if formFileTag := field.Tag.Get("form-file"); formFileTag != "" {
		wasSet, err = c.bindFileFieldWithStatus(formFileTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Handle headers
	if headerTag := field.Tag.Get("header"); headerTag != "" {
		wasSet, err = c.bindHeaderFieldWithStatus(headerTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Handle form values (including files and arrays)
	if formTag := field.Tag.Get("form"); formTag != "" {
		// Check if this is a file field based on type
		if c.isFileField(valField) {
			wasSet, err = c.bindFileFieldWithStatus(formTag, valField, field)
			if err != nil {
				return err
			}
			if wasSet {
				return nil
			}
		} else {
			wasSet, err = c.bindFormFieldWithStatus(formTag, valField, field)
			if err != nil {
				return err
			}
			if wasSet {
				return nil
			}
		}
	}

	// Handle query parameters (including arrays)
	if queryTag := field.Tag.Get("query"); queryTag != "" {
		wasSet, err = c.bindQueryFieldWithStatus(queryTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Handle path parameters
	if paramTag := field.Tag.Get("param"); paramTag != "" {
		wasSet, err = c.bindParamFieldWithStatus(paramTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Apply default values if field is empty and check required only if no value was set
	return c.applyDefaultAndValidate(valField, field, wasSet)
}

// Helper function to determine if a field should be treated as a file field
func (c *Context) isFileField(valField reflect.Value) bool {
	fieldType := valField.Type()

	// Check for *multipart.FileHeader
	if fieldType == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		return true
	}

	// Check for multipart.File interface
	if fieldType.AssignableTo(reflect.TypeOf((*multipart.File)(nil)).Elem()) {
		return true
	}

	// Check for []*multipart.FileHeader (multiple files)
	if fieldType.Kind() == reflect.Slice {
		elemType := fieldType.Elem()
		if elemType == reflect.TypeOf((*multipart.FileHeader)(nil)) {
			return true
		}
	}

	return false
}

func (c *Context) bindFileFieldWithStatus(tag string, valField reflect.Value, field reflect.StructField) (bool, error) {
	// Handle multiple files ([]*multipart.FileHeader)
	if valField.Kind() == reflect.Slice && valField.Type().Elem() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		return c.bindMultipleFilesWithStatus(tag, valField)
	}

	// Handle single file
	file, header, err := c.Request.FormFile(tag)
	if err != nil {
		// File not found or error - return false to indicate no value was set
		return false, nil
	}
	defer func(file multipart.File) {
		err = file.Close()
		if err != nil {
			_, err = fmt.Fprintf(defaultErrorWriter, "Failed to close response body")
			if err != nil {
				return
			}
		}
	}(file)

	// Handle *multipart.FileHeader type
	if valField.Type() == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		valField.Set(reflect.ValueOf(header))
		return true, nil
	}

	// Handle multipart.File type
	if valField.Type().AssignableTo(reflect.TypeOf((*multipart.File)(nil)).Elem()) {
		valField.Set(reflect.ValueOf(file))
		return true, nil
	}

	return false, fmt.Errorf("unsupported file field type %s for field %s", valField.Type(), field.Name)
}

func (c *Context) bindMultipleFilesWithStatus(tag string, valField reflect.Value) (bool, error) {
	// Get the multipart form
	if c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(c.okapi.maxMultipartMemory); err != nil {
			return false, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	}

	fileHeaders := c.Request.MultipartForm.File[tag]
	if len(fileHeaders) == 0 {
		// No files found - return false to indicate no value was set
		return false, nil
	}

	// Create slice of file headers
	slice := reflect.MakeSlice(valField.Type(), len(fileHeaders), len(fileHeaders))
	for i, header := range fileHeaders {
		slice.Index(i).Set(reflect.ValueOf(header))
	}
	valField.Set(slice)

	return true, nil
}

func (c *Context) bindHeaderFieldWithStatus(tag string, v reflect.Value, fld reflect.StructField) (bool, error) {
	headerValue := c.Request.Header.Get(tag)
	if headerValue == "" {
		// No header value found - return false to indicate no value was set
		return false, nil
	}

	err := setValueWithValidation(v, headerValue, fld)
	return true, err
}

func (c *Context) bindFormFieldWithStatus(tag string, valField reflect.Value, field reflect.StructField) (bool, error) {
	// Handle slice types (arrays)
	if valField.Kind() == reflect.Slice && valField.Type().Elem().Kind() == reflect.String {
		values := c.Request.MultipartForm.Value[tag]
		if len(values) == 0 {
			// No form values found - return false to indicate no value was set
			return false, nil
		}

		// Handle comma-separated values in a single parameter (like ?tags=a,b)
		var allValues []string
		for _, value := range values {
			if strings.Contains(value, ",") {
				allValues = append(allValues, strings.Split(value, ",")...)
			} else {
				allValues = append(allValues, value)
			}
		}

		// Trim whitespace from each value
		for i, val := range allValues {
			allValues[i] = strings.TrimSpace(val)
		}

		slice := reflect.MakeSlice(valField.Type(), len(allValues), len(allValues))
		for i, val := range allValues {
			slice.Index(i).SetString(val)
		}
		valField.Set(slice)
		return true, nil
	}

	// Handle single values
	values := c.Request.MultipartForm.Value[tag]
	if len(values) == 0 {
		// No form values found - return false to indicate no value was set
		return false, nil
	}

	err := setValueWithValidation(valField, values[0], field)
	return true, err
}

func (c *Context) bindQueryFieldWithStatus(tag string, vf reflect.Value, fld reflect.StructField) (bool, error) {
	// Parse query parameters if not already parsed
	if c.Request.Form == nil {
		if err := c.Request.ParseForm(); err != nil {
			return false, fmt.Errorf("failed to parse query parameters: %w", err)
		}
	}

	// Handle slice types (arrays)
	if vf.Kind() == reflect.Slice && vf.Type().Elem().Kind() == reflect.String {
		values := c.Request.Form[tag]
		if len(values) == 0 {
			// No query values found - return false to indicate no value was set
			return false, nil
		}

		// Handle comma-separated values
		var allValues []string
		for _, value := range values {
			if strings.Contains(value, ",") {
				allValues = append(allValues, strings.Split(value, ",")...)
			} else {
				allValues = append(allValues, value)
			}
		}

		// Trim whitespace
		for i, val := range allValues {
			allValues[i] = strings.TrimSpace(val)
		}

		slice := reflect.MakeSlice(vf.Type(), len(allValues), len(allValues))
		for i, val := range allValues {
			slice.Index(i).SetString(val)
		}
		vf.Set(slice)
		return true, nil
	}

	// Handle single values
	value := c.Request.FormValue(tag)
	if value == "" {
		// No query value found - return false to indicate no value was set
		return false, nil
	}

	err := setValueWithValidation(vf, value, fld)
	return true, err
}

func (c *Context) bindParamFieldWithStatus(tag string, vf reflect.Value, fld reflect.StructField) (bool, error) {
	value := c.Param(tag)
	if value == "" {
		// No param value found - return false to indicate no value was set
		return false, nil
	}

	err := setValueWithValidation(vf, value, fld)
	return true, err
}

func (c *Context) applyDefaultAndValidate(valField reflect.Value, field reflect.StructField, wasSet bool) error {
	// Only apply default if no value was set and field is currently zero
	if !wasSet && isZero(valField) {
		if def := field.Tag.Get("default"); def != "" {
			return setValueWithValidation(valField, def, field)
		}
	}

	// Only check required if no value was set and field is still zero after potential default application
	if !wasSet && field.Tag.Get("required") == TRUE && isZero(valField) {
		return fmt.Errorf("field %s is required", field.Name)
	}

	return nil
}

// Updated bindFromFields to handle the new field types
func (c *Context) bindFromFields(out any) error {
	v := reflect.ValueOf(out).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		valField := v.Field(i)

		if !valField.CanSet() {
			continue
		}

		// Skip if this is a multipart form - already handled
		if strings.Contains(c.ContentType(), FormData) {
			continue
		}

		wasSet := false
		var err error

		// Try to get value from different sources
		if tag := field.Tag.Get("param"); tag != "" {
			if value := c.Param(tag); value != "" {
				err = setValueWithValidation(valField, value, field)
				if err != nil {
					return fmt.Errorf("bind error for field %s: %w", field.Name, err)
				}
				wasSet = true
			}
		}

		if !wasSet {
			if tag := field.Tag.Get("query"); tag != "" {
				if value := c.Query(tag); value != "" {
					err = setValueWithValidation(valField, value, field)
					if err != nil {
						return fmt.Errorf("bind error for field %s: %w", field.Name, err)
					}
					wasSet = true
				}
			}
		}

		if !wasSet {
			if tag := field.Tag.Get("form"); tag != "" {
				if value := c.FormValue(tag); value != "" {
					err = setValueWithValidation(valField, value, field)
					if err != nil {
						return fmt.Errorf("bind error for field %s: %w", field.Name, err)
					}
					wasSet = true
				}
			}
		}

		if !wasSet {
			if tag := field.Tag.Get("header"); tag != "" {
				if value := c.Request.Header.Get(tag); value != "" {
					err = setValueWithValidation(valField, value, field)
					if err != nil {
						return fmt.Errorf("bind error for field %s: %w", field.Name, err)
					}
					wasSet = true
				}
			}
		}

		// Apply defaults and validate only if no value was set
		if !wasSet {
			if def := field.Tag.Get("default"); def != "" && isZero(valField) {
				err = setValueWithValidation(valField, def, field)
				if err != nil {
					return fmt.Errorf("bind error for field %s: %w", field.Name, err)
				}
				wasSet = true
			}
		}

		// Check required only if no value was set and field is still zero
		if !wasSet && field.Tag.Get("required") == TRUE && isZero(valField) {
			return fmt.Errorf("field %s is required", field.Name)
		}
	}

	return nil
}

func setValueWithValidation(field reflect.Value, value string, sf reflect.StructField) error {
	switch field.Kind() {
	case reflect.String:
		return setStringValue(field, value, sf)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setIntValue(field, value, sf)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return setUintValue(field, value, sf)

	case reflect.Float32, reflect.Float64:
		return setFloatValue(field, value, sf)

	case reflect.Bool:
		return setBoolValue(field, value)

	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setValueWithValidation(field.Elem(), value, sf)

	case reflect.Slice:
		return setSliceValue(field, value)

	default:
		return fmt.Errorf("unsupported field type %s", field.Kind())
	}
}

func setStringValue(field reflect.Value, value string, sf reflect.StructField) error {
	if err := checkStringLength(value, sf); err != nil {
		return err
	}
	field.SetString(value)
	return nil
}

func setIntValue(field reflect.Value, value string, sf reflect.StructField) error {
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	if err = validateMinMaxInt(i, sf); err != nil {
		return err
	}
	field.SetInt(i)
	return nil
}

func validateMinMaxInt(i int64, sf reflect.StructField) error {
	if minStr := sf.Tag.Get("min"); minStr != "" {
		if m, err := strconv.ParseInt(minStr, 10, 64); err == nil && i < m {
			return fmt.Errorf("value must be at least %d", m)
		}
	}
	if maxStr := sf.Tag.Get("max"); maxStr != "" {
		if mx, err := strconv.ParseInt(maxStr, 10, 64); err == nil && i > mx {
			return fmt.Errorf("value must be at most %d", mx)
		}
	}
	return nil
}

func setUintValue(field reflect.Value, value string, sf reflect.StructField) error {
	u, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return err
	}
	if err = validateMinMaxUint(u, sf); err != nil {
		return err
	}
	field.SetUint(u)
	return nil
}

func validateMinMaxUint(u uint64, sf reflect.StructField) error {
	if minStr := sf.Tag.Get("min"); minStr != "" {
		if m, err := strconv.ParseUint(minStr, 10, 64); err == nil && u < m {
			return fmt.Errorf("value must be at least %d", m)
		}
	}
	if maxStr := sf.Tag.Get("max"); maxStr != "" {
		if mx, err := strconv.ParseUint(maxStr, 10, 64); err == nil && u > mx {
			return fmt.Errorf("value must be at most %d", mx)
		}
	}
	return nil
}

func setFloatValue(field reflect.Value, value string, sf reflect.StructField) error {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	if err = validateMinMaxFloat(f, sf); err != nil {
		return err
	}
	field.SetFloat(f)
	return nil
}

func validateMinMaxFloat(f float64, sf reflect.StructField) error {
	if minStr := sf.Tag.Get("min"); minStr != "" {
		if m, err := strconv.ParseFloat(minStr, 64); err == nil && f < m {
			return fmt.Errorf("value must be at least %f", m)
		}
	}
	if maxStr := sf.Tag.Get("max"); maxStr != "" {
		if mx, err := strconv.ParseFloat(maxStr, 64); err == nil && f > mx {
			return fmt.Errorf("value must be at most %f", mx)
		}
	}
	return nil
}

func setBoolValue(field reflect.Value, value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	field.SetBool(b)
	return nil
}

func setSliceValue(field reflect.Value, value string) error {
	if field.Type().Elem().Kind() == reflect.String {
		values := strings.Split(value, ",")
		slice := reflect.MakeSlice(field.Type(), len(values), len(values))
		for i, val := range values {
			slice.Index(i).SetString(strings.TrimSpace(val))
		}
		field.Set(slice)
		return nil
	}
	return fmt.Errorf("unsupported slice type %s", field.Type().Elem().Kind())
}

func checkStringLength(s string, sf reflect.StructField) error {
	if minStr := sf.Tag.Get("min"); minStr != "" {
		if m, err := strconv.Atoi(minStr); err == nil && len(s) < m {
			return fmt.Errorf("string too short: length %d < min %d", len(s), m)
		}
	}
	if maxStr := sf.Tag.Get("max"); maxStr != "" {
		if mx, err := strconv.Atoi(maxStr); err == nil && len(s) > mx {
			return fmt.Errorf("string too long: length %d > max %d", len(s), mx)
		}
	}
	return nil
}

func isZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	default:
		zero := reflect.Zero(v.Type()).Interface()
		current := v.Interface()
		return reflect.DeepEqual(current, zero)
	}
}

func (c *Context) BindJSON(v any) error {
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) BindXML(v any) error {
	return xml.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) BindYAML(v any) error {
	return yaml.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) BindProtoBuf(v proto.Message) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			_, err = fmt.Fprintf(defaultErrorWriter, "Failed to close response body")
			if err != nil {
				return
			}
		}
	}(c.Request.Body)
	return proto.Unmarshal(body, v)
}

func (c *Context) BindQuery(v any) error {
	if err := c.Request.ParseForm(); err != nil {
		return fmt.Errorf("invalid query data: %w", err)
	}
	return formToStruct(c.Request.Form, v)
}

func (c *Context) BindForm(v any) error {
	if err := c.Request.ParseForm(); err != nil {
		return fmt.Errorf("invalid form data: %w", err)
	}
	return formToStruct(c.Request.Form, v)
}

func formToStruct(data url.Values, v any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal form data: %w", err)
	}
	if err := json.Unmarshal(jsonData, v); err != nil {
		return fmt.Errorf("failed to unmarshal form data: %w", err)
	}
	return validateStruct(v)
}

func validateStruct(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		sf := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		if field.Kind() == reflect.Struct {
			if err := validateStruct(field.Addr().Interface()); err != nil {
				return err
			}
		}
		if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Struct {
			for j := 0; j < field.Len(); j++ {
				if err := validateStruct(field.Index(j).Addr().Interface()); err != nil {
					return err
				}
			}
		}
		if sf.Tag.Get("required") == TRUE && isZero(field) {
			return fmt.Errorf("field %s is required", sf.Name)
		}
	}
	return nil
}
