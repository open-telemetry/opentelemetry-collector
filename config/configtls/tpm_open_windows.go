// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package configtls // import "go.opentelemetry.io/collector/config/configtls"

func openFunction(path string) func() (transport.TPMCloser, error) {
	return func() (transport.TPMCloser, error) {
		tpm, err := tpmutil.OpenTPM()
		if err != nil {
			return nil, fmt.Errorf("failed to open TPM (%s): %w", path, err)
		}
		return transport.FromReadWriteCloser(tpm), nil
	}
}
