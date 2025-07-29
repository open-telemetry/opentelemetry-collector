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
		mo.AdditionalEncodeHookFuncs = append(mo.AdditionalEncodeHookFuncs, scalarmarshalerHookFunc())
	})
}

// ScalarUnmarshaler is an interface which may be implemented by wrapper types
// to customize their behavior when the type under the wrapper is a scalar value.
type ScalarMarshaler interface {
	// Unmarshal a Conf into the struct in a custom way.
	// The Conf for this specific component may be nil or empty if no config available.
	// This method should only be called by decoding hooks when calling Conf.Unmarshal.
	MarshalScalar() (string, error)
}

// Provides a mechanism for individual structs to define their own unmarshal logic,
// by implementing the Unmarshaler interface, unless skipTopLevelUnmarshaler is
// true and the struct matches the top level object being unmarshaled.
func scalarmarshalerHookFunc() mapstructure.DecodeHookFuncValue {
	return safeWrapDecodeHookFunc(func(from reflect.Value, _ reflect.Value) (any, error) {
		marshaler, ok := from.Interface().(ScalarMarshaler)
		if !ok {
			return from.Interface(), nil
		}

		return marshaler.MarshalScalar()

	})
}
