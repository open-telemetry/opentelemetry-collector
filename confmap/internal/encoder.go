// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import (
	"github.com/go-viper/mapstructure/v2"

	encoder "go.opentelemetry.io/collector/confmap/internal/mapstructure"
)

// EncoderConfig returns a default encoder.EncoderConfig that includes
// an EncodeHook that handles both TextMarshaller and Marshaler
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
