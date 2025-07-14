// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build notpm

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"crypto/tls"
	"errors"

	"github.com/google/go-tpm/tpm2/transport"
)

var errorNoTPM = errors.New("package was not compiled with TPM support")

func (c TPMConfig) tpmCertificate(keyPem []byte, certPem []byte, openTPM func() (transport.TPMCloser, error)) (tls.Certificate, error) {
	return tls.Certificate{}, errorNoTPM
}
