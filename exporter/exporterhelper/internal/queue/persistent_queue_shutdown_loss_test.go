// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
)

// A request that fails while the queue is shutting down must stay on disk, so the
// backlog is redelivered after restart. Before #15677 only errors from the retry
// sender's backoff wait were kept; an exporter without retry_on_failure produced an
// ordinary error and the shutdown drain deleted the whole backlog. A hard kill kept
// the data that a clean shutdown destroyed.
func TestPersistentQueue_FailedSendDuringShutdownIsKept(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsSizer(t, ext, 1000)

	require.NoError(t, ps.Offer(context.Background(), intRequest(50)))
	require.Equal(t, int64(1), ps.Size())

	_, _, done, ok := ps.Read(context.Background())
	require.True(t, ok)

	require.NoError(t, ps.Shutdown(context.Background()))

	// An ordinary error, not experr.NewShutdownErr: this is what an exporter
	// without retry_on_failure reports when the destination is down.
	done.OnDone(errors.New("connection refused"))

	// Reopen against the same storage; the item must still be there.
	restored := createTestPersistentQueueWithRequestsSizer(t, ext, 1000)
	assert.Equal(t, int64(1), restored.Size(),
		"the queued item was deleted by shutdown, so the backlog is lost on restart")
}

// The other half of the contract: a request that succeeded during shutdown must
// still be removed, or restart would deliver it a second time.
func TestPersistentQueue_SuccessfulSendDuringShutdownIsRemoved(t *testing.T) {
	ext := storagetest.NewMockStorageExtension(nil)
	ps := createTestPersistentQueueWithRequestsSizer(t, ext, 1000)

	require.NoError(t, ps.Offer(context.Background(), intRequest(50)))

	_, _, done, ok := ps.Read(context.Background())
	require.True(t, ok)

	require.NoError(t, ps.Shutdown(context.Background()))
	done.OnDone(nil)

	restored := createTestPersistentQueueWithRequestsSizer(t, ext, 1000)
	assert.Equal(t, int64(0), restored.Size(),
		"a request that was delivered must not be redelivered after restart")
}
