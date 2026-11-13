// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux || darwin || freebsd || dragonfly || openbsd || netbsd

package confignet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddrConfigReusePort(t *testing.T) {
	first := AddrConfig{Endpoint: "127.0.0.1:0", Transport: TransportTypeTCP, ReusePort: true}
	require.NoError(t, first.Validate())
	ln1, err := first.Listen(t.Context())
	require.NoError(t, err)
	defer ln1.Close()

	second := AddrConfig{Endpoint: ln1.Addr().String(), Transport: TransportTypeTCP, ReusePort: true}
	ln2, err := second.Listen(t.Context())
	require.NoError(t, err)
	require.NoError(t, ln2.Close())

	second.ReusePort = false
	_, err = second.Listen(t.Context())
	assert.Error(t, err)
}
