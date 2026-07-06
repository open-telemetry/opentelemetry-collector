// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"reflect"
	"unicode/utf8"
)

// ValidateUTF8 returns false when any string in v contains invalid UTF-8.
func ValidateUTF8(v any) bool {
	return validateUTF8(reflect.ValueOf(v))
}

func validateUTF8(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return true
		}
		return validateUTF8(v.Elem())
	case reflect.String:
		return utf8.ValidString(v.String())
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !validateUTF8(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !validateUTF8(v.Index(i)) {
				return false
			}
		}
	}
	return true
}
