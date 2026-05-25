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
	"encoding/base64"
	"fmt"
	"math"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Precompiled patterns used by format validators.
var (
	semverRegex      = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	ulidRegex        = regexp.MustCompile(`(?i)^[0-7][0-9A-HJKMNP-TV-Z]{25}$`)
	e164Regex        = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
	alphaRegex       = regexp.MustCompile(`^[a-zA-Z]+$`)
	alphanumRegex    = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	numericRegex     = regexp.MustCompile(`^[-+]?[0-9]+(?:\.[0-9]+)?$`)
	slugRegex        = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hexColorRegex    = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	jsonPointerRegex = regexp.MustCompile(`^(?:/(?:[^/~]|~0|~1)*)*$`)
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

// fieldConstraintCheckers is the ordered set of grouped tag checks applied to a
// single field. Each returns an unprefixed error; callers add the field label.
// Shared by validateField, validateStruct, and the binder's validateStruct.
var fieldConstraintCheckers = []func(reflect.Value, reflect.StructField) error{
	checkNumericConstraints,
	checkLengthConstraints,
	checkChoiceConstraints,
	checkFormatConstraints,
	checkCollectionConstraints,
}

// checkNumericConstraints validates min, max, exclusiveMin, exclusiveMax, and multipleOf.
func checkNumericConstraints(field reflect.Value, sf reflect.StructField) error {
	if tag := sf.Tag.Get(tagMin); tag != "" {
		if err := checkMin(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagMax); tag != "" {
		if err := checkMax(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagExclusiveMin); tag != "" {
		if err := checkExclusiveMin(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagExclusiveMax); tag != "" {
		if err := checkExclusiveMax(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagMultipleOf); tag != "" {
		if err := checkMultipleOf(field, tag); err != nil {
			return err
		}
	}
	return nil
}

// checkLengthConstraints validates minLength and maxLength.
func checkLengthConstraints(field reflect.Value, sf reflect.StructField) error {
	if tag := sf.Tag.Get(tagMinLength); tag != "" {
		if err := checkMinLength(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagMaxLength); tag != "" {
		if err := checkMaxLength(field, tag); err != nil {
			return err
		}
	}
	return nil
}

// checkChoiceConstraints validates enum and const.
func checkChoiceConstraints(field reflect.Value, sf reflect.StructField) error {
	if tag := sf.Tag.Get(tagEnum); tag != "" {
		if err := checkEnum(field, tag); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagConst); tag != "" {
		if err := checkConst(field, tag); err != nil {
			return err
		}
	}
	return nil
}

// checkFormatConstraints validates format and pattern (both handle slices element-wise).
func checkFormatConstraints(field reflect.Value, sf reflect.StructField) error {
	if tag := sf.Tag.Get(tagFormat); tag != "" {
		if err := checkFormat(field, tag, sf); err != nil {
			return err
		}
	}
	if tag := sf.Tag.Get(tagPattern); tag != "" {
		if err := checkPattern(field, tag); err != nil {
			return err
		}
	}
	return nil
}

// checkCollectionConstraints validates slice item counts/uniqueness and map property counts.
func checkCollectionConstraints(field reflect.Value, sf reflect.StructField) error {
	switch field.Kind() {
	case reflect.Slice:
		if tag := sf.Tag.Get(tagMinItems); tag != "" {
			if err := checkMinItems(field, tag); err != nil {
				return err
			}
		}
		if tag := sf.Tag.Get(tagMaxItems); tag != "" {
			if err := checkMaxItems(field, tag); err != nil {
				return err
			}
		}
		if sf.Tag.Get(tagUniqueItems) == constTRUE {
			if err := checkUniqueItems(field); err != nil {
				return err
			}
		}
	case reflect.Map:
		if tag := sf.Tag.Get(tagMinProperties); tag != "" {
			if err := checkMinProperties(field, tag); err != nil {
				return err
			}
		}
		if tag := sf.Tag.Get(tagMaxProperties); tag != "" {
			if err := checkMaxProperties(field, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateField performs tag-based validations: required, min/max, length constraints,
// enum, const, multipleOf, format, pattern, and slice/map validations.
func (c *Context) validateField(field reflect.Value, sf reflect.StructField) error {
	if sf.Tag.Get(tagRequired) == constTRUE && isEmptyValue(field) {
		return fmt.Errorf("field %s is required", sf.Name)
	}
	for _, check := range fieldConstraintCheckers {
		if err := check(field, sf); err != nil {
			return fmt.Errorf("field %s: %w", sf.Name, err)
		}
	}
	return nil
}

// validateStruct validates nested struct fields using their struct tags.
func (c *Context) validateStruct(v reflect.Value, parentField reflect.StructField) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		sf := t.Field(i)

		if sf.Tag.Get(tagRequired) == constTRUE && isEmptyValue(field) {
			return fmt.Errorf("field %s.%s is required", parentField.Name, sf.Name)
		}
		for _, check := range fieldConstraintCheckers {
			if err := check(field, sf); err != nil {
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
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		minValue, err := strconv.ParseInt(minTag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minTag)
		}
		if field.Int() < minValue {
			return fmt.Errorf("value %d must be >= %d", field.Int(), minValue)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		minValue, err := strconv.ParseUint(minTag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minTag)
		}
		if field.Uint() < minValue {
			return fmt.Errorf("value %d must be >= %d", field.Uint(), minValue)
		}

	case reflect.Float32, reflect.Float64:
		minValue, err := strconv.ParseFloat(minTag, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %s", minTag)
		}
		if field.Float() < minValue {
			return fmt.Errorf("value %g must be >= %g", field.Float(), minValue)
		}

	case reflect.Slice, reflect.Array, reflect.Map:
		minValue, err := strconv.Atoi(minTag)
		if err != nil {
			return fmt.Errorf("invalid min length: %s", minTag)
		}
		if field.Len() < minValue {
			return fmt.Errorf("length %d must be >= %d", field.Len(), minValue)
		}
	}

	return nil
}

func checkMax(field reflect.Value, maxTag string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		maxValue, err := strconv.ParseInt(maxTag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxTag)
		}
		if field.Int() > maxValue {
			return fmt.Errorf("value %d must be <= %d", field.Int(), maxValue)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		maxValue, err := strconv.ParseUint(maxTag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxTag)
		}
		if field.Uint() > maxValue {
			return fmt.Errorf("value %d must be <= %d", field.Uint(), maxValue)
		}

	case reflect.Float32, reflect.Float64:
		maxValue, err := strconv.ParseFloat(maxTag, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %s", maxTag)
		}
		if field.Float() > maxValue {
			return fmt.Errorf("value %g must be <= %g", field.Float(), maxValue)
		}

	case reflect.Slice, reflect.Array, reflect.Map:
		maxValue, err := strconv.Atoi(maxTag)
		if err != nil {
			return fmt.Errorf("invalid max length: %s", maxTag)
		}
		if field.Len() > maxValue {
			return fmt.Errorf("length %d must be <= %d", field.Len(), maxValue)
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

// checkFormat validates field based on format type.
// For slice fields, each element is validated individually.
func checkFormat(field reflect.Value, formatTag string, sf reflect.StructField) error {
	// Handle slice fields: validate each element
	if field.Kind() == reflect.Slice {
		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i)
			if err := checkFormatValue(elem, formatTag, sf); err != nil {
				return fmt.Errorf("element [%d]: %w", i, err)
			}
		}
		return nil
	}
	return checkFormatValue(field, formatTag, sf)
}

// formatValidators maps each format tag to its validator.
var formatValidators = map[string]func(string) error{
	formatEmail:        validateEmail,
	formatDateTime:     validateDateTime,
	formatDate:         validateDate,
	formatTime:         validateTime,
	formatDuration:     validateDuration,
	formatIPv4:         validateIPv4,
	formatIPv6:         validateIPv6,
	formatHostname:     validateHostname,
	formatUri:          validateUri,
	formatUUID:         validateUUID,
	formatURL:          validateURL,
	formatURIReference: validateURIReference,
	formatByte:         validateBase64,
	formatBase64:       validateBase64,
	formatMAC:          validateMAC,
	formatCIDR:         validateCIDR,
	formatE164:         validateE164,
	formatPhone:        validateE164,
	formatCreditCard:   validateCreditCard,
	formatSemver:       validateSemver,
	formatJSONPointer:  validateJSONPointer,
	formatULID:         validateULID,
	formatAlpha:        validateAlpha,
	formatAlphanumeric: validateAlphanumeric,
	formatNumeric:      validateNumeric,
	formatASCII:        validateASCII,
	formatLowercase:    validateLowercase,
	formatUppercase:    validateUppercase,
	formatSlug:         validateSlug,
	formatHexColor:     validateHexColor,
}

// checkFormatValue validates a single value against a format tag
func checkFormatValue(field reflect.Value, formatTag string, sf reflect.StructField) error {
	var value string
	if field.Type() == reflect.TypeOf(time.Time{}) {
		if field.IsZero() {
			return nil
		}
		t := field.Interface().(time.Time)
		value = t.Format(time.RFC3339)
	} else {
		value = field.String()
		// Skip validation if value is empty
		if value == "" {
			return nil
		}
	}

	// regex needs the companion pattern tag, so it is not in formatValidators.
	if formatTag == formatRegex {
		pattern := sf.Tag.Get(tagPattern)
		if pattern == "" {
			return fmt.Errorf("regex format requires 'pattern' tag")
		}
		return validateRegex(value, pattern)
	}

	if validate, ok := formatValidators[formatTag]; ok {
		return validate(value)
	}
	return fmt.Errorf("unsupported format: %s", formatTag)
}

// checkPattern validates a field against a regex pattern.
// For slice fields, each element is validated individually.
func checkPattern(field reflect.Value, pattern string) error {
	if field.Kind() == reflect.Slice {
		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i)
			if err := checkPatternValue(elem, pattern); err != nil {
				return fmt.Errorf("element [%d]: %w", i, err)
			}
		}
		return nil
	}
	return checkPatternValue(field, pattern)
}

func checkPatternValue(field reflect.Value, pattern string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("pattern validation can only be applied to string fields")
	}
	value := field.String()

	if value == "" {
		return nil
	}

	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return fmt.Errorf("regex validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("value does not match pattern '%s': %s", pattern, value)
	}
	return nil
}

// checkEnum validates that the field value is one of the allowed enum values.
// For slice fields, each element is validated individually.
func checkEnum(field reflect.Value, enumTag string) error {
	if field.Kind() == reflect.Slice {
		for i := 0; i < field.Len(); i++ {
			elem := field.Index(i)
			if err := checkEnumValue(elem, enumTag); err != nil {
				return fmt.Errorf("element [%d]: %w", i, err)
			}
		}
		return nil
	}
	return checkEnumValue(field, enumTag)
}

func checkEnumValue(field reflect.Value, enumTag string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("enum validation can only be applied to string fields")
	}

	value := field.String()

	if value == "" {
		return nil
	}

	allowedValues := strings.Split(enumTag, ",")

	for i, v := range allowedValues {
		allowedValues[i] = strings.TrimSpace(v)
	}

	// Check if value exists in allowed values
	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("value '%s' is not one of the allowed values: [%s]", value, strings.Join(allowedValues, ", "))
}

// checkConst validates that a string field equals a fixed constant value.
// For slice fields, each element is validated individually.
func checkConst(field reflect.Value, constTag string) error {
	if field.Kind() == reflect.Slice {
		for i := 0; i < field.Len(); i++ {
			if err := checkConstValue(field.Index(i), constTag); err != nil {
				return fmt.Errorf("element [%d]: %w", i, err)
			}
		}
		return nil
	}
	return checkConstValue(field, constTag)
}

func checkConstValue(field reflect.Value, constTag string) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("const validation can only be applied to string fields")
	}

	value := field.String()
	if value == "" {
		return nil
	}

	if value != constTag {
		return fmt.Errorf("value '%s' must equal the constant '%s'", value, constTag)
	}
	return nil
}

// checkExclusiveMin validates that a numeric field is strictly greater than the bound.
func checkExclusiveMin(field reflect.Value, tag string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bound, err := strconv.ParseInt(tag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMin value: %s", tag)
		}
		if field.Int() <= bound {
			return fmt.Errorf("value %d must be > %d", field.Int(), bound)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bound, err := strconv.ParseUint(tag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMin value: %s", tag)
		}
		if field.Uint() <= bound {
			return fmt.Errorf("value %d must be > %d", field.Uint(), bound)
		}
	case reflect.Float32, reflect.Float64:
		bound, err := strconv.ParseFloat(tag, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMin value: %s", tag)
		}
		if field.Float() <= bound {
			return fmt.Errorf("value %g must be > %g", field.Float(), bound)
		}
	}
	return nil
}

// checkExclusiveMax validates that a numeric field is strictly less than the bound.
func checkExclusiveMax(field reflect.Value, tag string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bound, err := strconv.ParseInt(tag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMax value: %s", tag)
		}
		if field.Int() >= bound {
			return fmt.Errorf("value %d must be < %d", field.Int(), bound)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bound, err := strconv.ParseUint(tag, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMax value: %s", tag)
		}
		if field.Uint() >= bound {
			return fmt.Errorf("value %d must be < %d", field.Uint(), bound)
		}
	case reflect.Float32, reflect.Float64:
		bound, err := strconv.ParseFloat(tag, 64)
		if err != nil {
			return fmt.Errorf("invalid exclusiveMax value: %s", tag)
		}
		if field.Float() >= bound {
			return fmt.Errorf("value %g must be < %g", field.Float(), bound)
		}
	}
	return nil
}

// checkMinProperties validates the minimum number of entries in a map field.
func checkMinProperties(field reflect.Value, tag string) error {
	if field.Kind() != reflect.Map {
		return nil
	}
	minValue, err := strconv.Atoi(tag)
	if err != nil {
		return fmt.Errorf("invalid minProperties value: %s", tag)
	}
	if field.Len() < minValue {
		return fmt.Errorf("map has %d properties, must have at least %d", field.Len(), minValue)
	}
	return nil
}

// checkMaxProperties validates the maximum number of entries in a map field.
func checkMaxProperties(field reflect.Value, tag string) error {
	if field.Kind() != reflect.Map {
		return nil
	}
	maxValue, err := strconv.Atoi(tag)
	if err != nil {
		return fmt.Errorf("invalid maxProperties value: %s", tag)
	}
	if field.Len() > maxValue {
		return fmt.Errorf("map has %d properties, must have at most %d", field.Len(), maxValue)
	}
	return nil
}
func validateEmail(value string) error {
	emailRegex := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailRegex, value)
	if err != nil {
		return fmt.Errorf("email validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid email format: %s", value)
	}
	return nil
}

func validateDateTime(value string) error {
	// RFC3339 format: 2006-01-02T15:04:05Z07:00
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return fmt.Errorf("invalid date-time format (expected RFC3339): %s", value)
	}
	return nil
}

func validateDate(value string) error {
	// ISO 8601 date format: YYYY-MM-DD
	_, err := time.Parse("2006-01-02", value)
	if err != nil {
		return fmt.Errorf("invalid date format (expected YYYY-MM-DD): %s", value)
	}
	return nil
}

func validateDuration(value string) error {
	// Go duration format: "300ms", "1.5h", "2h45m"
	_, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration format: %s", value)
	}
	return nil
}

func validateIPv4(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", value)
	}
	if ip.To4() == nil {
		return fmt.Errorf("not a valid IPv4 address: %s", value)
	}
	return nil
}

func validateIPv6(value string) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", value)
	}
	if ip.To4() != nil {
		return fmt.Errorf("not a valid IPv6 address: %s", value)
	}
	return nil
}

func validateUUID(value string) error {
	uuidRegex := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	matched, err := regexp.MatchString(uuidRegex, value)
	if err != nil {
		return fmt.Errorf("UUID validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid UUID format: %s", value)
	}
	return nil
}

func validateRegex(value, pattern string) error {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return fmt.Errorf("regex validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("value does not match pattern '%s': %s", pattern, value)
	}
	return nil
}
func validateHostname(value string) error {
	hostnameRegex := `^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*\.?$`
	matched, err := regexp.MatchString(hostnameRegex, value)
	if err != nil {
		return fmt.Errorf("hostname validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid hostname format: %s", value)
	}
	return nil
}
func validateUri(value string) error {
	uriRegex := `^[a-zA-Z][a-zA-Z0-9+.-]*:[^\s]*$`
	matched, err := regexp.MatchString(uriRegex, value)
	if err != nil {
		return fmt.Errorf("URI validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid URI format: %s", value)
	}
	return nil
}

func validateTime(value string) error {
	// RFC3339 full-time: 15:04:05Z, 15:04:05+07:00, 15:04:05.123Z, or local 15:04:05
	layouts := []string{"15:04:05Z07:00", "15:04:05.999999999Z07:00", "15:04:05"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid time format (expected RFC3339 full-time, e.g. 15:04:05Z07:00): %s", value)
}

func validateURL(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URL: %s", value)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL (must use http or https scheme): %s", value)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL (missing host): %s", value)
	}
	return nil
}

func validateURIReference(value string) error {
	if _, err := url.Parse(value); err != nil {
		return fmt.Errorf("invalid URI reference: %s", value)
	}
	return nil
}

func validateBase64(value string) error {
	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		return fmt.Errorf("invalid base64 value: %s", value)
	}
	return nil
}

func validateMAC(value string) error {
	if _, err := net.ParseMAC(value); err != nil {
		return fmt.Errorf("invalid MAC address: %s", value)
	}
	return nil
}

func validateCIDR(value string) error {
	if _, _, err := net.ParseCIDR(value); err != nil {
		return fmt.Errorf("invalid CIDR notation: %s", value)
	}
	return nil
}

func validateE164(value string) error {
	if !e164Regex.MatchString(value) {
		return fmt.Errorf("invalid phone number (expected E.164 format, e.g. +14155552671): %s", value)
	}
	return nil
}

// validateCreditCard verifies the number passes the Luhn checksum.
func validateCreditCard(value string) error {
	cleaned := strings.NewReplacer(" ", "", "-", "").Replace(value)
	if len(cleaned) < 12 || len(cleaned) > 19 {
		return fmt.Errorf("invalid credit card number: %s", value)
	}

	var sum int
	double := false
	for i := len(cleaned) - 1; i >= 0; i-- {
		ch := cleaned[i]
		if ch < '0' || ch > '9' {
			return fmt.Errorf("invalid credit card number: %s", value)
		}
		digit := int(ch - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	if sum%10 != 0 {
		return fmt.Errorf("invalid credit card number (failed Luhn check): %s", value)
	}
	return nil
}

func validateSemver(value string) error {
	if !semverRegex.MatchString(value) {
		return fmt.Errorf("invalid semantic version: %s", value)
	}
	return nil
}

func validateJSONPointer(value string) error {
	if !jsonPointerRegex.MatchString(value) {
		return fmt.Errorf("invalid JSON pointer (RFC 6901): %s", value)
	}
	return nil
}

func validateULID(value string) error {
	if !ulidRegex.MatchString(value) {
		return fmt.Errorf("invalid ULID: %s", value)
	}
	return nil
}

func validateAlpha(value string) error {
	if !alphaRegex.MatchString(value) {
		return fmt.Errorf("value must contain only letters: %s", value)
	}
	return nil
}

func validateAlphanumeric(value string) error {
	if !alphanumRegex.MatchString(value) {
		return fmt.Errorf("value must contain only letters and digits: %s", value)
	}
	return nil
}

func validateNumeric(value string) error {
	if !numericRegex.MatchString(value) {
		return fmt.Errorf("value must be numeric: %s", value)
	}
	return nil
}

func validateASCII(value string) error {
	for i := 0; i < len(value); i++ {
		if value[i] > 127 {
			return fmt.Errorf("value must contain only ASCII characters: %s", value)
		}
	}
	return nil
}

func validateLowercase(value string) error {
	if value != strings.ToLower(value) {
		return fmt.Errorf("value must be lowercase: %s", value)
	}
	return nil
}

func validateUppercase(value string) error {
	if value != strings.ToUpper(value) {
		return fmt.Errorf("value must be uppercase: %s", value)
	}
	return nil
}

func validateSlug(value string) error {
	if !slugRegex.MatchString(value) {
		return fmt.Errorf("value must be a valid slug (lowercase alphanumeric and hyphens): %s", value)
	}
	return nil
}

func validateHexColor(value string) error {
	if !hexColorRegex.MatchString(value) {
		return fmt.Errorf("invalid hex color (expected #RGB or #RRGGBB): %s", value)
	}
	return nil
}
func checkMultipleOf(field reflect.Value, tag string) error {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		multipleOf, err := parseInt(tag)
		if err != nil {
			return fmt.Errorf("invalid multipleOf tag: %w", err)
		}
		if field.Int()%multipleOf != 0 {
			return fmt.Errorf("value %d is not a multiple of %d", field.Int(), multipleOf)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		multipleOf, err := parseUint(tag)
		if err != nil {
			return fmt.Errorf("invalid multipleOf tag: %w", err)
		}
		if field.Uint()%multipleOf != 0 {
			return fmt.Errorf("value %d is not a multiple of %d", field.Uint(), multipleOf)
		}
	case reflect.Float32, reflect.Float64:
		multipleOf, err := parseFloat(tag)
		if err != nil {
			return fmt.Errorf("invalid multipleOf tag: %w", err)
		}
		if multipleOf == 0 {
			return fmt.Errorf("multipleOf cannot be zero")
		}

		value := field.Float()
		remainder := math.Mod(value, multipleOf)

		const epsilon = 1e-9
		if math.Abs(remainder) > epsilon && math.Abs(remainder-multipleOf) > epsilon {
			return fmt.Errorf("value %f is not a multiple of %f", value, multipleOf)
		}
	default:
		return fmt.Errorf("multipleOf validation not supported for type %s", field.Kind())
	}
	return nil
}
func checkUniqueItems(field reflect.Value) error {
	if field.Kind() == reflect.Slice {
		seen := make(map[interface{}]bool)
		for i := 0; i < field.Len(); i++ {
			item := field.Index(i).Interface()
			if seen[item] {
				return fmt.Errorf("slice contains duplicate item: %v", item)
			}
			seen[item] = true
		}
	}
	return nil

}

func checkMaxItems(field reflect.Value, tag string) error {
	maxItems, err := strconv.Atoi(tag)
	if err != nil {
		return fmt.Errorf("invalid maxItems value: %s", tag)
	}

	if field.Kind() == reflect.Slice {
		if field.Len() > maxItems {
			return fmt.Errorf("slice length %d must be at most %d items", field.Len(), maxItems)
		}
	}
	return nil

}

func checkMinItems(field reflect.Value, tag string) error {
	minItems, err := strconv.Atoi(tag)
	if err != nil {
		return fmt.Errorf("invalid minItems value: %s", tag)
	}

	if field.Kind() == reflect.Slice {
		if field.Len() < minItems {
			return fmt.Errorf("slice length %d must be at least %d items", field.Len(), minItems)
		}
	}
	return nil

}
func parseFloat(tag string) (float64, error) {
	val, err := strconv.ParseFloat(tag, 64)
	if err != nil {
		return 0.0, fmt.Errorf("invalid float: %w", err)
	}
	return val, nil
}

func parseInt(tag string) (int64, error) {
	val, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}
	return val, nil
}

func parseUint(tag string) (uint64, error) {
	val, err := strconv.ParseUint(tag, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid unsigned integer: %w", err)
	}
	return val, nil
}
