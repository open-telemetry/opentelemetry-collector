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

// vsockDialContext is a variable so tests can substitute a dialer that blocks.
var vsockDialContext = func(ctx context.Context, cid, port uint32) (net.Conn, error) {
	conn, err := vsock.DialContext(ctx, cid, port, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return vsockDialContext(ctx, cid, port)
}

func listenVsock(endpoint string) (net.Listener, error) {
	cid, port, err := parseVsockEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	ln, err := vsock.ListenContextID(cid, port, nil)
	if err != nil {
		return nil, err
	}
	return ln, nil
}
