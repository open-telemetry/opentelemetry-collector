// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

// TPMConfig defines trusted platform module configuration for storing TLS keys.
type TPMConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// The path to the TPM device or Unix domain socket.
	// For instance /dev/tpm0 or /dev/tpmrm0.
	Path      string `mapstructure:"path"`
	OwnerAuth string `mapstructure:"owner_auth"`
	Auth      string `mapstructure:"auth"`
	// prevent unkeyed literal initialization
	_ struct{}
}
