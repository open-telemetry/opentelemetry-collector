// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !windows

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"errors"

	"github.com/google/go-tpm/tpm2/transport"
)

// for testing
var tpmSimulator transport.TPMCloser

func openTPM(path string) func() (transport.TPMCloser, error) {
	return func() (transport.TPMCloser, error) {
		if path == "simulator" {
			return tpmSimulator, nil
		}
		return nil, errors.New("TPM is not supported on this platform")
	}
}
