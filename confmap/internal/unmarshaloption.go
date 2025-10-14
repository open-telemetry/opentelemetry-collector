// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/confmap/internal"

<<<<<<< HEAD
import "github.com/go-viper/mapstructure/v2"

=======
>>>>>>> upstream/main
type UnmarshalOption interface {
	apply(*UnmarshalOptions)
}

// UnmarshalOptions is used by (*Conf).Unmarshal to toggle unmarshaling settings.
// It is in the `internal` package so experimental options can be added in xconfmap.
type UnmarshalOptions struct {
<<<<<<< HEAD
	IgnoreUnused              bool
	AdditionalDecodeHookFuncs []mapstructure.DecodeHookFunc
=======
	IgnoreUnused bool
>>>>>>> upstream/main
}

type UnmarshalOptionFunc func(*UnmarshalOptions)

func (fn UnmarshalOptionFunc) apply(set *UnmarshalOptions) {
	fn(set)
}
<<<<<<< HEAD

// Apply Option simply calls (UnmarshalOption).apply. This function allows us
// to keep the `apply“ function private and therefore keep `confmap.UnmarshalOption`
// without any exported methods.
func ApplyUnmarshalOption(uo UnmarshalOption, set *UnmarshalOptions) {
	uo.apply(set)
}
=======
>>>>>>> upstream/main
