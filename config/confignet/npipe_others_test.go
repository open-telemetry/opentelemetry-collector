// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package confignet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpipeDialUnsupported(t *testing.T) {
	nac := &AddrConfig{
		Endpoint:  `\\.\pipe\test`,
		Transport: TransportTypeNpipe,
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "npipe transport is only supported on Windows")
}

func TestNpipeListenUnsupported(t *testing.T) {
	nas := &AddrConfig{
		Endpoint:  `\\.\pipe\test`,
		Transport: TransportTypeNpipe,
	}
	_, err := nas.Listen(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "npipe transport is only supported on Windows")
}
