// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/tls"
	"errors"

	"github.com/google/go-tpm/tpm2/transport"
)

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

// tpmCertificate is not supported in FIPS mode because the TPM key-file format
// relies on ChaCha20-Poly1305 for key wrapping, which is not a FIPS-approved algorithm.
func (c TPMConfig) tpmCertificate(_ []byte, _ []byte, _ func() (transport.TPMCloser, error)) (tls.Certificate, error) {
	return tls.Certificate{}, errors.New("TPM key loading is not supported in FIPS mode")
}
