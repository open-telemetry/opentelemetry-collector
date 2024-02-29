// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal/queue"
)

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	be, err := newBaseExporter(defaultSettings, defaultType, newObservabilityConsumerSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	firstMockR := newErrorRequest()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(context.Background(), firstMockR))
	})

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	secondMockR := newMockRequest(3, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(context.Background(), secondMockR))
	})

	require.LessOrEqual(t, 1, be.queueSender.(*queueSender).queue.Size())

	assert.NoError(t, be.Shutdown(context.Background()))

	secondMockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 3)
	ocs.checkDroppedItemsCount(t, 7)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_DoNotPreserveCancellation(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	be, err := newBaseExporter(defaultSettings, defaultType, newObservabilityConsumerSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	mockR := newMockRequest(2, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(ctx, mockR))
	})
	ocs.awaitAsyncProcessing()

	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_RejectOnFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.QueueSize = 0
	qCfg.NumConsumers = 0
	set := exportertest.NewNopCreateSettings()
	logger, observed := observer.New(zap.ErrorLevel)
	set.Logger = zap.New(logger)
	be, err := newBaseExporter(set, defaultType, newNoopObsrepSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	require.Error(t, be.send(context.Background(), newMockRequest(2, nil)))
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Rejecting data.", observed.All()[0].Message)
	assert.Equal(t, "sending queue is full", observed.All()[0].ContextMap()["error"])
}

func TestQueuedRetryHappyPath(t *testing.T) {
	tests := []struct {
		name         string
		queueOptions []Option
	}{
		{
			name: "WithQueue",
			queueOptions: []Option{
				withMarshaler(mockRequestMarshaler),
				withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
				WithQueue(QueueSettings{
					Enabled:      true,
					QueueSize:    10,
					NumConsumers: 1,
				}),
				WithRetry(configretry.NewDefaultBackOffConfig()),
			},
		},
		{
			name: "WithRequestQueue/MemoryQueueFactory",
			queueOptions: []Option{
				WithRequestQueue(exporterqueue.Config{
					Enabled:      true,
					QueueSize:    10,
					NumConsumers: 1,
				}, exporterqueue.NewMemoryQueueFactory[Request]()),
				WithRetry(configretry.NewDefaultBackOffConfig()),
			},
		},
		{
			name: "WithRequestQueue/PersistentQueueFactory",
			queueOptions: []Option{
				WithRequestQueue(exporterqueue.Config{
					Enabled:      true,
					QueueSize:    10,
					NumConsumers: 1,
				}, exporterqueue.NewPersistentQueueFactory[Request](nil, exporterqueue.PersistentQueueSettings[Request]{})),
				WithRetry(configretry.NewDefaultBackOffConfig()),
			},
		},
		{
			name: "WithRequestQueue/PersistentQueueFactory/RequestsLimit",
			queueOptions: []Option{
				WithRequestQueue(exporterqueue.Config{
					Enabled:      true,
					QueueSize:    10,
					NumConsumers: 1,
				}, exporterqueue.NewPersistentQueueFactory[Request](nil, exporterqueue.PersistentQueueSettings[Request]{})),
				WithRetry(configretry.NewDefaultBackOffConfig()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := componenttest.SetupTelemetry(defaultID)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

			set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tel.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
			be, err := newBaseExporter(set, defaultType, newObservabilityConsumerSender, tt.queueOptions...)
			require.NoError(t, err)
			ocs := be.obsrepSender.(*observabilityConsumerSender)

			wantRequests := 10
			reqs := make([]*mockRequest, 0, 10)
			for i := 0; i < wantRequests; i++ {
				ocs.run(func() {
					req := newMockRequest(2, nil)
					reqs = append(reqs, req)
					require.NoError(t, be.send(context.Background(), req))
				})
			}

			// expect queue to be full
			require.Error(t, be.send(context.Background(), newMockRequest(2, nil)))

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				assert.NoError(t, be.Shutdown(context.Background()))
			})

			// Wait until all batches received
			ocs.awaitAsyncProcessing()

			require.Len(t, reqs, wantRequests)
			for _, req := range reqs {
				req.checkNumRequests(t, 1)
			}

			ocs.checkSendItemsCount(t, 2*wantRequests)
			ocs.checkDroppedItemsCount(t, 0)
		})
	}
}
func TestQueuedRetry_QueueMetricsReported(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(defaultID)
	require.NoError(t, err)

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, defaultType, newObservabilityConsumerSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, tt.CheckExporterMetricGauge("exporter_queue_capacity", int64(defaultQueueSize)))

	for i := 0; i < 7; i++ {
		require.NoError(t, be.send(context.Background(), newErrorRequest()))
	}
	require.NoError(t, tt.CheckExporterMetricGauge("exporter_queue_size", int64(7)))

	assert.NoError(t, be.Shutdown(context.Background()))
}

func TestNoCancellationContext(t *testing.T) {
	deadline := time.Now().Add(1 * time.Second)
	ctx, cancelFunc := context.WithDeadline(context.Background(), deadline)
	cancelFunc()
	require.Error(t, ctx.Err())
	d, ok := ctx.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, d)

	nctx := noCancellationContext{Context: ctx}
	assert.NoError(t, nctx.Err())
	d, ok = nctx.Deadline()
	assert.False(t, ok)
	assert.True(t, d.IsZero())
}

func TestQueueSettings_Validate(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	assert.NoError(t, qCfg.Validate())

	qCfg.QueueSize = 0
	assert.EqualError(t, qCfg.Validate(), "queue size must be positive")

	qCfg = NewDefaultQueueSettings()
	qCfg.NumConsumers = 0

	assert.EqualError(t, qCfg.Validate(), "number of queue consumers must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}

func TestQueueRetryWithDisabledQueue(t *testing.T) {
	tests := []struct {
		name         string
		queueOptions []Option
	}{
		{
			name: "WithQueue",
			queueOptions: []Option{
				withMarshaler(mockRequestMarshaler),
				withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
				func() Option {
					qs := NewDefaultQueueSettings()
					qs.Enabled = false
					return WithQueue(qs)
				}(),
			},
		},
		{
			name: "WithRequestQueue",
			queueOptions: []Option{
				func() Option {
					qs := exporterqueue.NewDefaultConfig()
					qs.Enabled = false
					return WithRequestQueue(qs, exporterqueue.NewMemoryQueueFactory[Request]())
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopCreateSettings()
			logger, observed := observer.New(zap.ErrorLevel)
			set.Logger = zap.New(logger)
			be, err := newBaseExporter(set, component.DataTypeLogs, newObservabilityConsumerSender, tt.queueOptions...)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			ocs := be.obsrepSender.(*observabilityConsumerSender)
			mockR := newMockRequest(2, errors.New("some error"))
			ocs.run(func() {
				require.Error(t, be.send(context.Background(), mockR))
			})
			assert.Len(t, observed.All(), 1)
			assert.Equal(t, "Exporting failed. Rejecting data. Try enabling sending_queue to survive temporary failures.", observed.All()[0].Message)
			ocs.awaitAsyncProcessing()
			mockR.checkNumRequests(t, 1)
			ocs.checkSendItemsCount(t, 0)
			ocs.checkDroppedItemsCount(t, 2)
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}

}

func TestQueueFailedRequestDropped(t *testing.T) {
	set := exportertest.NewNopCreateSettings()
	logger, observed := observer.New(zap.ErrorLevel)
	set.Logger = zap.New(logger)
	be, err := newBaseExporter(set, component.DataTypeLogs, newNoopObsrepSender,
		WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[Request]()))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	mockR := newMockRequest(2, errors.New("some error"))
	require.NoError(t, be.send(context.Background(), mockR))
	require.NoError(t, be.Shutdown(context.Background()))
	mockR.checkNumRequests(t, 1)
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
}

func TestQueuedRetryPersistenceEnabled(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, defaultType, newObservabilityConsumerSender,
		withMarshaler(mockRequestMarshaler), withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
	}
	host := &mockHost{ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	tt, err := componenttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, defaultType, newObservabilityConsumerSender, withMarshaler(mockRequestMarshaler),
		withUnmarshaler(mockRequestUnmarshaler(&mockRequest{})), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(storageError),
	}
	host := &mockHost{ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestQueuedRetryPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence to ensure data is re-queued on shutdown

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered

	mockReq := newErrorRequest()
	be, err := newBaseExporter(defaultSettings, defaultType, newNoopObsrepSender, withMarshaler(mockRequestMarshaler),
		withUnmarshaler(mockRequestUnmarshaler(mockReq)), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: queue.NewMockStorageExtension(nil),
	}
	host := &mockHost{ext: extensions}

	require.NoError(t, be.Start(context.Background(), host))

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, be.send(context.Background(), mockReq))

	// first wait for the item to be consumed from the queue
	assert.Eventually(t, func() bool {
		return be.queueSender.(*queueSender).queue.Size() == 0
	}, time.Second, 1*time.Millisecond)

	// shuts down the exporter, unsent data should be preserved as in-flight data in the persistent queue.
	require.NoError(t, be.Shutdown(context.Background()))

	// start the exporter again replacing the preserved mockRequest in the unmarshaler with a new one that doesn't fail.
	replacedReq := newMockRequest(1, nil)
	be, err = newBaseExporter(defaultSettings, defaultType, newNoopObsrepSender, withMarshaler(mockRequestMarshaler),
		withUnmarshaler(mockRequestUnmarshaler(replacedReq)), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), host))
	t.Cleanup(func() { require.NoError(t, be.Shutdown(context.Background())) })

	// wait for the item to be consumed from the queue
	replacedReq.checkNumRequests(t, 1)
}

func TestQueueSenderNoStartShutdown(t *testing.T) {
	queue := queue.NewBoundedMemoryQueue[Request](queue.MemoryQueueSettings[Request]{})
	qs := newQueueSender(queue, exportertest.NewNopCreateSettings(), 1, "")
	assert.NoError(t, qs.Shutdown(context.Background()))
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}
