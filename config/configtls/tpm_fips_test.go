// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build requirefips

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTPM_notSupportedInFIPSMode(t *testing.T) {
	tlsCfg := Config{
		CertPem: "anything",
		KeyPem:  "anything",
		TPMConfig: TPMConfig{
			Enabled: true,
			Path:    "/dev/tpm0",
		},
	}
	_, err := tlsCfg.loadCertificate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TPM key loading is not supported in FIPS mode")
}
