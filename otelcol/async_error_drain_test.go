// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrainAsyncErrorsForwardsFirstError(t *testing.T) {
	src := make(chan error)
	dst := make(chan error, 1)
	stop := make(chan struct{})
	defer close(stop)

	go drainAsyncErrors(src, dst, stop)

	firstErr := errors.New("first")
	src <- firstErr

	select {
	case err := <-dst:
		require.ErrorIs(t, err, firstErr)
	case <-time.After(2 * time.Second):
		t.Fatal("expected first error to be forwarded")
	}
}

func TestDrainAsyncErrorsCoalescesExtraErrors(t *testing.T) {
	src := make(chan error)
	dst := make(chan error, 1)
	stop := make(chan struct{})
	defer close(stop)

	go drainAsyncErrors(src, dst, stop)

	firstErr := errors.New("first")
	src <- firstErr
	// Let the drain pick up the first error before filling the channel.
	require.Eventually(t, func() bool {
		select {
		case err := <-dst:
			require.ErrorIs(t, err, firstErr)
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond)

	// The destination is now empty; send a second error and make sure the
	// drain does not block on it even with no reader consuming dst.
	src <- errors.New("second")
	src <- errors.New("third")
	select {
	case err := <-dst:
		assert.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("drain blocked on coalescing extra errors")
	}
}

func TestDrainAsyncErrorsExitsOnStop(t *testing.T) {
	src := make(chan error)
	dst := make(chan error, 1)
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		drainAsyncErrors(src, dst, stop)
		close(done)
	}()

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drain goroutine did not exit after stop was closed")
	}
}
