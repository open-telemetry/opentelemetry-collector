// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/experr"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

type countedEnvelope struct{ intRequest }

func (e countedEnvelope) QueueRequestsCount() int64 { return int64(e.intRequest) }

type countedEnvelopeEncoding struct{}

func (countedEnvelopeEncoding) Marshal(_ context.Context, e countedEnvelope) ([]byte, error) {
	return []byte(strconv.FormatInt(int64(e.intRequest), 10)), nil
}

func (countedEnvelopeEncoding) Unmarshal(data []byte) (context.Context, countedEnvelope, error) {
	value, err := strconv.ParseInt(string(data), 10, 64)
	return context.Background(), countedEnvelope{intRequest(value)}, err
}

func TestAtomicEnvelopeRequestChargesSurvivePersistentRestart(t *testing.T) {
	ctx := context.Background()
	id := component.MustNewIDWithName("file_storage", "counted")
	host := hosttest.NewHost(map[component.ID]component.Component{id: storagetest.NewMockStorageExtension(nil)})
	set := Settings[countedEnvelope]{SizerType: request.SizerTypeRequests, Capacity: 4, StorageID: &id, Encoding: countedEnvelopeEncoding{}, Telemetry: exportertest.NewNopSettings(exportertest.NopType).TelemetrySettings}
	q := newPersistentQueue[countedEnvelope](set)
	require.NoError(t, q.Start(ctx, host))
	require.NoError(t, q.Offer(ctx, countedEnvelope{4}))
	require.Equal(t, int64(4), q.Size())
	require.ErrorIs(t, q.Offer(ctx, countedEnvelope{1}), ErrQueueIsFull)
	_, _, done, ok := q.Read(ctx)
	require.True(t, ok)
	done.OnDone(experr.NewShutdownErr(context.Canceled))
	require.NoError(t, q.Shutdown(ctx))
	restored := newPersistentQueue[countedEnvelope](set)
	require.NoError(t, restored.Start(ctx, host))
	require.Equal(t, int64(4), restored.Size())
	require.ErrorIs(t, restored.Offer(ctx, countedEnvelope{1}), ErrQueueIsFull)
	_, _, done, ok = restored.Read(ctx)
	require.True(t, ok)
	done.OnDone(nil)
	require.Zero(t, restored.Size())
	require.NoError(t, restored.Shutdown(ctx))
}

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
