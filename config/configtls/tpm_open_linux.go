// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"errors"
	"fmt"

	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpmutil"
)

// for testing
var tpmSimulator transport.TPMCloser

func openTPM(path string) func() (transport.TPMCloser, error) {
	return func() (transport.TPMCloser, error) {
		if path == "" {
			return nil, errors.New("TPM path is not set")
		}
		if path == "simulator" {
			return tpmSimulator, nil
		}

		tpm, err := tpmutil.OpenTPM(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open TPM (%s): %w", path, err)
		}
		return transport.FromReadWriteCloser(tpm), nil
	}
}
