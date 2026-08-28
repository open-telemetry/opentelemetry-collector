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

// tempSocketDir creates a temp directory short enough to stay under the
// Unix socket path length limit and registers cleanup.
func tempSocketDir(t *testing.T) string {
	t.Helper()
	//nolint:usetesting // short path needed for Unix socket limit
	dir, err := os.MkdirTemp("", "confignet-test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// createStaleSocket creates a socket file directly via syscall, bypassing
// net.Listen, so it is not auto-removed when closed (Go's net.Listener.Close
// removes socket files on some OSes) and remains on disk as a "stale" socket.
func createStaleSocket(t *testing.T, path string) {
	t.Helper()
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	require.NoError(t, err)
	require.NoError(t, syscall.Bind(fd, &syscall.SockaddrUnix{Name: path}))
	require.NoError(t, syscall.Close(fd))
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
		require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))

		err := removeStaleSocket(path)
		require.ErrorContains(t, err, "not a socket")
		// File should still exist.
		_, statErr := os.Stat(path)
		assert.NoError(t, statErr)
	})

	t.Run("path is a stale socket", func(t *testing.T) {
		t.Parallel()
		dir := tempSocketDir(t)
		path := filepath.Join(dir, "stale.sock")
		createStaleSocket(t, path)
		// Socket file should still exist after closing the fd.
		_, err := os.Stat(path)
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
	dir := tempSocketDir(t)
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
	dir := tempSocketDir(t)
	path := filepath.Join(dir, "stale.sock")
	createStaleSocket(t, path)

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
	dir := tempSocketDir(t)
	path := filepath.Join(dir, "regular.txt")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))

	na := &AddrConfig{
		Endpoint:  path,
		Transport: TransportTypeUnix,
	}
	_, err := na.Listen(context.Background())
	assert.ErrorContains(t, err, "not a socket")
}

func TestAddrConfig_Listen_UnixSocketPermissions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name              string
		socketPermissions os.FileMode
		want              os.FileMode
	}{
		{name: "default", want: defaultSocketPermissions},
		{name: "custom", socketPermissions: 0o700, want: 0o700},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tempSocketDir(t)
			path := filepath.Join(dir, "perms.sock")

			na := &AddrConfig{
				Endpoint:          path,
				Transport:         TransportTypeUnix,
				SocketPermissions: tt.socketPermissions,
			}
			ln, err := na.Listen(context.Background())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, ln.Close()) })

			fi, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, tt.want|os.ModeSocket, fi.Mode())
		})
	}
}

func TestAddrConfig_Listen_UnixSocketManagementDisabled_StaleSocketNotRemoved(t *testing.T) {
	t.Parallel()
	dir := tempSocketDir(t)
	path := filepath.Join(dir, "stale.sock")
	createStaleSocket(t, path)

	na := &AddrConfig{
		Endpoint:                 path,
		Transport:                TransportTypeUnix,
		SocketManagementDisabled: true,
	}
	_, err := na.Listen(context.Background())
	assert.Error(t, err)
}

func TestAddrConfig_Listen_UnixInvalidEndpoint(t *testing.T) {
	t.Parallel()
	na := &AddrConfig{
		Endpoint:  "/nonexistent/dir/deep/socket.sock",
		Transport: TransportTypeUnix,
	}
	_, err := na.Listen(context.Background())
	assert.Error(t, err)
}

func TestAddrConfig_Listen_UnixChmodFailure(t *testing.T) {
	t.Parallel()
	dir := tempSocketDir(t)

	// Make the directory read-only so Listen succeeds at binding but Chmod
	// fails due to permission denied.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o700))
	subPath := filepath.Join(dir, "sub", "chmod.sock")
	na := &AddrConfig{
		Endpoint:  subPath,
		Transport: TransportTypeUnix,
	}
	ln, err := na.Listen(context.Background())
	require.NoError(t, err)
	require.NoError(t, ln.Close())

	// Make dir read-only to cause Chmod failure
	require.NoError(t, os.Chmod(filepath.Join(dir, "sub"), 0o444))       //nolint:gosec // intentional for test
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, "sub"), 0o700) }) //nolint:gosec // restore perms for cleanup

	_, err = na.Listen(context.Background())
	// On some systems this fails at Listen (can't bind), on others at Chmod.
	// Either way it should error.
	assert.Error(t, err)
}

func Test_removeStaleSocket_StatError(t *testing.T) {
	t.Parallel()
	dir := tempSocketDir(t)
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // restore perms for cleanup
	path := filepath.Join(dir, "socket.sock")

	// Create a file so the path exists.
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
	// Remove permission on the directory so Stat fails with permission denied.
	require.NoError(t, os.Chmod(dir, 0o000))

	err := removeStaleSocket(path)
	assert.Error(t, err)
}

func TestAddrConfig_Listen_UnixSocketCloseRemovesFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		socketManagementDisabled bool
		wantRemoved              bool
	}{
		{name: "managed: file removed on close", wantRemoved: true},
		{name: "unmanaged: file kept on close", socketManagementDisabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tempSocketDir(t)
			path := filepath.Join(dir, "cleanup.sock")

			na := &AddrConfig{
				Endpoint:                 path,
				Transport:                TransportTypeUnix,
				SocketManagementDisabled: tt.socketManagementDisabled,
			}
			ln, err := na.Listen(context.Background())
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.Remove(path) })

			_, err = os.Stat(path)
			require.NoError(t, err)

			require.NoError(t, ln.Close())

			_, err = os.Stat(path)
			if tt.wantRemoved {
				assert.True(t, os.IsNotExist(err))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
