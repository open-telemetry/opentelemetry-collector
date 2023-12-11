// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_SetOrder(t *testing.T) {
	t.Run("TestNetAddrTimeout", testNetAddrTimeout)
	t.Run("TestTcpAddrTimeout", testTcpAddrTimeout)
	t.Run("TestNetAddr", testNetAddr)
	t.Run("testTcpAddr", testTcpAddr)
}

func testNetAddrTimeout(t *testing.T) {
	nac := &NetAddr{
		Endpoint:      "localhost:0",
		Transport:     "tcp",
		DialerTimeout: time.Millisecond,
	}
	_, err := nac.Dial()
	assert.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func testTcpAddrTimeout(t *testing.T) {
	nac := &TCPAddr{
		Endpoint:      "localhost:0",
		DialerTimeout: time.Millisecond,
	}
	_, err := nac.Dial()
	assert.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func testNetAddr(t *testing.T) {
	nas := &NetAddr{
		Endpoint:  "localhost:0",
		Transport: "tcp",
	}
	ln, err := nas.Listen()
	assert.NoError(t, err)
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

	nac := &NetAddr{
		Endpoint:  ln.Addr().String(),
		Transport: "tcp",
	}
	var conn net.Conn
	conn, err = nac.Dial()
	assert.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}

func testTcpAddr(t *testing.T) {
	nas := &TCPAddr{
		Endpoint: "localhost:0",
	}
	ln, err := nas.Listen()
	assert.NoError(t, err)
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

	nac := &TCPAddr{
		Endpoint: ln.Addr().String(),
	}
	var conn net.Conn
	conn, err = nac.Dial()
	assert.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}
