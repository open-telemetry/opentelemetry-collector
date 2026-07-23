// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package confignet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVsockDialUnsupported(t *testing.T) {
	nac := &AddrConfig{
		Endpoint:  "2:8080",
		Transport: TransportTypeVsock,
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vsock transport is only supported on Linux")
}

func TestVsockListenUnsupported(t *testing.T) {
	nas := &AddrConfig{
		Endpoint:  "2:8080",
		Transport: TransportTypeVsock,
	}
	_, err := nas.Listen(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vsock transport is only supported on Linux")
}
