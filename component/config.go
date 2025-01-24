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
// MUST implement the ConfigValidator if any validation is required for that part of the configuration
// (e.g. check if a required field is present).
//
// A valid implementation MUST pass the check componenttest.CheckConfigStruct (return nil error).
type Config any

// As interface types are only used for static typing, a common idiom to find the reflection Type
// for an interface type Foo is to use a *Foo value.
var configValidatorType = reflect.TypeOf((*ConfigValidator)(nil)).Elem()

// ConfigValidator defines an optional interface for configurations to implement to do validation.
type ConfigValidator interface {
	// Validate the configuration and returns an error if invalid.
	Validate() error
}

// ValidateConfig validates a config, by doing this:
//   - Call Validate on the config itself if the config implements ConfigValidator.
func ValidateConfig(cfg Config) error {
	var err error
	errs := validate(reflect.ValueOf(cfg))

	for _, suberr := range errs {
		if suberr.err != nil {
			if suberr.path != "" {
				err = errors.Join(err, fmt.Errorf("%s: %w", suberr.path, suberr.err))
			} else {
				err = errors.Join(err, suberr.err)
			}
		}
	}

	return err
}

type pathError struct {
	err  error
	path string
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
		errs = append(errs, pathError{err: callValidateIfPossible(v), path: ""})
		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() {
				continue
			}
			subpathErrs := validate(v.Field(i))
			for _, err := range subpathErrs {
				field := v.Type().Field(i)
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

				var path string
				if len(err.path) > 0 {
					path = strings.Join([]string{fieldName, err.path}, "::")
				} else {
					path = fieldName
				}
				errs = append(errs, pathError{
					err:  err.err,
					path: path,
				})
			}
		}
		return errs
	case reflect.Slice, reflect.Array:
		errs = append(errs, pathError{err: callValidateIfPossible(v), path: ""})
		// Reflect on the pointed data and check each of its fields.
		for i := 0; i < v.Len(); i++ {
			subPathErrs := validate(v.Index(i))

			for _, err := range subPathErrs {
				var path string
				if len(err.path) > 0 {
					path = strings.Join([]string{strconv.Itoa(i), err.path}, "::")
				} else {
					path = strconv.Itoa(i)
				}

				errs = append(errs, pathError{
					err:  err.err,
					path: path,
				})
			}
		}
		return errs
	case reflect.Map:
		errs = append(errs, pathError{err: callValidateIfPossible(v), path: ""})
		iter := v.MapRange()
		for iter.Next() {
			keyErrs := validate(iter.Key())
			valueErrs := validate(iter.Value())
			var key string

			if str, ok := iter.Key().Interface().(string); ok {
				key = str
			} else if stringer, ok := iter.Key().Interface().(fmt.Stringer); ok {
				key = stringer.String()
			} else {
				key = fmt.Sprintf("[%T key]", iter.Key().Interface())
			}

			for _, err := range keyErrs {
				var path string
				if len(err.path) > 0 {
					path = strings.Join([]string{key, err.path}, "::")
				} else {
					path = key
				}
				errs = append(errs, pathError{err: err.err, path: path})
			}

			for _, err := range valueErrs {
				var path string
				if len(err.path) > 0 {
					path = strings.Join([]string{key, err.path}, "::")
				} else {
					path = key
				}
				errs = append(errs, pathError{err: err.err, path: path})
			}
		}
		return errs
	default:
		return []pathError{{err: callValidateIfPossible(v), path: ""}}
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
