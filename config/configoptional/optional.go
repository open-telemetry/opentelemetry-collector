// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"fmt"
	"reflect"

	"go.opentelemetry.io/collector/confmap"
)

// Optional represents a value that may or may not be present.
// It supports three flavors: Some(value), Default(value), and None.
// The zero value of Optional is None.
type Optional[T any] struct {
	// Enabled indicates if the value is present.
	Enabled bool `mapstructure:"enabled"`

	value T
}

// deref a reflect.Type to its underlying type.
func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// assertNoEnabledField checks that a struct type
// does not have a field with a mapstructure tag "enabled".
func assertNoEnabledField[T any]() {
	var i T
	t := deref(reflect.TypeOf(i))

	if t.Kind() != reflect.Struct {
		// Not a struct, no need to check for "enabled" field.
		return
	}

	// Check if the struct has a field with the name "enabled".
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("mapstructure") == "enabled" {
			panic("configoptional: underlying type cannot have a field with mapstructure tag 'enabled'")
		}
	}
}

// assertStruct checks that a given type is of struct kind.
func assertStruct[T any]() {
	var i T
	t := deref(reflect.TypeOf(i))

	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("configoptional: %q does not have a struct kind", t))
	}
}

// Some creates an Optional with a set value.
// It panics if T has a field with the mapstructure tag "enabled".
func Some[T any](value T) Optional[T] {
	assertNoEnabledField[T]()
	return Optional[T]{value: value, Enabled: true}
}

<<<<<<< Updated upstream
// None has no value.
||||||| Stash base
func Default[T any](value T) Optional[T] {
	assertNoEnabledField[T]()
	return Optional[T]{value: value, Enabled: false}
}

// None has no value.
=======
// Default creates an Optional with a default value for unmarshaling.
// It panics if
// - T is not a struct OR
// - T has a field with the mapstructure tag "enabled".
func Default[T any](value T) Optional[T] {
	assertStruct[T]()
	assertNoEnabledField[T]()
	return Optional[T]{value: value, Enabled: false}
}

// None has no value. It is equivalent to Default(zero value).
>>>>>>> Stashed changes
// It panics if T has a field with the mapstructure tag "enabled".
func None[T any]() Optional[T] {
	assertNoEnabledField[T]()
	return Optional[T]{}
}

// Get returns the value of the Optional.
// If the value is not present, it returns nil.
func (o *Optional[T]) Get() *T {
	if !o.Enabled {
		return nil
	}
	return &o.value
}

var _ confmap.Unmarshaler = (*Optional[any])(nil)

// Unmarshal implements the confmap.Unmarshaler interface.
// Unmarshaling from an empty map will set Enabled to true.
//
// It only works for struct types.
// It panics if T has a field with the mapstructure tag "enabled".
func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	assertNoEnabledField[T]()

	if !conf.IsSet("enabled") {
		o.Enabled = true
	} else {
		if b, ok := conf.Get("enabled").(bool); ok {
			o.Enabled = b
		} else {
			return fmt.Errorf("configoptional: expected bool for 'enabled' field, got %T", conf.Get("enabled"))
		}

		m := conf.ToStringMap()
		delete(m, "enabled")
		conf = confmap.NewFromStringMap(m)
	}

	return conf.Unmarshal(&o.value)
}
