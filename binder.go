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
	"strings"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// ShouldBind is a convenience method that binds request data to a struct and returns a boolean indicating success.
func (c *Context) ShouldBind(v any) (bool, error) {
	if err := c.bindRequest(v); err != nil {
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

// Bind populates the given struct with request data by inspecting tags and content type.
// It supports two binding styles:
//
//  1. **Flat binding (legacy style)**
//     Request body fields (JSON, XML, YAML, Protobuf, Form) can be mixed directly with
//     query parameters, headers, cookies, and path params in a single struct.
//
//  2. **Body field binding (recommended style)**
//     A struct may contain a dedicated `Body` field (or tagged as `body`) that represents
//     the request payload, while sibling fields represent query params, headers, cookies,
//     and path params. This style enforces a clear separation between metadata and body.
//
// Validation tags such as `required`, `min`, `max`, `minLength`, and `maxLength` are supported,
// along with descriptive metadata (`description`) that can be used for documentation.
//
// Example (Body field binding):
//
//	type BookInput struct {
//	  // Query parameter
//	  Tags []string `query:"tags" description:"List of book tags"`
//
//	  // Header parameter
//	  Accept string `header:"Accept" required:"true" description:"Accept header"`
//
//	  // Cookie parameter
//	  SessionID string `cookie:"SessionID" required:"true" description:"Session ID cookie"`
//	  // Path parameter
//	  BookID string `path:"bookId" required:"true" description:"Book ID"`
//
//	  // Request body
//	  Body struct {
//	    Name  string `json:"name" required:"true" minLength:"2" maxLength:"100" description:"Book name"`
//	    Price int    `json:"price" required:"true" min:"5" max:"100" yaml:"price" description:"Book price"`
//	  }
//	}
//
//	okapi.Put("/books/:bookId", func(c okapi.Context) error {
//	  book := &BookInput{}
//	  if err := c.Bind(book); err != nil {
//	    return c.AbortBadRequest("Invalid input", err)
//	  }
//	  return c.Respond(book)
//	})
func (c *Context) Bind(out any) error {
	if hasBodyField(out) {
		return c.bindStruct(out)
	}
	return c.bindRequest(out)
}

// Bind binds the request data to the provided struct based on the content type and tags.
func (c *Context) bindRequest(out any) error {
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
	case strings.Contains(contentType, constJSON):
		_ = c.BindJSON(out) // ignore error for now
	case strings.Contains(contentType, constXML):
		_ = c.BindXML(out)
	case strings.Contains(contentType, constYAML),
		strings.Contains(contentType, constYamlX),
		strings.Contains(contentType, constYamlText):
		_ = c.BindYAML(out)
	case strings.Contains(contentType, constPROTOBUF):
		if msg, ok := out.(proto.Message); ok {
			_ = c.BindProtoBuf(msg)
		}
	case strings.Contains(contentType, constFormData):
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

// BindMultipart binds multipart form data to the provided struct.
func (c *Context) BindMultipart(out any) error {
	if err := c.request.ParseMultipartForm(c.okapi.maxMultipartMemory); err != nil {
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

	// Handle headers
	if headerTag := field.Tag.Get(tagHeader); headerTag != "" {
		wasSet, err = c.bindHeaderFieldWithStatus(headerTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Handle form values (including files and arrays)
	if formTag := field.Tag.Get(tagForm); formTag != "" {
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
	if queryTag := field.Tag.Get(tagQuery); queryTag != "" {
		wasSet, err = c.bindQueryFieldWithStatus(queryTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}

	// Handle path parameters
	if paramTag := field.Tag.Get(tagParam); paramTag != "" {
		wasSet, err = c.bindParamFieldWithStatus(paramTag, valField, field)
		if err != nil {
			return err
		}
		if wasSet {
			return nil
		}
	}
	if paramTag := field.Tag.Get(tagPath); paramTag != "" {
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
	file, header, err := c.request.FormFile(tag)
	if err != nil {
		// File not found or error - return false to indicate no value was set
		return false, nil
	}
	defer func(file multipart.File) {
		err = file.Close()
		if err != nil {
			fPrintError("Failed to close response body")
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
	if c.request.MultipartForm == nil {
		if err := c.request.ParseMultipartForm(c.okapi.maxMultipartMemory); err != nil {
			return false, fmt.Errorf("failed to parse multipart form: %w", err)
		}
	}

	fileHeaders := c.request.MultipartForm.File[tag]
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
	headerValue := c.request.Header.Get(tag)
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
		values := c.request.MultipartForm.Value[tag]
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
	values := c.request.MultipartForm.Value[tag]
	if len(values) == 0 {
		// No form values found - return false to indicate no value was set
		return false, nil
	}

	err := setValueWithValidation(valField, values[0], field)
	return true, err
}

func (c *Context) bindQueryFieldWithStatus(tag string, vf reflect.Value, fld reflect.StructField) (bool, error) {
	// Parse query parameters if not already parsed
	if c.request.Form == nil {
		if err := c.request.ParseForm(); err != nil {
			return false, fmt.Errorf("failed to parse query parameters: %w", err)
		}
	}

	// Handle slice types (arrays)
	if vf.Kind() == reflect.Slice && vf.Type().Elem().Kind() == reflect.String {
		values := c.request.Form[tag]
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
	value := c.request.FormValue(tag)
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
	if !wasSet && isEmptyValue(valField) {
		if def := field.Tag.Get(tagDefault); def != "" {
			return setValueWithValidation(valField, def, field)
		}
	}

	// Only check required if no value was set and field is still zero after potential default application
	if !wasSet && field.Tag.Get(tagRequired) == constTRUE && isEmptyValue(valField) {
		return fmt.Errorf("field %s is required", field.Name)
	}

	return nil
}

// Updated bindFromFields to handle the new field types
func (c *Context) bindFromFields(out any) error {
	v := reflect.ValueOf(out).Elem()
	t := v.Type()

	// Helper to try to set a field from a value source
	trySet := func(valField reflect.Value, value string, field reflect.StructField) (bool, error) {
		if value == "" {
			return false, nil
		}
		if err := setValueWithValidation(valField, value, field); err != nil {
			return false, fmt.Errorf("bind error for field %s: %w", field.Name, err)
		}
		return true, nil
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		valField := v.Field(i)

		if !valField.CanSet() {
			continue
		}

		// Skip multipart forms
		if strings.Contains(c.ContentType(), constFormData) {
			continue
		}

		wasSet := false

		// Map of tag type → function returning value
		tagSources := map[string]func(string) string{
			tagParam:  c.Param,
			tagPath:   c.Param,
			tagQuery:  c.Query,
			tagForm:   c.FormValue,
			tagHeader: func(key string) string { return c.request.Header.Get(key) },
		}

		// Try each tag source
		for tag, getter := range tagSources {
			if tagVal := field.Tag.Get(tag); tagVal != "" {
				set, err := trySet(valField, getter(tagVal), field)
				if err != nil {
					return err
				}
				if set {
					wasSet = true
					break
				}
			}
		}

		// Cookie is special, since it returns error
		if !wasSet {
			if key := field.Tag.Get(tagCookie); key != "" {
				if value, err := c.Cookie(key); err == nil {
					set, err := trySet(valField, value, field)
					if err != nil {
						return err
					}
					if set {
						wasSet = true
					}
				}
			}
		}

		// Default value
		if !wasSet {
			if def := field.Tag.Get(tagDefault); def != "" && isEmptyValue(valField) {
				if _, err := trySet(valField, def, field); err != nil {
					return err
				}
				wasSet = true
			}
		}

		// Required check
		if !wasSet && field.Tag.Get(tagRequired) == constTRUE && isEmptyValue(valField) {
			return fmt.Errorf("field %s is required", field.Name)
		}
	}

	return nil
}

func setValueWithValidation(field reflect.Value, value string, sf reflect.StructField) error {
	if field.CanSet() {
		if value != "" {
			if err := setWithType(field, value); err != nil {
				return fmt.Errorf("cannot set field %s: %w", sf.Name, err)
			}
			return nil
		}
	}
	return fmt.Errorf("unsupported field type %s", field.Kind())
}

func (c *Context) BindJSON(v any) error {
	return json.NewDecoder(c.request.Body).Decode(v)
}

func (c *Context) BindXML(v any) error {
	return xml.NewDecoder(c.request.Body).Decode(v)
}

func (c *Context) BindYAML(v any) error {
	return yaml.NewDecoder(c.request.Body).Decode(v)
}

func (c *Context) BindProtoBuf(v proto.Message) error {
	body, err := io.ReadAll(c.request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fPrintError("Failed to close response body")
		}
	}(c.request.Body)
	return proto.Unmarshal(body, v)
}

func (c *Context) BindQuery(v any) error {
	if err := c.request.ParseForm(); err != nil {
		return fmt.Errorf("invalid query data: %w", err)
	}
	return formToStruct(c.request.Form, v)
}

func (c *Context) BindForm(v any) error {
	if err := c.request.ParseForm(); err != nil {
		return fmt.Errorf("invalid form data: %w", err)
	}
	return formToStruct(c.request.Form, v)
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
		// Required validation
		if sf.Tag.Get(tagRequired) == "true" && isEmptyValue(field) {
			return fmt.Errorf("field %s is required", sf.Name)
		}
		// Min/Max validation
		if err := validateMinMax(field, sf); err != nil {
			return err
		}
		// Enum validation
		if enumTag := sf.Tag.Get(tagEnum); enumTag != "" {
			if err := checkEnum(field, enumTag); err != nil {
				return fmt.Errorf("field %s.%s: %w", sf.Name, sf.Name, err)
			}
		}
		// MultipleOf validation
		if multipleOfTag := sf.Tag.Get(tagMultipleOf); multipleOfTag != "" {
			if err := checkMultipleOf(field, multipleOfTag); err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
		}

		// Format validation
		if formatTag := sf.Tag.Get(tagFormat); formatTag != "" {
			if err := checkFormat(field, formatTag, sf); err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
		}
		// Pattern validation
		if patternTag := sf.Tag.Get(tagPattern); patternTag != "" {
			if err := checkPattern(field, patternTag); err != nil {
				return fmt.Errorf("field %s: %w", sf.Name, err)
			}
		}
		// Slice validations
		if field.Kind() == reflect.Slice {
			if err := validateSlice(field, sf); err != nil {
				return err
			}
		}

	}
	return nil
}

// validateMinMax checks min/max and minLength/maxLength constraints
func validateMinMax(field reflect.Value, sf reflect.StructField) error {
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

	// String minLength/maxLength
	if minLenTag := sf.Tag.Get(tagMinLength); minLenTag != "" {
		if err := checkMinLength(field, minLenTag); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}
	if maxLenTag := sf.Tag.Get(tagMaxLength); maxLenTag != "" {
		if err := checkMaxLength(field, maxLenTag); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}
	return nil
}

// validateSlice maxItems and minItems
func validateSlice(field reflect.Value, sf reflect.StructField) error {
	// Slice minItems/maxItems
	if minItemsTag := sf.Tag.Get(tagMinItems); minItemsTag != "" {
		if err := checkMinItems(field, minItemsTag); err != nil {
			return fmt.Errorf("field %s: %v", sf.Name, err)
		}
	}
	if maxItemsTag := sf.Tag.Get(tagMaxItems); maxItemsTag != "" {
		if err := checkMaxItems(field, maxItemsTag); err != nil {
			return fmt.Errorf("field %s: %v", sf.Name, err)
		}
	}
	// UniqueItems validation
	if sf.Tag.Get(tagUniqueItems) == constTRUE {
		if err := checkUniqueItems(field); err != nil {
			return fmt.Errorf("field %s: %v", sf.Name, err)
		}
	}
	return nil
}
