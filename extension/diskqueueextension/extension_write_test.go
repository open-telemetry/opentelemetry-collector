// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package diskqueueextension

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/extension/xextension/queue"
)

// Those tests use low level OS calls to simulate write failures, and only work on Linux.

func TestWriteSeekError(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	t.Cleanup(func() { _ = c.Shutdown(context.Background()) })
	// FIFOs cannot be seeked. Opening a FIFO O_RDWR does not block (unlike
	// opening it read-only or write-only alone), so write()'s OpenFile call
	// succeeds and its Seek() call fails deterministically.
	require.NoError(t, syscall.Mkfifo(c.fileName(0), 0o600))
	c.metadata.pos = 5
	err := c.write(queue.WriteOp{Payload: []byte("hi"), Size: 1})
	require.Error(t, err)
	assert.Nil(t, c.writeFile)
}

func TestReadLoopResumesWaitingAfterFailedWrite(t *testing.T) {
	dir := t.TempDir()
	c := newTestClient(t, dir)
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hi"), Size: 1}))
	msg := <-c.Peek()
	msg.ConsumeCallback(nil)

	peekResult := make(chan struct{})
	go func() {
		<-c.Peek() // caught up to head: readOne returns false, readLoop parks on waitForWriteChan.
		close(peekResult)
	}()
	time.Sleep(50 * time.Millisecond)

	// Close the writeFile's fd out from under the client so the next write
	// fails, while writeLoop still unconditionally pings waitForWriteChan
	// afterwards; readLoop wakes up, finds nothing new, and parks again rather than delivering to peekChan.
	require.NotNil(t, c.writeFile)
	require.NoError(t, syscall.Close(int(c.writeFile.Fd())))
	require.Error(t, c.Write(queue.WriteOp{Payload: []byte("hi2"), Size: 1}))

	time.Sleep(50 * time.Millisecond)
	select {
	case <-peekResult:
		t.Fatal("peek should still be blocked, nothing was successfully written")
	default:
	}

	c.writeFile = nil
	require.NoError(t, c.Write(queue.WriteOp{Payload: []byte("hi3"), Size: 1}))
	<-peekResult
	require.NoError(t, c.Shutdown(context.Background()))
}
