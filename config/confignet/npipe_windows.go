// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Microsoft/go-winio"
)

func dialNpipe(ctx context.Context, endpoint string, timeout time.Duration) (net.Conn, error) {
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return winio.DialPipeContext(ctx, endpoint)
}

func listenNpipe(endpoint, securityDescriptor string) (net.Listener, error) {
	return winio.ListenPipe(endpoint, &winio.PipeConfig{SecurityDescriptor: securityDescriptor})
}

// validateNpipeSecurityDescriptor checks that the given SDDL string can be converted
// into a Windows security descriptor. An empty string is valid and means that the
// Windows default named pipe DACL is used.
func validateNpipeSecurityDescriptor(securityDescriptor string) error {
	if securityDescriptor == "" {
		return nil
	}
	if _, err := winio.SddlToSecurityDescriptor(securityDescriptor); err != nil {
		return fmt.Errorf("invalid named pipe security descriptor: %w", err)
	}
	return nil
}
