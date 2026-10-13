// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package confignet

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVsockDialErrors(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name     string
		ctx      context.Context
		endpoint string
		timeout  time.Duration
		errIs    error
	}{
		{name: "invalid endpoint", ctx: context.Background(), endpoint: "invalid"},
		{name: "canceled context", ctx: canceledCtx, endpoint: "2:8080", errIs: context.Canceled},
		{name: "canceled context with timeout", ctx: canceledCtx, endpoint: "2:8080", timeout: time.Second, errIs: context.Canceled},
		// VMADDR_CID_ANY (4294967295) is not a valid remote address.
		{name: "unreachable CID", ctx: context.Background(), endpoint: "4294967295:8080"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nac := &AddrConfig{
				Endpoint:     tt.endpoint,
				Transport:    TransportTypeVsock,
				DialerConfig: DialerConfig{Timeout: tt.timeout},
			}
			conn, err := nac.Dial(tt.ctx)
			require.Error(t, err)
			assert.Nil(t, conn)
			if tt.errIs != nil {
				assert.ErrorIs(t, err, tt.errIs)
			}
		})
	}
}

func TestVsockDialHonorsContext(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		cancel  bool
		errIs   error
	}{
		{name: "timeout", timeout: 10 * time.Millisecond, errIs: context.DeadlineExceeded},
		{name: "canceled mid-dial", cancel: true, errIs: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := make(chan struct{})
			dialStarted := make(chan struct{})
			client, server := net.Pipe()
			t.Cleanup(func() { _ = server.Close() })
			lateConn := &closeTrackingConn{Conn: client, closed: make(chan struct{})}

			origDial := vsockDial
			t.Cleanup(func() { vsockDial = origDial })
			vsockDial = func(uint32, uint32) (net.Conn, error) {
				close(dialStarted)
				<-release
				return lateConn, nil
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.cancel {
				go func() {
					<-dialStarted
					cancel()
				}()
			}

			nac := &AddrConfig{
				Endpoint:     "2:8080",
				Transport:    TransportTypeVsock,
				DialerConfig: DialerConfig{Timeout: tt.timeout},
			}
			conn, err := nac.Dial(ctx)
			require.ErrorIs(t, err, tt.errIs)
			assert.Nil(t, conn)

			// A connection established after the context is done must be closed.
			close(release)
			select {
			case <-lateConn.closed:
			case <-time.After(5 * time.Second):
				t.Fatal("connection established after the context was done was not closed")
			}
		})
	}
}

type closeTrackingConn struct {
	net.Conn
	closed chan struct{}
}

func (c *closeTrackingConn) Close() error {
	close(c.closed)
	return c.Conn.Close()
}

func TestVsockListenErrors(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{name: "invalid endpoint", endpoint: "invalid"},
		// The hypervisor CID is never a local address.
		{name: "non-local CID", endpoint: fmt.Sprintf("%d:8080", vsock.Hypervisor)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nas := &AddrConfig{
				Endpoint:  tt.endpoint,
				Transport: TransportTypeVsock,
			}
			ln, err := nas.Listen(context.Background())
			require.Error(t, err)
			assert.Nil(t, ln)
		})
	}
}

func TestVsockListenAndDial(t *testing.T) {
	// Don't gate on /dev/vsock: it only appears once the vsock module is
	// loaded, and the kernel loads vsock and vsock_loopback on demand when
	// the first AF_VSOCK socket is created.
	//
	// Use the local loopback CID (vsock.Local = 1) for self-connections,
	// which requires Linux 5.6+. Port 0 lets the kernel assign a free port.
	listenEndpoint := fmt.Sprintf("%d:0", vsock.Local)
	nas := &AddrConfig{
		Endpoint:  listenEndpoint,
		Transport: TransportTypeVsock,
	}
	ln, err := nas.Listen(context.Background())
	if err != nil {
		t.Skipf("vsock local loopback not available: %v", err)
	}
	t.Cleanup(func() {
		assert.NoError(t, ln.Close())
	})

	// Retrieve the kernel-assigned port from the listener address.
	vsockAddr, ok := ln.Addr().(*vsock.Addr)
	require.True(t, ok, "expected *vsock.Addr from listener")
	dialEndpoint := fmt.Sprintf("%d:%d", vsockAddr.ContextID, vsockAddr.Port)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, errGo := ln.Accept()
		if !assert.NoError(t, errGo) {
			return
		}
		buf := make([]byte, 10)
		var numChr int
		numChr, errGo = conn.Read(buf)
		assert.NoError(t, errGo)
		assert.Equal(t, "test", string(buf[:numChr]))
		assert.NoError(t, conn.Close())
	}()

	nac := &AddrConfig{
		Endpoint:  dialEndpoint,
		Transport: TransportTypeVsock,
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
}
