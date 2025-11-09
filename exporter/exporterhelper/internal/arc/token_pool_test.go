// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package arc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenPool_New(t *testing.T) {
	p := newTokenPool(10)
	assert.Equal(t, 10, p.cap)
	assert.Equal(t, 0, p.inUse)
	assert.NotNil(t, p.cond)
	assert.False(t, p.dead)
}

func TestTokenPool_AcquireRelease(t *testing.T) {
	p := newTokenPool(2)

	// Acquire first permit
	err := p.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, p.inUse)

	// Acquire second permit
	err = p.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, p.inUse)

	// Release first permit
	p.Release()
	assert.Equal(t, 1, p.inUse)

	// Acquire again
	err = p.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, p.inUse)

	// Release both
	p.Release()
	p.Release()
	assert.Equal(t, 0, p.inUse)
}

func TestTokenPool_Release_NoOverRelease(t *testing.T) {
	p := newTokenPool(1)
	p.Release()
	assert.Equal(t, 0, p.inUse) // Should not go negative
}

func TestTokenPool_Blocking(t *testing.T) {
	p := newTokenPool(1)

	// Acquire the only permit
	err := p.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, p.inUse)

	// This next acquire should block
	blockerDone := make(chan struct{})
	go func() {
		err := p.Acquire(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 1, p.inUse)
		p.Release()
		close(blockerDone)
	}()

	// Give goroutine time to block
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, p.inUse)

	// Release the permit
	p.Release()

	// The goroutine should now unblock
	select {
	case <-blockerDone:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine did not unblock after release")
	}

	assert.Equal(t, 0, p.inUse)
}

func TestTokenPool_Close(t *testing.T) {
	p := newTokenPool(1)

	// Acquire the only permit
	err := p.Acquire(context.Background())
	require.NoError(t, err)

	// This next acquire should block
	blockerDone := make(chan error, 1)
	go func() {
		blockerDone <- p.Acquire(context.Background())
	}()

	// Give goroutine time to block
	time.Sleep(20 * time.Millisecond)

	// Close the pool
	p.Close()
	assert.True(t, p.dead)

	// The goroutine should unblock with an error
	select {
	case closeErr := <-blockerDone: // Renamed 'err' to 'closeErr' to avoid shadow
		require.Error(t, closeErr)
		assert.Contains(t, closeErr.Error(), "token pool closed")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine did not unblock after Close")
	}

	// New acquires should also fail
	err = p.Acquire(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token pool closed")
}

func TestTokenPool_Grow(t *testing.T) {
	p := newTokenPool(1)

	// Acquire the only permit
	err := p.Acquire(context.Background())
	require.NoError(t, err)

	// This next acquire should block
	blockerDone := make(chan struct{})
	go func() {
		err := p.Acquire(context.Background())
		assert.NoError(t, err)
		close(blockerDone)
	}()

	// Give goroutine time to block
	time.Sleep(20 * time.Millisecond)

	// Grow the pool
	p.Grow(1)
	assert.Equal(t, 2, p.cap)

	// The goroutine should unblock
	select {
	case <-blockerDone:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine did not unblock after Grow")
	}

	assert.Equal(t, 2, p.inUse)
}

func TestTokenPool_Acquire_ContextCanceled(t *testing.T) {
	p := newTokenPool(1)

	// Acquire the only permit
	err := p.Acquire(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, p.inUse)

	// This next acquire should block
	ctx, cancel := context.WithCancel(context.Background())
	blockerDone := make(chan error, 1)
	go func() {
		blockerDone <- p.Acquire(ctx)
	}()

	// Give goroutine time to block
	time.Sleep(20 * time.Millisecond)

	// Cancel the context
	cancel()

	// The goroutine should unblock with an error
	select {
	case acqErr := <-blockerDone: // This line fixes the shadow error
		require.Error(t, acqErr)
		require.ErrorIs(t, acqErr, context.Canceled)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("goroutine did not unblock after context cancel")
	}

	// The permit was not acquired
	assert.Equal(t, 1, p.inUse)
}
