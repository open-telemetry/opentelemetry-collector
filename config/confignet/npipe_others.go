// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"errors"
	"net"
	"time"
)

var errNpipeUnsupported = errors.New("npipe transport is only supported on Windows")

func dialNpipe(_ context.Context, _ string, _ time.Duration) (net.Conn, error) {
	return nil, errNpipeUnsupported
}

func listenNpipe(_, _ string) (net.Listener, error) {
	return nil, errNpipeUnsupported
}

// validateNpipeSecurityDescriptor is a no-op on non-Windows platforms: parsing an SDDL
// string requires the Windows API.
func validateNpipeSecurityDescriptor(_ string) error {
	return nil
}
