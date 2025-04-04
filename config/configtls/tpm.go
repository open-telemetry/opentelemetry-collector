// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	tpmkeyfile "github.com/foxboron/go-tpm-keyfiles"
	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpmutil"
)

// TPMConfig defines trusted platform module configuration for storing TLS keys.
type TPMConfig struct {
	// The path to the TPM device or Unix domain socket.
	// For instance /dev/tpm0 or /dev/tpmrm0.
	Path      string `mapstructure:"path"`
	OwnerAuth string `mapstructure:"owner_auth"`
	Auth      string `mapstructure:"auth"`
}

func (c TPMConfig) tpmCertificate(keyPem []byte, certPem []byte, openTPM func() (transport.TPMCloser, error)) (tls.Certificate, error) {
	tpm, err := openTPM()
	if err != nil {
		return tls.Certificate{}, err
	}

	tpmKey, err := tpmkeyfile.Decode(keyPem)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load TPM key: %w", err)
	}

	tpmSigner, err := c.signer(tpm, tpmKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load TPM signer: %w", err)
	}

	certDER, _ := pem.Decode(certPem)
	x509Cert, err := x509.ParseCertificate(certDER.Bytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{x509Cert.Raw},
		Leaf:        x509Cert,
		PrivateKey:  tpmSigner,
	}, nil
}

func (c TPMConfig) open() (transport.TPMCloser, error) {
	if c.Path == "" {
		return nil, errors.New("TPM path is not set")
	}
	tpm, err := tpmutil.OpenTPM(c.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open TPM (%s): %w", c.Path, err)
	}
	return transport.FromReadWriteCloser(tpm), nil
}

func (c TPMConfig) signer(tpm transport.TPMCloser, tss2key *tpmkeyfile.TPMKey) (crypto.Signer, error) {
	signer, err := tss2key.Signer(tpm, []byte(c.OwnerAuth), []byte(c.Auth))
	if err != nil {
		return nil, err
	}
	return signer, err
}
