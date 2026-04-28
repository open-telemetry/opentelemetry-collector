// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package confignet

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))

		err := removeStaleSocket(path)
		require.ErrorContains(t, err, "not a socket")
		// File should still exist.
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr)
	})

	t.Run("path is a stale socket", func(t *testing.T) {
		t.Parallel()
		//nolint:usetesting // short path needed for Unix socket limit
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
		require.NoError(t, err)
		// Socket file should be removed.
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})
}

func Test_unixListener_Close(t *testing.T) {
	t.Parallel()
	//nolint:usetesting // short path needed for Unix socket limit
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
	//nolint:usetesting // short path needed for Unix socket limit
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
	//nolint:usetesting // short path needed for Unix socket limit
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := filepath.Join(dir, "regular.txt")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))

	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	_, err = na.Listen(context.Background())
	assert.ErrorContains(t, err, "not a socket")
}

func TestAddrConfig_Listen_UnixSocketPermissions(t *testing.T) {
	t.Parallel()
	//nolint:usetesting // short path needed for Unix socket limit
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
	//nolint:usetesting // short path needed for Unix socket limit
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
