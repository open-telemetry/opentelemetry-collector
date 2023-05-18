// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ballastextension // import "go.opentelemetry.io/collector/extension/ballastextension"

import (
	"errors"
)

// Config has the configuration for the ballast extension.
type Config struct {
	// SizeMiB is the size, in MiB, of the memory ballast
	// to be created for this process.
	SizeMiB uint64 `mapstructure:"size_mib"`

	// SizeInPercentage is the maximum amount of memory ballast, in %, targeted to be
	// allocated. The fixed memory settings SizeMiB has a higher precedence.
	SizeInPercentage uint64 `mapstructure:"size_in_percentage"`
}

// Validate checks if the extension configuration is valid
func (cfg *Config) Validate() error {
	// no need to validate less than 0 case for uint64
	if cfg.SizeInPercentage > 100 {
		return errors.New("size_in_percentage is not in range 0 to 100")
	}
	return nil
}
