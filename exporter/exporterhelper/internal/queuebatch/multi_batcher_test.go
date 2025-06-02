// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
)

func TestMultiBatcher_NoTimeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 0,
		MinSize:      10,
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba := newMultiBatcher(cfg, batcherSettings[request.Request]{
		sizerType: request.SizerTypeItems,
		sizer:     request.NewItemsSizer(),
		partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		next:       sink.Export,
		maxWorkers: 1,
	})

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	// Neither batch should be flushed since they haven't reached min threshold.
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 28
	}, 1*time.Second, 10*time.Millisecond)
	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 4, done.success.Load())

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
}

func TestMultiBatcher_Timeout(t *testing.T) {
	cfg := BatchConfig{
		FlushTimeout: 100 * time.Millisecond,
		MinSize:      100,
	}
	sink := requesttest.NewSink()

	type partitionKey struct{}

	ba := newMultiBatcher(cfg, batcherSettings[request.Request]{
		sizerType: request.SizerTypeItems,
		sizer:     request.NewItemsSizer(),
		partitioner: NewPartitioner(func(ctx context.Context, _ request.Request) string {
			return ctx.Value(partitionKey{}).(string)
		}),
		next:       sink.Export,
		maxWorkers: 1,
	})

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		require.NoError(t, ba.Shutdown(context.Background()))
	})

	done := newFakeDone()
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	// Neither batch should be flushed since they haven't reached min threshold.
	assert.Equal(t, 0, sink.RequestsCount())
	assert.Equal(t, 0, sink.ItemsCount())

	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p1"), &requesttest.FakeRequest{Items: 8}, done)
	ba.Consume(context.WithValue(context.Background(), partitionKey{}, "p2"), &requesttest.FakeRequest{Items: 6}, done)

	assert.Eventually(t, func() bool {
		return sink.RequestsCount() == 2 && sink.ItemsCount() == 28
	}, 1*time.Second, 10*time.Millisecond)
	// Check that done callback is called for the right amount of times.
	assert.EqualValues(t, 0, done.errors.Load())
	assert.EqualValues(t, 4, done.success.Load())

	require.NoError(t, ba.Start(context.Background(), componenttest.NewNopHost()))
}
