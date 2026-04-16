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
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
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
	return strings.Contains(field.Tag.Get("bridge"), "required") ||
		strings.Contains(field.Tag.Get("validate"), "required")
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

