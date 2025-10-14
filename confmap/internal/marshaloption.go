// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import "github.com/go-viper/mapstructure/v2"

type MarshalOption interface {
	apply(*MarshalOptions)
}

// MarshalOptions is used by (*Conf).Marshal to toggle unmarshaling settings.
// It is in the `internal` package so experimental options can be added in xconfmap.
type MarshalOptions struct {
	ScalarMarshalingEncodeHookFunc mapstructure.DecodeHookFunc
}

type MarshalOptionFunc func(*MarshalOptions)

func (fn MarshalOptionFunc) apply(set *MarshalOptions) {
	fn(set)
}

// ApplyMarshalOption simply calls (MarshalOption).apply. This function allows us
// to keep the `apply“ function private and therefore keep `confmap.MarshalOption`
// without any exported methods.
func ApplyMarshalOption(mo MarshalOption, set *MarshalOptions) {
	mo.apply(set)
}
