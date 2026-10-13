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

// vsockDial is a variable so tests can substitute a dialer that blocks.
var vsockDial = func(cid, port uint32) (net.Conn, error) {
	conn, err := vsock.Dial(cid, port, nil)
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

	// vsock.Dial does not accept a context, so run it in a goroutine to honor
	// the timeout and cancellation.
	// TODO: Replace this wrapper with vsock.DialContext once the upstream
	// package supports it (https://github.com/mdlayher/vsock/pull/78).
	type dialResult struct {
		conn net.Conn
		err  error
	}
	resCh := make(chan dialResult, 1)
	go func() {
		conn, dialErr := vsockDial(cid, port)
		resCh <- dialResult{conn: conn, err: dialErr}
	}()

	select {
	case res := <-resCh:
		return res.conn, res.err
	case <-ctx.Done():
		// Close any connection that is established after the context is done.
		go func() {
			if res := <-resCh; res.err == nil {
				_ = res.conn.Close()
			}
		}()
		return nil, ctx.Err()
	}
}

func listenVsock(endpoint string) (net.Listener, error) {
	cid, port, err := parseVsockEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	return vsock.ListenContextID(cid, port, nil)
}
