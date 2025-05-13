// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configoptional // import "go.opentelemetry.io/collector/config/configoptional"

import (
	"go.opentelemetry.io/collector/confmap"
)

// Optional is a type that can be used to represent a value that may or may not be present.
// It supports two flavors: Some(value), and None(factory).
type Optional[T any] struct {
	value T

	// defaultFn returns a default value for the type T.
	// It also acts as a flag to indicate whether the value is present or not.
	// If defaultFn is nil, it means the value is present.
	defaultFn *DefaultFunc[T]
}

// DefaultFunc is a function type that returns a default value of type T.
type DefaultFunc[T any] func() T

var _ confmap.Unmarshaler = (*Optional[any])(nil)

// Some creates an Optional with a value and no factory.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value}
}

// Factory is an opaque type that can be used to create a default value for the type T.
// Factories should be package variables.
type Factory[T any] struct {
	defaultFn *DefaultFunc[T]
}

// NewFactory creates a new Factory with the given default function.
// Factories should be package variables.
// The reason we use a factory instead of passing a default T directly
// is to allow this to work with T being a pointer,
// because we wouldn't want to copy the value of a pointer
// since it might reuse (and override) some shared state.
func NewFactory[T any](defaultFn DefaultFunc[T]) Factory[T] {
	return Factory[T]{defaultFn: &defaultFn}
}

// None creates an Optional which has no value
// and a factory to create a default value.
//
// On successful unmarshal, the factory is erased.
func None[T any](factory Factory[T]) Optional[T] {
	return Optional[T]{defaultFn: factory.defaultFn}
}

// GetOrInsertDefault returns the value of the Optional.
// If the Optional is None(factory), it creates the default value.
func (o *Optional[T]) GetOrInsertDefault() *T {
	if !o.HasValue() {
		o.value = (*o.defaultFn)()
		o.defaultFn = nil
	}
	return &o.value
}

// HasValue checks if the Optional has a value.
func (o Optional[T]) HasValue() bool {
	return o.defaultFn == nil
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
	defaultFn := o.defaultFn
	_ = o.GetOrInsertDefault()
	if err := conf.Unmarshal(&o.value); err != nil {
		// restore defaultFn if unmarshaling fails
		o.defaultFn = defaultFn
		return err
	}
	return nil
}
