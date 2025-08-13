// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type flavor int

const (
	noneFlavor    flavor = 0
	defaultFlavor flavor = 1
	someFlavor    flavor = 2
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

	// flavor indicates the flavor of the Optional.
	// The zero value of flavor is noneFlavor.
	flavor flavor
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
	return Optional[T]{value: value, flavor: someFlavor}
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
	return Optional[T]{value: value, flavor: defaultFlavor}
}

// None has no value. It has the same behavior as a nil pointer when unmarshaling.
//
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
	return o.flavor == someFlavor
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
// The behavior of this method depends on the state of the Optional:
//   - None[T]: does nothing if the configuration is nil, otherwise it unmarshals into the zero value of T.
//   - Some[T](val): equivalent to unmarshaling into a field of type T with value val.
//   - Default[T](val), equivalent to unmarshaling into a field of type T with base value val,
//     using val without overrides from the configuration if the configuration is nil.
//
// T must be derefenceable to a type with struct kind and not have an 'enabled' field.
// Scalar values are not supported.
func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	if err := assertNoEnabledField[T](); err != nil {
		return err
	}

	if o.flavor == noneFlavor && conf.ToStringMap() == nil {
		// If the Optional is None and the configuration is nil, we do nothing.
		// This replicates the behavior of unmarshaling into a field with a nil pointer.
		return nil
	}

	if err := conf.Unmarshal(&o.value); err != nil {
		return err
	}

	o.flavor = someFlavor
	return nil
}

var _ confmap.Marshaler = (*Optional[any])(nil)

// Marshal the Optional value into the configuration.
// If the Optional is None or Default, it does not marshal anything.
// If the Optional is Some, it marshals the value into the configuration.
//
// T must be derefenceable to a type with struct kind.
// Scalar values are not supported.
func (o Optional[T]) Marshal(conf *confmap.Conf) error {
	if err := assertStructKind[T](); err != nil {
		return err
	}

	if o.flavor == noneFlavor || o.flavor == defaultFlavor {
		// Optional is None or Default, do not marshal anything.
		return conf.Marshal(map[string]any(nil))
	}

	if err := conf.Marshal(o.value); err != nil {
		return fmt.Errorf("configoptional: failed to marshal Optional value: %w", err)
	}

	return nil
}

var _ xconfmap.Validator = (*Optional[any])(nil)

// Validate implements [xconfmap.Validator]. This is required because the
// private fields in [xconfmap.Validator] can't be seen by the reflection used
// by [xconfmap.Validate], and therefore we have to continue the validation
// chain manually. This method isn't meant to be called directly, and should
// generally only be called by [xconfmap.Validate].
func (o *Optional[T]) Validate() error {
	// When the flavor is None, the user has not passed this value,
	// and therefore we should not validate it. The parent struct holding
	// the Optional type can determine whether a None value is valid for
	// a given config.
	//
	// If the flavor is still Default, then the user has not passed this
	// value and we should also not validate it.
	if o.flavor == noneFlavor || o.flavor == defaultFlavor {
		return nil
	}

	// For the some flavor, validate the actual value.
	return xconfmap.Validate(o.value)
}
