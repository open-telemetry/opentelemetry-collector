// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

import "github.com/go-viper/mapstructure/v2"

type UnmarshalOption interface {
	apply(*UnmarshalOptions)
}

// UnmarshalOptions is used by (*Conf).Unmarshal to toggle unmarshaling settings.
// It is in the `internal` package so experimental options can be added in xconfmap.
type UnmarshalOptions struct {
	IgnoreUnused              bool
	ForceUnmarshaler          bool
	AdditionalDecodeHookFuncs []mapstructure.DecodeHookFunc
}

type UnmarshalOptionFunc func(*UnmarshalOptions)

func (fn UnmarshalOptionFunc) apply(set *UnmarshalOptions) {
	fn(set)
}

func ApplyUnmarshalOptions(set *UnmarshalOptions, opts []UnmarshalOption) *UnmarshalOptions {
	if set == nil {
		set = &UnmarshalOptions{}
	}
	for _, opt := range opts {
		opt.apply(set)
	}
	return set
}
