// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package confignet

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/mdlayher/vsock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVsockListenAndDial(t *testing.T) {
	if _, err := os.Stat("/dev/vsock"); err != nil {
		t.Skip("vsock device not available")
	}

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

	done := make(chan bool, 1)
	go func() {
		conn, errGo := ln.Accept()
		assert.NoError(t, errGo)
		buf := make([]byte, 10)
		var numChr int
		numChr, errGo = conn.Read(buf)
		assert.NoError(t, errGo)
		assert.Equal(t, "test", string(buf[:numChr]))
		assert.NoError(t, conn.Close())
		done <- true
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
