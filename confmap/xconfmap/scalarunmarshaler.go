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
// to customize their behavior when the type under the wrapper is a scalar value.
type ScalarUnmarshaler interface {
	//UnmarshalScalar unmarshals a scalar into a value in a custom way.
	UnmarshalScalar(val any) error

	// ScalarType returns a value that can be used to get the type
	// of the scalar using reflection.
	ScalarType() any
}

// Provides a mechanism for individual structs to define their own unmarshal logic,
// by implementing the Unmarshaler interface, unless skipTopLevelUnmarshaler is
// true and the struct matches the top level object being unmarshaled.
func scalarunmarshalerHookFunc(opts *internal.UnmarshalOptions) mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from, to reflect.Value) (any, error) {
		if !to.CanAddr() {
			return from.Interface(), nil
		}

		// if from.Kind() == reflect.Struct ||
		// 	from.Kind() == reflect.Pointer && from.Elem().Kind() == reflect.Struct {
		// 	return from.Interface(), nil
		// }

		toPtr := to.Addr().Interface()

		unmarshaler, ok := toPtr.(ScalarUnmarshaler)
		if !ok {
			return from.Interface(), nil
		}

		if to.Addr().IsNil() {
			unmarshaler = reflect.New(to.Type()).Interface().(ScalarUnmarshaler)
		}

		resultVal := reflect.New(reflect.TypeOf(unmarshaler.ScalarType()))

		if err := internal.Decode(from.Interface(), resultVal.Interface(), *opts, false); err != nil {
			return nil, err
		}

		if err := unmarshaler.UnmarshalScalar(resultVal.Elem().Interface()); err != nil {
			return nil, err
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
