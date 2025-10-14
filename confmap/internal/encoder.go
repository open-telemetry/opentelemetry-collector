// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import (
	"reflect"

	"github.com/go-viper/mapstructure/v2"

	encoder "go.opentelemetry.io/collector/confmap/internal/mapstructure"
)

// EncoderConfig returns a default encoder.EncoderConfig that includes
// an EncodeHook that handles both TextMarshaler and Marshaler
// interfaces.
func EncoderConfig(rawVal any, set MarshalOptions) *encoder.EncoderConfig {
	hooks := []mapstructure.DecodeHookFunc{
		encoder.YamlMarshalerHookFunc(),
		encoder.TextMarshalerHookFunc(),
	}

	if set.ScalarMarshalingEncodeHookFunc != nil {
		hooks = append(hooks, set.ScalarMarshalingEncodeHookFunc)
	}

	// This hook must come after the scalar marshaling hook, if present.
	hooks = append(hooks, marshalerHookFunc(rawVal))

	return &encoder.EncoderConfig{
		EncodeHook: mapstructure.ComposeDecodeHookFunc(hooks...),
	}
}

// marshalerHookFunc returns a DecodeHookFuncValue that checks structs that aren't
// the original to see if they implement the Marshaler interface.
func marshalerHookFunc(orig any) mapstructure.DecodeHookFuncValue {
	origType := reflect.TypeOf(orig)
	return safeWrapDecodeHookFunc(func(from, _ reflect.Value) (any, error) {
		if from.Kind() != reflect.Struct {
			return from.Interface(), nil
		}

		// ignore original to avoid infinite loop.
		if from.Type() == origType && reflect.DeepEqual(from.Interface(), orig) {
			return from.Interface(), nil
		}
		marshaler, ok := from.Interface().(Marshaler)
		if !ok {
			return from.Interface(), nil
		}
		conf := NewFromStringMap(nil)
		if err := marshaler.Marshal(conf); err != nil {
			return nil, err
		}

		stringMap := conf.ToStringMap()
		if stringMap == nil {
			// If conf is still nil after marshaling, we want to encode it as an untyped nil
			// instead of a map-typed nil. This ensures the value is a proper null value
			// in the final marshaled output instead of an empty map. We hit this case
			// when marshaling wrapper structs that have no direct representation
			// in the marshaled output that aren't tagged with "squash" on the fields
			// they're used on.
			return nil, nil
		}
		return stringMap, nil
	})
}
