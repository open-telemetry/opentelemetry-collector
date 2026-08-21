// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

var exporterID = component.NewID(exportertest.NopType)

type fakeQueue[T any] struct {
	Queue[T]
	offerErr error
	size     int64
	capacity int64
}

func (fq *fakeQueue[T]) Size() int64 {
	return fq.size
}

func (fq *fakeQueue[T]) Capacity() int64 {
	return fq.capacity
}

func (fq *fakeQueue[T]) Offer(context.Context, T) error {
	return fq.offerErr
}

func newFakeQueue[T request.Request](offerErr error, size, capacity int64) Queue[T] {
	return &fakeQueue[T]{offerErr: offerErr, size: size, capacity: capacity}
}

type enqueueSize struct {
	items int64
	bytes int64
}

// recordingMetrics records the operations reported by the queue.
type recordingMetrics struct {
	enqueueFailures []int64
	enqueueSizes    []enqueueSize
	sizeObserver    func() int64
	capacityObserve func() int64
	sizeErr         error
	capacityErr     error
}

// metrics returns the QueueMetrics that feeds the recorder.
func (rm *recordingMetrics) metrics() QueueMetrics {
	return NewQueueMetrics(
		func(_ context.Context, items int64) {
			rm.enqueueFailures = append(rm.enqueueFailures, items)
		},
		func(_ context.Context, items int64, bytesSize func() int64) {
			rm.enqueueSizes = append(rm.enqueueSizes, enqueueSize{items: items, bytes: bytesSize()})
		},
		func(observe func() int64) error {
			rm.sizeObserver = observe
			return rm.sizeErr
		},
		func(observe func() int64) error {
			rm.capacityObserve = observe
			return rm.capacityErr
		},
	)
}

func newTestSettings() Settings[request.Request] {
	return Settings[request.Request]{
		Signal:    pipeline.SignalLogs,
		ID:        exporterID,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
}

func TestObsQueueRegistersSizeAndCapacityObservers(t *testing.T) {
	om := &recordingMetrics{}
	set := newTestSettings()
	set.QueueMetrics = om.metrics()

	_, err := newObsQueue[request.Request](set, newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)

	require.NotNil(t, om.sizeObserver)
	require.NotNil(t, om.capacityObserve)
	assert.Equal(t, int64(7), om.sizeObserver())
	assert.Equal(t, int64(9), om.capacityObserve())
}

func TestObsQueueRecordsEnqueueSize(t *testing.T) {
	om := &recordingMetrics{}
	set := newTestSettings()
	set.QueueMetrics = om.metrics()

	te, err := newObsQueue[request.Request](set, newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 100}))

	assert.Equal(t, []enqueueSize{{items: 2, bytes: 100}}, om.enqueueSizes)
	assert.Empty(t, om.enqueueFailures)
}

func TestObsQueueRecordsEnqueueFailure(t *testing.T) {
	om := &recordingMetrics{}
	set := newTestSettings()
	set.QueueMetrics = om.metrics()

	te, err := newObsQueue[request.Request](set, newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 12, Bytes: 200}))

	assert.Equal(t, []enqueueSize{{items: 12, bytes: 200}}, om.enqueueSizes)
	assert.Equal(t, []int64{12}, om.enqueueFailures)
}

// A queue created without QueueMetrics reports nothing.
func TestObsQueueWithoutMetricsIsNoOp(t *testing.T) {
	te, err := newObsQueue[request.Request](newTestSettings(), newFakeQueue[request.Request](nil, 7, 9))
	require.NoError(t, err)
	require.NoError(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 100}))

	te, err = newObsQueue[request.Request](newTestSettings(), newFakeQueue[request.Request](errors.New("my error"), 0, 0))
	require.NoError(t, err)
	require.Error(t, te.Offer(context.Background(), &requesttest.FakeRequest{Items: 2, Bytes: 100}))
}

func TestObsQueueRegistrationFailure(t *testing.T) {
	errRegister := errors.New("register failed")

	for _, tt := range []struct {
		name string
		om   *recordingMetrics
	}{
		{name: "size", om: &recordingMetrics{sizeErr: errRegister}},
		{name: "capacity", om: &recordingMetrics{capacityErr: errRegister}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			set := newTestSettings()
			set.QueueMetrics = tt.om.metrics()
			_, err := newObsQueue[request.Request](set, newFakeQueue[request.Request](nil, 7, 9))
			require.ErrorIs(t, err, errRegister)
		})
	}
}
