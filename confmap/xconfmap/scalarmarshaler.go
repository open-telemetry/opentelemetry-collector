// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconfmap

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
)

func WithScalarMarshaler() confmap.MarshalOption {
	return internal.MarshalOptionFunc(func(mo *internal.MarshalOptions) {
		mo.ScalarMarshalingEncodeHookFunc = scalarmarshalerHookFunc(*mo)
	})
}

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar value.
type ScalarMarshaler interface {
	// GetScalarValue gets the scalar value and marshals it.
	// The struct implementing the interface is free to
	// do any pre-processing to the value as part of marshaling.
	GetScalarValue() (any, error)
}

// Provides a mechanism for individual structs to define their own unmarshal logic,
// by implementing the Unmarshaler interface, unless skipTopLevelUnmarshaler is
// true and the struct matches the top level object being unmarshaled.
func scalarmarshalerHookFunc(mo internal.MarshalOptions) mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from reflect.Value, _ reflect.Value) (any, error) {
		marshaler, ok := from.Interface().(ScalarMarshaler)
		if !ok {
			return from.Interface(), nil
		}

		v, err := marshaler.GetScalarValue()
		if err != nil {
			return nil, err
		}

		return internal.Encode(v, mo)
	})
}
