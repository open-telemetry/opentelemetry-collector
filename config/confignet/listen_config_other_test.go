// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux && !darwin && !freebsd && !dragonfly && !openbsd && !netbsd

package confignet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddrConfigReusePortUnsupported(t *testing.T) {
	na := AddrConfig{Endpoint: "127.0.0.1:0", Transport: TransportTypeTCP, ReusePort: true}
	require.EqualError(t, na.Validate(), "reuse_port is not supported on this platform")
	_, err := na.Listen(t.Context())
	require.EqualError(t, err, "reuse_port is not supported on this platform")
}
