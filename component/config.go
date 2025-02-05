// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package component // import "go.opentelemetry.io/collector/component"

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Config defines the configuration for a component.Component.
//
// Implementations and/or any sub-configs (other types embedded or included in the Config implementation)
// MUST implement xconfmap.Validator if any validation is required for that part of the configuration
// (e.g. check if a required field is present).
//
// A valid implementation MUST pass the check componenttest.CheckConfigStruct (return nil error).
type Config any

// As interface types are only used for static typing, a common idiom to find the reflection Type
// for an interface type Foo is to use a *Foo value.
var configValidatorType = reflect.TypeOf((*ConfigValidator)(nil)).Elem()

// ConfigValidator defines an optional interface for configurations to implement to do validation.
//
// Deprecated: [v0.120.0] use xconfmap.Validator.
type ConfigValidator interface {
	// Validate the configuration and returns an error if invalid.
	Validate() error
}

// ValidateConfig validates a config, by doing this:
//   - Call Validate on the config itself if the config implements ConfigValidator.
//
// Deprecated: [v0.120.0] use xconfmap.Validate.
func ValidateConfig(cfg Config) error {
	var err error

	for _, validationErr := range validate(reflect.ValueOf(cfg)) {
		err = errors.Join(err, validationErr)
	}

	return err
}

type pathError struct {
	err  error
	path []string
}

func (pe pathError) Error() string {
	if len(pe.path) > 0 {
		var path string
		sb := strings.Builder{}

		_, _ = sb.WriteString(pe.path[len(pe.path)-1])
		for i := len(pe.path) - 2; i >= 0; i-- {
			_, _ = sb.WriteString("::")
			_, _ = sb.WriteString(pe.path[i])
		}
		path = sb.String()

		return fmt.Sprintf("%s: %s", path, pe.err)
	}

	return pe.err.Error()
}

func (pe pathError) Unwrap() error {
	return pe.err
}

func validate(v reflect.Value) []pathError {
	errs := []pathError{}
	// Validate the value itself.
	switch v.Kind() {
	case reflect.Invalid:
		return nil
	case reflect.Ptr, reflect.Interface:
		return validate(v.Elem())
	case reflect.Struct:
		err := callValidateIfPossible(v)
		if err != nil {
			errs = append(errs, pathError{err: err})
		}

		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			field := v.Type().Field(i)
			path := fieldName(field)

			subpathErrs := validate(v.Field(i))
			for _, err := range subpathErrs {
				errs = append(errs, pathError{
					err:  err.err,
					path: append(err.path, path),
				})
			}
		}
		return errs
	case reflect.Slice, reflect.Array:
		err := callValidateIfPossible(v)
		if err != nil {
			errs = append(errs, pathError{err: err})
		}

		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.Len(); i++ {
			subPathErrs := validate(v.Index(i))

			for _, err := range subPathErrs {
				errs = append(errs, pathError{
					err:  err.err,
					path: append(err.path, strconv.Itoa(i)),
				})
			}
		}
		return errs
	case reflect.Map:
		err := callValidateIfPossible(v)
		if err != nil {
			errs = append(errs, pathError{err: err})
		}

		iter := v.MapRange()
		for iter.Next() {
			keyErrs := validate(iter.Key())
			valueErrs := validate(iter.Value())
			key := stringifyMapKey(iter.Key())

			for _, err := range keyErrs {
				errs = append(errs, pathError{err: err.err, path: append(err.path, key)})
			}

			for _, err := range valueErrs {
				errs = append(errs, pathError{err: err.err, path: append(err.path, key)})
			}
		}
		return errs
	default:
		err := callValidateIfPossible(v)
		if err != nil {
			return []pathError{{err: err}}
		}

		return nil
	}
}

func callValidateIfPossible(v reflect.Value) error {
	// If the value type implements ConfigValidator just call Validate
	if v.Type().Implements(configValidatorType) {
		return v.Interface().(ConfigValidator).Validate()
	}

	// If the pointer type implements ConfigValidator call Validate on the pointer to the current value.
	if reflect.PointerTo(v.Type()).Implements(configValidatorType) {
		// If not addressable, then create a new *V pointer and set the value to current v.
		if !v.CanAddr() {
			pv := reflect.New(reflect.PointerTo(v.Type()).Elem())
			pv.Elem().Set(v)
			v = pv.Elem()
		}
		return v.Addr().Interface().(ConfigValidator).Validate()
	}

	return nil
}

func fieldName(field reflect.StructField) string {
	var fieldName string
	if tag, ok := field.Tag.Lookup("mapstructure"); ok {
		tags := strings.Split(tag, ",")
		if len(tags) > 0 {
			fieldName = tags[0]
		}
	}
	// Even if the mapstructure tag exists, the field name may not
	// be available, so set it if it is still blank.
	if len(fieldName) == 0 {
		fieldName = strings.ToLower(field.Name)
	}

	return fieldName
}

func stringifyMapKey(val reflect.Value) string {
	var key string

	if str, ok := val.Interface().(string); ok {
		key = str
	} else if stringer, ok := val.Interface().(fmt.Stringer); ok {
		key = stringer.String()
	} else {
		switch val.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
			key = fmt.Sprintf("[%T key]", val.Interface())
		default:
			key = fmt.Sprintf("%v", val.Interface())
		}
	}

	return key
}
