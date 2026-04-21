// Package validator contains small validation helpers used after decoding.
package validator

import (
	"fmt"
	"reflect"
	"strings"

	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
)

// ValidateRequired checks struct fields tagged with `bridge:"required"` or
// `validate:"required"`.
func ValidateRequired(out any) error {
	value := reflect.ValueOf(out)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil
	}
	return validateValue(value.Elem(), "")
}

func validateValue(value reflect.Value, prefix string) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return validateValue(value.Elem(), prefix)
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		// Continue into field iteration below.
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			path := fmt.Sprintf("%s[%d]", prefix, i)
			if prefix == "" {
				path = fmt.Sprintf("[%d]", i)
			}
			if err := validateValue(value.Index(i), path); err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			path := fmt.Sprintf("%s[%v]", prefix, iter.Key().Interface())
			if prefix == "" {
				path = fmt.Sprintf("[%v]", iter.Key().Interface())
			}
			if err := validateValue(iter.Value(), path); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldValue := value.Field(i)
		name := jsonName(field)
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if hasRequiredTag(field) && isZero(fieldValue) {
			return fmt.Errorf("%w: %s is required", bridgeerrors.ErrValidationFailed, path)
		}
		if err := validateValue(fieldValue, path); err != nil {
			return err
		}
	}
	return nil
}

func hasRequiredTag(field reflect.StructField) bool {
	return tagHasToken(field.Tag.Get("bridge"), "required") ||
		tagHasToken(field.Tag.Get("validate"), "required")
}

func tagHasToken(tag string, want string) bool {
	for _, token := range strings.Split(tag, ",") {
		if strings.TrimSpace(token) == want {
			return true
		}
	}
	return false
}

func jsonName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" || name == "-" {
		return field.Name
	}
	return name
}

func isZero(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	return value.IsZero()
}
