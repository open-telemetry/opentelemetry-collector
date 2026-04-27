// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confignet

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultDialerConfig(t *testing.T) {
	expectedDialerConfig := DialerConfig{}
	dialerConfig := NewDefaultDialerConfig()
	require.Equal(t, expectedDialerConfig, dialerConfig)
}

func TestNewDefaultAddrConfig(t *testing.T) {
	expectedAddrConfig := AddrConfig{}
	addrConfig := NewDefaultAddrConfig()
	require.Equal(t, expectedAddrConfig, addrConfig)
}

func TestNewDefaultTCPAddrConfig(t *testing.T) {
	expectedTCPAddrConfig := TCPAddrConfig{}
	tcpAddrconfig := NewDefaultTCPAddrConfig()
	require.Equal(t, expectedTCPAddrConfig, tcpAddrconfig)
}

func TestAddrConfigTimeout(t *testing.T) {
	nac := &AddrConfig{
		Endpoint:  "localhost:0",
		Transport: TransportTypeTCP,
		DialerConfig: DialerConfig{
			Timeout: -1 * time.Second,
		},
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func TestTCPAddrConfigTimeout(t *testing.T) {
	nac := &TCPAddrConfig{
		Endpoint: "localhost:0",
		DialerConfig: DialerConfig{
			Timeout: -1 * time.Second,
		},
	}
	_, err := nac.Dial(context.Background())
	require.Error(t, err)
	var netErr net.Error
	if errors.As(err, &netErr) {
		assert.True(t, netErr.Timeout())
	} else {
		assert.Fail(t, "error should be a net.Error")
	}
}

func TestAddrConfig(t *testing.T) {
	nas := &AddrConfig{
		Endpoint:  "localhost:0",
		Transport: TransportTypeTCP,
	}
	ln, err := nas.Listen(context.Background())
	require.NoError(t, err)
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
		Endpoint:  ln.Addr().String(),
		Transport: TransportTypeTCP,
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}

func Test_NetAddr_Validate(t *testing.T) {
	na := &AddrConfig{
		Transport: TransportTypeTCP,
	}
	require.NoError(t, na.Validate())

	na = &AddrConfig{
		Transport: transportTypeEmpty,
	}
	require.Error(t, na.Validate())

	na = &AddrConfig{
		Transport: "random string",
	}
	assert.Error(t, na.Validate())
}

func Test_NetAddr_Validate_Npipe(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{name: "local pipe", endpoint: `\\.\pipe\test`, wantErr: false},
		{name: "remote pipe", endpoint: `\\server\pipe\myapp`, wantErr: false},
		{name: "mixed case pipe component", endpoint: `\\.\PIPE\test`, wantErr: false},
		{name: "special chars in name", endpoint: `\\.\pipe\my-pipe.v2`, wantErr: false},
		{name: "empty", endpoint: "", wantErr: true},
		{name: "missing leading backslashes", endpoint: `\pipe\test`, wantErr: true},
		{name: "missing server name", endpoint: `\\\pipe\test`, wantErr: true},
		{name: "missing pipe component", endpoint: `\\.\test`, wantErr: true},
		{name: "empty pipe name", endpoint: `\\.\pipe\`, wantErr: true},
		{name: "backslash in pipe name", endpoint: `\\.\pipe\a\b`, wantErr: true},
		{name: "too long", endpoint: `\\.\pipe\` + strings.Repeat("a", 249), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			na := &AddrConfig{
				Transport: TransportTypeNpipe,
				Endpoint:  tt.endpoint,
			}
			err := na.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTCPAddrConfig(t *testing.T) {
	nas := &TCPAddrConfig{
		Endpoint: "localhost:0",
	}
	ln, err := nas.Listen(context.Background())
	require.NoError(t, err)
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

	nac := &TCPAddrConfig{
		Endpoint: ln.Addr().String(),
	}
	var conn net.Conn
	conn, err = nac.Dial(context.Background())
	require.NoError(t, err)
	_, err = conn.Write([]byte("test"))
	assert.NoError(t, err)
	assert.NoError(t, conn.Close())
	<-done
	assert.NoError(t, ln.Close())
}

func Test_TransportType_UnmarshalText(t *testing.T) {
	var tt TransportType
	err := tt.UnmarshalText([]byte("tcp"))
	require.NoError(t, err)
	assert.Equal(t, TransportTypeTCP, tt)
	err = tt.UnmarshalText([]byte("npipe"))
	require.NoError(t, err)
	assert.Equal(t, TransportTypeNpipe, tt)
	err = tt.UnmarshalText([]byte("invalid"))
	require.Error(t, err)
}

func Test_removeStaleSocket(t *testing.T) {
	t.Parallel()
	t.Run("path does not exist", func(t *testing.T) {
		t.Parallel()
		err := removeStaleSocket(filepath.Join(t.TempDir(), "nonexistent.sock"))
		assert.NoError(t, err)
	})

	t.Run("path is a regular file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "regular.txt")
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))

		err := removeStaleSocket(path)
		assert.ErrorContains(t, err, "not a socket")
		// File should still exist.
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr)
	})

	t.Run("path is a stale socket", func(t *testing.T) {
		t.Parallel()
		// Use os.MkdirTemp with an empty base so the OS chooses a short path,
		// avoiding the 104-character Unix socket path limit on macOS/BSD.
		dir, err := os.MkdirTemp("", "confignet-sock-test-*")
		require.NoError(t, err)
		t.Cleanup(func() { os.RemoveAll(dir) })
		path := filepath.Join(dir, "stale.sock")
		// Create a socket file directly via syscall so it is not auto-removed
		// when closed (Go's net.Listener.Close removes socket files on some OSes).
		fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
		require.NoError(t, err)
		err = syscall.Bind(fd, &syscall.SockaddrUnix{Name: path})
		require.NoError(t, err)
		require.NoError(t, syscall.Close(fd))
		// Socket file should still exist after closing the fd.
		_, err = os.Stat(path)
		require.NoError(t, err)

		err = removeStaleSocket(path)
		assert.NoError(t, err)
		// Socket file should be removed.
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})

}

func Test_isAbstractSocket(t *testing.T) {
	t.Parallel()
	assert.True(t, isAbstractSocket("@abstract"))
	assert.False(t, isAbstractSocket("/var/run/test.sock"))
	assert.False(t, isAbstractSocket(""))
}

func Test_unixListener_Close(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "listener.sock")

	ln, err := net.Listen("unix", path)
	require.NoError(t, err)

	// Socket file exists after listen.
	_, err = os.Stat(path)
	require.NoError(t, err)

	uln := &unixListener{Listener: ln, path: path}
	require.NoError(t, uln.Close())

	// Socket file should be removed after close.
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestAddrConfig_Listen_UnixRemovesStaleSocket(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "stale.sock")

	// Create a stale socket using syscall (macOS removes socket on net.Listener.Close).
	fd, sysErr := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, sysErr)
	require.NoError(t, syscall.Bind(fd, &syscall.SockaddrUnix{Name: path}))
	require.NoError(t, syscall.Close(fd))
	_, sysErr = os.Stat(path)
	require.NoError(t, sysErr)

	// Listen should succeed despite stale socket.
	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	ln, err := na.Listen(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ln.Close()) })
}

func TestAddrConfig_Listen_UnixRefusesToRemoveNonSocket(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "regular.txt")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o644))

	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	_, err = na.Listen(context.Background())
	assert.ErrorContains(t, err, "not a socket")
}

func TestAddrConfig_Listen_UnixSocketPermissions(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "perms.sock")

	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	ln, err := na.Listen(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, ln.Close()) })

	fi, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, socketFileMode|os.ModeSocket, fi.Mode())
}

func TestAddrConfig_Listen_UnixSocketCloseRemovesFile(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "cleanup.sock")

	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	ln, err := na.Listen(context.Background())
	require.NoError(t, err)

	_, err = os.Stat(path)
	require.NoError(t, err)

	require.NoError(t, ln.Close())

	// Socket file should be removed after close.
	_, err = os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func Test_AddrConfig_isUnixTransport(t *testing.T) {
	t.Parallel()
	tests := []struct {
		transport TransportType
		expected  bool
	}{
		{TransportTypeTCP, false},
		{TransportTypeTCP4, false},
		{TransportTypeTCP6, false},
		{TransportTypeUDP, false},
		{TransportTypeUDP4, false},
		{TransportTypeUDP6, false},
		{TransportTypeIP, false},
		{TransportTypeIP4, false},
		{TransportTypeIP6, false},
		{TransportTypeUnix, true},
		{TransportTypeUnixgram, true},
		{TransportTypeUnixPacket, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.transport), func(t *testing.T) {
			t.Parallel()
			na := &AddrConfig{Transport: tt.transport}
			assert.Equal(t, tt.expected, na.isUnixTransport())
		})
	}
}
