// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !windows

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"errors"
)

func openFunction(path string) func() (transport.TPMCloser, error) {
	return func() (transport.TPMCloser, error) {
		return nil, errors.New("TPM is not supported on this platform")
	}
}
