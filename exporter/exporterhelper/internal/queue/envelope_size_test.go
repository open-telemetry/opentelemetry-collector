// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

type countedEnvelope struct{ intRequest }

func (e countedEnvelope) QueueRequestsCount() int64 { return int64(e.intRequest) }

func TestAtomicEnvelopeRequestChargesBoundMemoryQueue(t *testing.T) {
	ctx := context.Background()
	q := newMemoryQueue[countedEnvelope](Settings[countedEnvelope]{SizerType: request.SizerTypeRequests, Capacity: 4})
	require.NoError(t, q.Offer(ctx, countedEnvelope{4}))
	require.Equal(t, int64(4), q.Size())
	require.ErrorIs(t, q.Offer(ctx, countedEnvelope{1}), ErrQueueIsFull)
	_, _, done, ok := q.Read(ctx)
	require.True(t, ok)
	done.OnDone(nil)
	require.Zero(t, q.Size())
}
