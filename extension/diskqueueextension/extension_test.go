// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package diskqueueextension

import (
	"context"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/queue"
)

func newTestExtension(t *testing.T, dir string) *diskAccessExtension {
	t.Helper()
	return &diskAccessExtension{
		cfg: &Config{
			DataPath:              dir,
			MaxBytesPerFile:       10_000_000,
			SyncEvery:             1,
			SyncTimeout:           time.Second,
			MetadataTruncateEvery: 1000,
		},
		logger: zap.NewNop(),
	}
}

func newTestClient(t *testing.T, dir string) *diskAccessClient {
	t.Helper()
	c, err := newTestExtension(t, dir).GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	return c.(*diskAccessClient)
}

func TestExtensionStartShutdown(t *testing.T) {
	ext := newTestExtension(t, t.TempDir())
	require.NoError(t, ext.Start(context.Background(), nil))
	require.NoError(t, ext.Shutdown(context.Background()))
}

func TestExtensionShutdownDirectly(t *testing.T) {
	ext := newTestExtension(t, t.TempDir())
	require.NoError(t, ext.Shutdown(context.Background()))
}

func TestEmptyQueue(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	select {
	case <-c.Peek():
		assert.Fail(t, "should not peek")
	default:
	}
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestPutPeekConsume(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	assert.Equal(t, "hello world", string(msg.Payload))
	msg.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestCatchUpToHeadAndReadOne(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	// we caught up to tip, now do one more
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg = <-c.Peek()
	msg.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestWaitForOneMore(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = c.Write(queue.WriteOp{Payload: []byte("hello world2"), Size: 1})
	}()
	msg = <-c.Peek()
	assert.Equal(t, "hello world2", string(msg.Payload))

	msg.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestThreePutsThreeConsumes(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world2"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world3"), Size: 1}))
	msg := <-c.Peek()
	assert.Equal(t, "hello world", string(msg.Payload))
	msg = <-c.Peek()
	assert.Equal(t, "hello world2", string(msg.Payload))
	msg = <-c.Peek()
	assert.Equal(t, "hello world3", string(msg.Payload))
	msg.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestThreePutsThreeConsumesOutOfOrder(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world2"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world3"), Size: 1}))
	msg1 := <-c.Peek()
	msg1.ConsumeCallback(nil)
	msg2 := <-c.Peek()
	msg3 := <-c.Peek()
	msg3.ConsumeCallback(nil)
	msg2.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestStartStopRestart(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world2"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world3"), Size: 1}))
	msg1 := <-c.Peek()
	msg1.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))

	c2 := newTestClient(t, dir)
	msg2 := <-c2.Peek()
	assert.Equal(t, "hello world2", string(msg2.Payload))
	require.NoError(t, c2.Shutdown(context.Background()))
}

func TestSize(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	assert.Equal(t, int64(0), c.Size())
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 5}))
	assert.Equal(t, int64(5), c.Size())
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	assert.Eventually(t, func() bool {
		return c.Size() == 0
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestFileRotation(t *testing.T) {
	ext := newTestExtension(t, t.TempDir())
	ext.cfg.MaxBytesPerFile = 60
	c, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	for range 5 {
		require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	}
	for range 5 {
		msg := <-c.Peek()
		msg.ConsumeCallback(nil)
	}
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestPeekReturnsNilWhenExiting(t *testing.T) {
	d := &diskAccessClient{}
	d.exitFlag.Store(true)
	assert.Nil(t, d.Peek())
}

// TestGetClientCorruptedMetadataSeekError forces a Seek error (not IsNotExist)
// while retrieving the write metadata: the existing file is too short to seek
// backward the expected number of bytes.
func TestGetClientCorruptedMetadataSeekError(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.diskaccess.meta.dat"), []byte("x"), 0o600))
	_, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.Error(t, err)
	assert.False(t, os.IsNotExist(err))
}

func TestGetClientCorruptedPeekMetadataSeekError(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.diskaccess.peek.dat"), []byte("x"), 0o600))
	_, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.Error(t, err)
	assert.False(t, os.IsNotExist(err))
}

// TestGetClientMetadataReadFromError forces the ReadFrom call to fail by
// pointing the metadata file path at a directory instead of a regular file.
func TestGetClientMetadataReadFromError(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "test.diskaccess.meta.dat"), 0o600))
	_, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.Error(t, err)
}

func TestGetClientPeekMetadataReadFromError(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "test.diskaccess.peek.dat"), 0o600))
	_, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.Error(t, err)
}

func TestWriteOpenFileError(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	require.NoError(t, os.Chmod(dir, 0o500))       // #nosec G302
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // #nosec G302
	err := c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1})
	require.Error(t, err)
}

func TestPersistMetaDataOpenFileError(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	require.NoError(t, os.Chmod(dir, 0o500))       // #nosec G302
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // #nosec G302
	err := c.persistMetaData()
	require.Error(t, err)
}

func TestPersistMetaDataWriteError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close())
	require.NoError(t, w.Close())
	c.metadataFile = w
	err = c.persistMetaData()
	require.Error(t, err)
	assert.Nil(t, c.metadataFile)
}

func TestPersistMetaDataSyncError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	c.metadataFile = w
	err = c.persistMetaData()
	require.Error(t, err)
	assert.Nil(t, c.metadataFile)
}

func TestSyncPeekOpenFileError(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	require.NoError(t, os.Chmod(dir, 0o500))       // #nosec G302
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) // #nosec G302
	err := c.syncPeek()
	require.Error(t, err)
}

func TestSyncWriteFileSyncError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)

	// force the writeFile.Sync() call inside sync() to fail by closing the
	// underlying file out from under the client.
	require.NoError(t, c.writeFile.Close())
	err := c.sync()
	require.Error(t, err)
	assert.Nil(t, c.writeFile)
}

func TestPeekDataFileOpenError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	_, err := c.peekData()
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestPeekDataReadLenEOF(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	require.NoError(t, os.WriteFile(c.fileName(0), []byte{}, 0o600))
	op, err := c.peekData()
	require.NoError(t, err)
	assert.Empty(t, op.Payload)
}

func TestPeekDataReadSizeEOF(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	buf := binary.BigEndian.AppendUint64(nil, 5)
	require.NoError(t, os.WriteFile(c.fileName(0), buf, 0o600))
	op, err := c.peekData()
	require.NoError(t, err)
	assert.Empty(t, op.Payload)
}

func TestPeekDataReadBufError(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	buf := binary.BigEndian.AppendUint64(nil, 100) // datalen
	buf = binary.BigEndian.AppendUint64(buf, 1)    // size
	require.NoError(t, os.WriteFile(c.fileName(0), buf, 0o600))
	_, err := c.peekData()
	require.Error(t, err)
}

// TestPeekDataReadSizeErrNonEOF exercises the branch where the readLen
// ReadAt succeeds but the subsequent readSize ReadAt fails with a non-EOF
// error (extension.go:302-304), as opposed to the readLen ReadAt itself
// failing (extension.go:291-293). There's no way to force this
// deterministically through the public API: which of the two fails depends
// on exactly when a concurrent Close() lands relative to peekData()'s two
// ReadAt calls. This races many independent Close()s (each against a fresh
// file/fd) against tight peekData() loops; across enough independent
// attempts, landing on the readSize side at least once is overwhelmingly
// likely.
func TestPeekDataReadSizeErrNonEOF(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })

	for range 300 {
		fileName := c.fileName(0)
		require.NoError(t, os.WriteFile(fileName, make([]byte, 32), 0o600))
		f, err := os.OpenFile(fileName, os.O_RDONLY, 0o600) // #nosec G304
		require.NoError(t, err)
		c.peekFile = f

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = f.Close()
		}()

		var lastErr error
		for i := 0; i < 20_000 && lastErr == nil; i++ {
			if c.peekFile == nil {
				c.peekFile = f
			}
			_, lastErr = c.peekData()
		}
		<-done
		require.Error(t, lastErr)
		require.NotErrorIs(t, lastErr, io.EOF)
		assert.Nil(t, c.peekFile)
	}
}

func TestPeekDataReadLenErrNonEOF(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	// open write-only so ReadAt on the peekFile fails with a non-EOF error.
	f, err := os.OpenFile(c.fileName(0), os.O_WRONLY|os.O_CREATE, 0o600)
	require.NoError(t, err)
	c.peekFile = f
	_, err = c.peekData()
	require.Error(t, err)
	assert.Nil(t, c.peekFile)
}

func TestSyncPeekWriteError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close())
	require.NoError(t, w.Close())
	c.peekMetadataFile = w
	err = c.syncPeek()
	require.Error(t, err)
	assert.Nil(t, c.peekMetadataFile)
}

func TestSyncPeekSyncError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	c.peekMetadataFile = w
	err = c.syncPeek()
	require.Error(t, err)
	assert.Nil(t, c.peekMetadataFile)
}

func TestWriteOpenFileErrorOnRotatedFile(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	c.metadata.pos = 100
	require.NoError(t, os.WriteFile(c.fileName(0), []byte("short"), 0o600))
	require.NoError(t, os.Chmod(c.fileName(0), 0o000))
	t.Cleanup(func() { _ = os.Chmod(c.fileName(0), 0o600) })
	err := c.write(queue.WriteOp{Payload: []byte("hi"), Size: 1})
	require.Error(t, err)
}

func TestWriteWriteError(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	r, w, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, r.Close())
	c.writeFile = w
	err = c.write(queue.WriteOp{Payload: []byte("hi"), Size: 1})
	require.Error(t, err)
	assert.Nil(t, c.writeFile)
}

func TestWriteRotationSyncFailureIsLoggedAndReturned(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.MaxBytesPerFile = 5
	c0, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	c := c0.(*diskAccessClient)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	// point the data file at /dev/null: the payload write itself succeeds,
	// but the Sync() call inside the sync() triggered by rotating into a
	// new file fails deterministically; write() logs that error and also
	// returns it as its own result, even though the payload was persisted
	// and the rotation to the next file number still took effect.
	require.NoError(t, os.Symlink("/dev/null", c.fileName(0)))
	err = c.write(queue.WriteOp{Payload: []byte("hello world"), Size: 1})
	require.Error(t, err)
	assert.Equal(t, int64(1), c.metadata.fileNum)
	assert.Nil(t, c.writeFile)
}

func TestWriteLoopSyncErrorLogged(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.SyncEvery = 1
	ext.cfg.SyncTimeout = 10 * time.Millisecond
	core, observed := observer.New(zap.ErrorLevel)
	ext.logger = zap.New(core)
	c0, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	c := c0.(*diskAccessClient)
	// point the data file at /dev/null: writes succeed but Sync() fails,
	// deterministically forcing the periodic sync() triggered by the
	// ticker to fail; writeLoop only logs the error and keeps running.
	require.NoError(t, os.Symlink("/dev/null", c.fileName(0)))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	require.Eventually(t, func() bool {
		return observed.FilterMessage("failed to sync").Len() > 0
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestReadLoopSyncPeekErrorLogged(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.SyncEvery = 1
	ext.cfg.SyncTimeout = 10 * time.Millisecond
	core, observed := observer.New(zap.ErrorLevel)
	ext.logger = zap.New(core)
	c0, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	c := c0.(*diskAccessClient)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	// force the periodic syncPeek() triggered by the ticker to fail.
	require.NoError(t, os.Chmod(dir, 0o500)) // #nosec G302
	require.Eventually(t, func() bool {
		return observed.FilterMessage("failed to sync").Len() > 0
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, os.Chmod(dir, 0o700)) // #nosec G302
	msg.ConsumeCallback(nil)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestCallbackRemovesOldFile(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.MaxBytesPerFile = 10
	c0, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	c := c0.(*diskAccessClient)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("world"), Size: 1}))
	oldFile := c.fileName(0)
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	assert.Eventually(t, func() bool {
		_, statErr := os.Stat(oldFile)
		return os.IsNotExist(statErr)
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestCallbackRemoveOldFileErrorLogged(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.MaxBytesPerFile = 10
	core, observed := observer.New(zap.ErrorLevel)
	ext.logger = zap.New(core)
	c0, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)
	c := c0.(*diskAccessClient)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello"), Size: 1}))
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("world"), Size: 1}))
	// make the directory read-only so the os.Remove of the now-unreferenced
	// rotated-out file fails with permission-denied rather than
	// IsNotExist; readLoop only logs the error and keeps running.
	require.NoError(t, os.Chmod(dir, 0o500)) // #nosec G302
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	require.Eventually(t, func() bool {
		return observed.FilterMessage(" failed to Remove").Len() > 0
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, os.Chmod(dir, 0o700)) // #nosec G302
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestReadOneExitDuringSend(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	// stop the background loops so we can drive readOne directly without
	// anyone competing for the peekChan/exitChan.
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	// drain the auto-delivered message so the loops go idle waiting on peekRequestChan.
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world2"), Size: 1}))

	close(c.exitChan)
	c.exitFlag.Store(true)
	c.exitWG.Wait()

	done := make(chan bool, 1)
	go func() {
		done <- c.readOne(map[int64]int{})
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readOne did not return after exitChan was closed")
	}

	close(c.peekChan)
	_ = c.sync()
	_ = c.syncPeek()
	if c.writeFile != nil {
		_ = c.writeFile.Close()
	}
	if c.peekFile != nil {
		_ = c.peekFile.Close()
	}
	if c.metadataFile != nil {
		_ = c.metadataFile.Close()
	}
	if c.peekMetadataFile != nil {
		_ = c.peekMetadataFile.Close()
	}
}

func TestWriteLoopAndReadLoopTicker(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtension(t, dir)
	ext.cfg.SyncTimeout = 10 * time.Millisecond
	ext.cfg.SyncEvery = 1
	ext.cfg.MetadataTruncateEvery = 1
	c, err := ext.GetClient(context.Background(), component.KindExporter, component.MustNewID("foo"), "test")
	require.NoError(t, err)

	// let the ticker fire at least once with no activity (opCount/peekOps == 0).
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 1}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)

	// let the ticker fire again, this time with activity to sync/truncate.
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, c.Shutdown(context.Background()))
}

func TestRetrieveMetaDataSuccess(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 5}))
	require.NoError(t, c.sync())

	m, err := c.retrieveMetaData(c.metaDataFilePath())
	require.NoError(t, err)
	assert.Equal(t, int64(0), m.fileNum)
	assert.Equal(t, int64(5), m.size.Load())
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestRetrievePeekMetaDataSuccess(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hello world"), Size: 5}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)
	require.NoError(t, c.syncPeek())

	m, err := c.retrievePeekMetaData(c.peekMetaDataFilePath())
	require.NoError(t, err)
	assert.Equal(t, int64(0), m.fileNum)
	require.NoError(t, c.Shutdown(context.Background()))
}

func TestClientSizeAtomic(t *testing.T) {
	d := &diskAccessClient{metadata: metadata{size: &atomic.Int64{}}}
	d.metadata.size.Store(42)
	assert.Equal(t, int64(42), d.Size())
}

func TestShutdownWithNothingOpened(t *testing.T) {
	c := newTestClient(t, t.TempDir())
	require.NoError(t, c.Shutdown(context.Background()))
}
