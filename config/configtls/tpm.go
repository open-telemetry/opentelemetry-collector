// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	tpmkeyfile "github.com/foxboron/go-tpm-keyfiles"
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

func (c TPMConfig) tpmCertificate(keyPem, certPem []byte, openTPM func() (transport.TPMCloser, error)) (tls.Certificate, error) {
	tpm, err := openTPM()
	if err != nil {
		return tls.Certificate{}, err
	}

	tpmKey, err := tpmkeyfile.Decode(keyPem)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load TPM key: %w", err)
	}

	tpmSigner, err := tpmKey.Signer(tpm, []byte(c.OwnerAuth), []byte(c.Auth))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load TPM signer: %w", err)
	}

	certDER, _ := pem.Decode(certPem)
	x509Cert, err := x509.ParseCertificate(certDER.Bytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{
			x509Cert.Raw,
		},
		Leaf:       x509Cert,
		PrivateKey: tpmSigner,
	}, nil
}
