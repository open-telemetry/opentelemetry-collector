// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"go.opentelemetry.io/collector/confmap"
)

// Optional is a type that can be used to represent a value that may or may not be present.
// It supports two flavors: Some(value), and Default(factory).
type Optional[T any] struct {
	value    T
	hasValue bool

	// defaultFn returns a default value for the type T.
	// It MUST be nil if `hasValue` is true.
	defaultFn *DefaultFunc[T]
}

// DefaultFunc is a function type that returns a default value of type T.
type DefaultFunc[T any] func() T

var _ confmap.Unmarshaler = (*Optional[any])(nil)

// Some creates an Optional with a value and no factory.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, hasValue: true}
}

// Factory is an opaque type that can be used to create a default value for the type T.
// Factories should be package variables.
type Factory[T any] struct {
	defaultFn *DefaultFunc[T]
}

// NewFactory creates a new Factory with the given default function.
// Factories should be package variables, to ensure the default function
// is the same for all instances of the type T.
func NewFactory[T any](defaultFn DefaultFunc[T]) Factory[T] {
	return Factory[T]{defaultFn: &defaultFn}
}

// Default creates an Optional which has no value
// and a factory to create a default value.
//
// On successful unmarshal, the factory is erased.
func Default[T any](factory Factory[T]) Optional[T] {
	return Optional[T]{defaultFn: factory.defaultFn, hasValue: false}
}

// None is equivalent to Default(zeroFactory) where zeroFactory creates a zero value of type T.
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// GetOrInsertDefault returns the value of the Optional.
// If the Optional is None(factory), it creates the default value.
func (o *Optional[T]) GetOrInsertDefault() *T {
	if !o.HasValue() && o.defaultFn != nil {
		o.value = (*o.defaultFn)()
		o.defaultFn = nil
		o.hasValue = true
	}
	return &o.value
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

// Unmarshal the configuration into the Optional value.
// If the value was None, on success, the factory is erased.
func (o *Optional[T]) Unmarshal(conf *confmap.Conf) error {
	// GetOrInsertDefault would erase the factory.
	// We only want to do so if unmarshaling succeeds.
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
