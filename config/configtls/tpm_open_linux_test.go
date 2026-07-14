// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package configtls // import "go.opentelemetry.io/collector/config/configtls"

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTPM_open(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "app.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer listener.Close()

	tests := []struct {
		path string
		err  string
	}{
		{
			path: "",
			err:  "TPM path is not set",
		},
		{
			path: "/foo",
			err:  "failed to open TPM (/foo): stat /foo: no such file or directory",
		},
		{
			path: socketPath,
		},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			tpm, openErr := openTPM(test.path)()
			if test.err != "" {
				require.Nil(t, tpm)
				assert.Equal(t, test.err, openErr.Error())
			} else {
				require.NoError(t, openErr)
				require.NotNil(t, tpm)
				tpm.Close()
			}
		})
	}
}
