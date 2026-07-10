// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package confignet // import "go.opentelemetry.io/collector/config/confignet"

import (
	"context"
	"net"
	"time"

	"github.com/mdlayher/vsock"
)

func dialVsock(ctx context.Context, endpoint string, timeout time.Duration) (net.Conn, error) {
	cid, port, err := parseVsockEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	if timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	// vsock.Dial does not accept a context; check for cancellation before dialing.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return vsock.Dial(cid, port, nil)
}

func listenVsock(endpoint string) (net.Listener, error) {
	cid, port, err := parseVsockEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	return vsock.ListenContextID(cid, port, nil)
}
