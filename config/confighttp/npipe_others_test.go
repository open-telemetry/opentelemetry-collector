// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package confighttp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confignet"
)

func testHttpServerNpipeTransport(t *testing.T) {
	t.Run("npipe", func(t *testing.T) {
		sc := &ServerConfig{
			NetAddr: confignet.AddrConfig{
				Endpoint:  `\\.\pipe\otel-test-confighttp`,
				Transport: confignet.TransportTypeNpipe,
			},
		}
		_, err := sc.ToListener(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "npipe transport is only supported on Windows")
	})
}
