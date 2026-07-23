// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
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
	var cfg *winio.PipeConfig
	if securityDescriptor != "" {
		cfg = &winio.PipeConfig{SecurityDescriptor: securityDescriptor}
	}
	return winio.ListenPipe(endpoint, cfg)
}
