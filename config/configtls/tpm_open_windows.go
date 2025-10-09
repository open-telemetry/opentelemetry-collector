// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"fmt"

	"github.com/google/go-tpm/tpm2/transport"
	"github.com/google/go-tpm/tpmutil"
)

func openTPM(_ string) func() (transport.TPMCloser, error) {
	return func() (transport.TPMCloser, error) {
		tpm, err := tpmutil.OpenTPM()
		if err != nil {
			return nil, fmt.Errorf("failed to open TPM: %w", err)
		}
		return transport.FromReadWriteCloser(tpm), nil
	}
}
