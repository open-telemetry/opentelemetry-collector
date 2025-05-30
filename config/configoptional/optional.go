// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/collector/confmap"
)

// Optional represents a value that may or may not be present.
// It supports two flavors for all types: Some(value) and None.
// It supports a third flavor for struct types: Default(defaultVal).
//
// For struct types, it supports unmarshaling from a configuration source.
// The zero value of Optional is None.
type Optional[T any] struct {
	// value is the value of the Optional.
	value T

	// hasValue indicates if the Optional has a value.
	hasValue bool
}

// deref a reflect.Type to its underlying type.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// assertStructKind checks if T can be dereferenced into a type with struct kind.
//
// We assert this because our unmarshaling logic currently only supports structs.
// This can be removed if we ever support scalar values.
func assertStructKind[T any]() error {
	var instance T
	t := deref(reflect.TypeOf(instance))
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("configoptional: %q does not have a struct kind", t)
	}

	return nil
}

// assertNoEnabledField checks that a struct type
// does not have a field with a mapstructure tag "enabled".
//
// We assert this because we discussed an alternative design where we have an explicit
// "enabled" field in the struct to indicate if the struct is enabled or not.
// See https://github.com/open-telemetry/opentelemetry-collector/pull/13060.
// This can be removed if we ever support such a design (or if we just want to allow
// the "enabled" field in the struct).
func assertNoEnabledField[T any]() error {
	var i T
	t := deref(reflect.TypeOf(i))
	if t.Kind() != reflect.Struct {
		// Not a struct, no need to check for "enabled" field.
		return nil
	}

	// Check if the struct has a field with the name "enabled".
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		mapstructureTags := strings.SplitN(field.Tag.Get("mapstructure"), ",", 2)
		if len(mapstructureTags) > 0 && mapstructureTags[0] == "enabled" {
			return errors.New("configoptional: underlying type cannot have a field with mapstructure tag 'enabled'")
		}
	}
	return nil
}

// Some creates an Optional with a value and no factory.
//
// It panics if T has a field with the mapstructure tag "enabled".
func Some[T any](value T) Optional[T] {
	if err := assertNoEnabledField[T](); err != nil {
		panic(err)
	}
	return Optional[T]{value: value, hasValue: true}
}

// Default creates an Optional with a default value for unmarshaling.
//
// It panics if
// - T is not a struct OR
// - T has a field with the mapstructure tag "enabled".
func Default[T any](value T) Optional[T] {
	err := errors.Join(assertStructKind[T](), assertNoEnabledField[T]())
	if err != nil {
		panic(err)
	}
	return Optional[T]{value: value, hasValue: false}
}

// None has no value.
//
// For T of struct or pointer to struct kind, this is equivalent to
// Default(zeroVal) where zeroVal is the zero value of type T.
// The zero value of Optional[T] is None[T]. Prefer using this constructor
// for validation.
//
// It panics if T has a field with the mapstructure tag "enabled".
func None[T any]() Optional[T] {
	if err := assertNoEnabledField[T](); err != nil {
		panic(err)
	}
	return Optional[T]{}
}

// HasValue checks if the Optional has a value.
func (o Optional[T]) HasValue() bool {
	return o.hasValue
}

// Get returns the value of the Optional.
// If the value is not present, it returns nil.
func (o *Optional[T]) Get() *T {
	if !o.HasValue() {
		return nil
	}
	return &o.value
}

var _ confmap.Unmarshaler = (*Optional[any])(nil)

// Unmarshal the configuration into the Optional value.
//
// T must be derefenceable to a type with struct kind and not have an 'enabled' field.
// Scalar values are not supported.
func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	if err := assertNoEnabledField[T](); err != nil {
		return err
	}

	if err := conf.Unmarshal(&o.value); err != nil {
		return err
	}

	o.hasValue = true
	return nil
}
