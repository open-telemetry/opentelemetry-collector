// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"encoding"
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
)

func WithScalarUnmarshaler() confmap.UnmarshalOption {
	return internal.UnmarshalOptionFunc(func(uo *internal.UnmarshalOptions) {
		uo.AdditionalDecodeHookFuncs = append(uo.AdditionalDecodeHookFuncs, scalarunmarshalerHookFunc())
	})
}

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar value.
type ScalarUnmarshaler interface {
	// Unmarshal a Conf into the struct in a custom way.
	// The Conf for this specific component may be nil or empty if no config available.
	// This method should only be called by decoding hooks when calling Conf.Unmarshal.
	UnmarshalScalar(val any) error

	ScalarType() any
}

// Provides a mechanism for individual structs to define their own unmarshal logic,
// by implementing the Unmarshaler interface, unless skipTopLevelUnmarshaler is
// true and the struct matches the top level object being unmarshaled.
func scalarunmarshalerHookFunc() mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from reflect.Value, to reflect.Value) (any, error) {
		if !to.CanAddr() {
			return from.Interface(), nil
		}

		toPtr := to.Addr().Interface()

		unmarshaler, ok := toPtr.(ScalarUnmarshaler)
		if !ok {
			return from.Interface(), nil
		}

		var v any
		tp := reflect.New(reflect.TypeOf(unmarshaler.ScalarType()))
		if tu, ok := tp.Interface().(encoding.TextUnmarshaler); ok {
			// Should we error out here?
			if str, ok := from.Interface().(string); ok {
				if err := tu.UnmarshalText([]byte(str)); err != nil {
					return nil, err
				}
				v = tp.Elem().Interface()
			}
		} else if from.CanConvert(tp.Elem().Type()) {
			from.Convert(tp.Elem().Type())
			v = from.Interface()
		} else if _, ok := from.Interface().(map[string]any); ok {
			v = tp.Elem().Interface()
		} else {
			v = from.Interface()
		}

		if err := unmarshaler.UnmarshalScalar(v); err != nil {
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
	return func(fromVal reflect.Value, toVal reflect.Value) (any, error) {
		if !fromVal.IsValid() {
			return nil, nil
		}
		return f(fromVal, toVal)
	}
}
