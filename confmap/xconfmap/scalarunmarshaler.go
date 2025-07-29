// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap // import "go.opentelemetry.io/collector/confmap/xconfmap"

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
)

func WithScalarUnmarshaler() confmap.UnmarshalOption {
	return internal.UnmarshalOptionFunc(func(uo *internal.UnmarshalOptions) {
		uo.AdditionalDecodeHookFuncs = append(uo.AdditionalDecodeHookFuncs, scalarunmarshalerHookFunc(uo))
	})
}

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar
// value.
//
// This should be used for types like `Wrapper[T]` where T is a scalar type, and
// the wrapper type needs to implement custom logic for unmarshaling from a
// scalar value (e.g. `5` for `Wrapper[int]`) into the wrapper type (e.g.
// `Wrapper[int]{inner: 5}`).
type ScalarUnmarshaler interface {
	// UnmarshalScalar unmarshals a scalar into a value in a custom way. The
	// input value is guaranteed to be a scalar value as defined by the YAML
	// spec.
	//
	// In the case where `null` is specified in the config, `nil` is passed to
	// this function, which should be handled accordingly.
	UnmarshalScalar(val any) error

	// ScalarType returns a value that can be used to get the type of the scalar
	// using reflection. This is intended to be used for decoding the value from
	// the input config type into the corresponding Go type.
	//
	// The type returned here should be type `T` for a generic wrapper
	// `Wrapper[T]` as Go's reflection utilities do not allow for retrieving the
	// type of `T` directly from the wrapper type.
	ScalarType() any
}

// scalarunmarshalerHookFunc handles decoding for types implementing the
// ScalarUnmarshaler interface.
func scalarunmarshalerHookFunc(opts *internal.UnmarshalOptions) mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from, to reflect.Value) (any, error) {
		if !to.CanAddr() {
			return from.Interface(), nil
		}

		toPtr := to.Addr().Interface()

		unmarshaler, ok := toPtr.(ScalarUnmarshaler)
		if !ok {
			return from.Interface(), nil
		}

		if to.Addr().IsNil() {
			unmarshaler = reflect.New(to.Type()).Interface().(ScalarUnmarshaler)
		}

		resultVal := reflect.New(reflect.TypeOf(unmarshaler.ScalarType()))

		// Non-nil maps shouldn't be handled by this hook as they indicate
		// struct-typed input. Nil maps where the Wrapper type wraps a struct
		// also shouldn't by handled by this hook since `null` semantics are
		// different for structs vs. scalar types.
		if from.Kind() == reflect.Map && (!from.IsNil() || resultVal.Elem().Kind() == reflect.Struct) {
			return from.Interface(), nil
		}

		// A nil map where the wrapped type isn't a struct indicate `null` input
		// for a scalar value. To match unmarshaling behavior with
		// pointer-to-scalar types, we pass a nil value to the UnmarshalScalar
		// method.
		if from.Kind() == reflect.Map && from.IsNil() {
			if err := unmarshaler.UnmarshalScalar(nil); err != nil {
				return nil, err
			}
		} else {
			// If we have a non-nil, non-map value, we decode it into the scalar
			// type and then pass it to UnmarshalScalar.
			if err := internal.Decode(from.Interface(), resultVal.Interface(), *opts, false); err != nil {
				return nil, err
			}
			if err := unmarshaler.UnmarshalScalar(resultVal.Elem().Interface()); err != nil {
				return nil, err
			}
		}

		return unmarshaler, nil
	})
}

// safeWrapDecodeHookFunc wraps a DecodeHookFuncValue to ensure fromVal is a valid `reflect.Value`
// object and therefore it is safe to call `reflect.Value` methods on fromVal.
//
// Use this only if the hook does not need to be called on untyped nil values.
// Typed nil values are safe to call and will be passed to the hook.
// See https://github.com/golang/go/issues/51649
func safeWrapDecodeHookFunc(
	f mapstructure.DecodeHookFuncValue,
) mapstructure.DecodeHookFuncValue {
	return func(fromVal, toVal reflect.Value) (any, error) {
		if !fromVal.IsValid() {
			return nil, nil
		}
		return f(fromVal, toVal)
	}
}
