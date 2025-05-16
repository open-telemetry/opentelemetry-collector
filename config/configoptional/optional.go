// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/confmap"
)

// Optional represents a value that may or may not be present.
// It supports two flavors for all types: Some(value) and None.
// It supports a third flavor for struct types: Default(defaultFn).
//
// For struct types, it supports unmarshaling from a configuration source.
// The zero value of Optional is None.
type Optional[T any] struct {
	// value is the value of the Optional.
	value T

	// hasValue indicates if the Optional has a value.
	// It MUST be false if the defaultFn is not nil.
	hasValue bool

	// defaultFn returns a default value for the type T.
	// It MUST be nil if hasValue is true.
	defaultFn *DefaultFunc[T]
}

// DefaultFunc returns a default value of type T.
//
// DefaultFuncs should be defined as package-level variables to be able to
// use them in the Default constructor.
type DefaultFunc[T any] func() T

// assertStructKind checks if T is of struct or pointer to struct kind.
func assertStructKind[T any]() error {
	var instance T
	t := reflect.TypeOf(instance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("configoptional: %q does not have a struct kind", t)
	}

	return nil
}

// Some creates an Optional with a value and no factory.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

// Default creates an Optional which has no value
// and a pointer to a DefaultFunc to create a default value.
// T must be of struct or pointer to struct kind.
//
// On successful unmarshal, the default function is erased.
//
// Define default functions as package-level variables to avoid
// creating a new function each time.
//
// This function panics if
//   - defaultFn is nil OR
//   - T is not of struct or pointer to struct kind.
func Default[T any](defaultFn *DefaultFunc[T]) Optional[T] {
	if defaultFn == nil {
		panic("configoptional: defaultFn must not be nil")
	}

	if err := assertStructKind[T](); err != nil {
		panic(err)
	}

	return Optional[T]{defaultFn: defaultFn, hasValue: false}
}

// None has no value.
// For T of struct or pointer to struct kind, this is equivalent to
// Default(zeroFn) where zeroFn creates a zero value of type T.
func None[T any]() Optional[T] {
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
// If the value was None, on success, the factory is erased.
//
// T must be of struct or pointer to struct kind.
// Scalar values are not supported.
func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	if !o.HasValue() && o.defaultFn != nil {
		o.value = (*o.defaultFn)()
	}
	if err := conf.Unmarshal(&o.value); err != nil {
		return err
	}

	o.defaultFn = nil
	o.hasValue = true
	return nil
}
