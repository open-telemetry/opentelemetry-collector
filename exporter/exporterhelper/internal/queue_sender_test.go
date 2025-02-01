// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/storagetest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	firstMockR := newErrorRequest()
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), firstMockR))
	})

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	secondMockR := newMockRequest(3, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(context.Background(), secondMockR))
	})

	require.LessOrEqual(t, int64(1), be.QueueSender.(*QueueSender).queue.Size())

	require.NoError(t, be.Shutdown(context.Background()))

	secondMockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 3)
	ocs.checkDroppedItemsCount(t, 7)
	require.Zero(t, be.QueueSender.(*QueueSender).queue.Size())
}

func TestQueuedRetry_DoNotPreserveCancellation(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.ObsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	mockR := newMockRequest(2, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.Send(ctx, mockR))
	})
	ocs.awaitAsyncProcessing()

	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.QueueSender.(*QueueSender).queue.Size())
}

func TestQueuedRetry_RejectOnFull(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.QueueSize = 0
	qCfg.NumConsumers = 0
	set := exportertest.NewNopSettings()
	logger, observed := observer.New(zap.ErrorLevel)
	set.Logger = zap.New(logger)
	be, err := NewBaseExporter(set, defaultSignal, newNoopObsrepSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	require.Error(t, be.Send(context.Background(), newMockRequest(2, nil)))
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
				WithMarshaler(mockRequestMarshaler),
				WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
				WithQueue(QueueConfig{
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
				}, exporterqueue.NewMemoryQueueFactory[internal.Request]()),
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
				}, exporterqueue.NewPersistentQueueFactory[internal.Request](nil, exporterqueue.PersistentQueueSettings[internal.Request]{})),
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
				}, exporterqueue.NewPersistentQueueFactory[internal.Request](nil, exporterqueue.PersistentQueueSettings[internal.Request]{})),
				WithRetry(configretry.NewDefaultBackOffConfig()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tel, err := componenttest.SetupTelemetry(defaultID)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, tel.Shutdown(context.Background())) })

			set := exporter.Settings{ID: defaultID, TelemetrySettings: tel.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
			be, err := NewBaseExporter(set, defaultSignal, newObservabilityConsumerSender, tt.queueOptions...)
			require.NoError(t, err)
			ocs := be.ObsrepSender.(*observabilityConsumerSender)

			wantRequests := 10
			reqs := make([]*mockRequest, 0, 10)
			for i := 0; i < wantRequests; i++ {
				ocs.run(func() {
					req := newMockRequest(2, nil)
					reqs = append(reqs, req)
					require.NoError(t, be.Send(context.Background(), req))
				})
			}

			// expect queue to be full
			require.Error(t, be.Send(context.Background(), newMockRequest(2, nil)))

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
	dataTypes := []pipeline.Signal{pipeline.SignalLogs, pipeline.SignalTraces, pipeline.SignalMetrics}
	for _, dataType := range dataTypes {
		tt, err := componenttest.SetupTelemetry(defaultID)
		require.NoError(t, err)

		qCfg := NewDefaultQueueConfig()
		qCfg.NumConsumers = -1 // to make QueueMetricsReportedvery request go straight to the queue
		rCfg := configretry.NewDefaultBackOffConfig()
		set := exporter.Settings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
		be, err := NewBaseExporter(set, dataType, newObservabilityConsumerSender,
			WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
			WithRetry(rCfg), WithQueue(qCfg))
		require.NoError(t, err)
		require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

		metadatatest.AssertEqualExporterQueueCapacity(t, tt.Telemetry,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", defaultID.String())),
					Value: int64(1000),
				},
			}, metricdatatest.IgnoreTimestamp())

		for i := 0; i < 7; i++ {
			require.NoError(t, be.Send(context.Background(), newErrorRequest()))
		}
		metadatatest.AssertEqualExporterQueueSize(t, tt.Telemetry,
			[]metricdata.DataPoint[int64]{
				{
					Attributes: attribute.NewSet(
						attribute.String("exporter", defaultID.String()),
						attribute.String(DataTypeKey, dataType.String())),
					Value: int64(7),
				},
			}, metricdatatest.IgnoreTimestamp())

		assert.NoError(t, be.Shutdown(context.Background()))

		// metrics should be unregistered at shutdown to prevent memory leak
		var md metricdata.ResourceMetrics
		require.NoError(t, tt.Reader.Collect(context.Background(), &md))
		assert.Empty(t, md.ScopeMetrics)
	}
}

func TestNoCancellationContext(t *testing.T) {
	deadline := time.Now().Add(1 * time.Second)
	ctx, cancelFunc := context.WithDeadline(context.Background(), deadline)
	cancelFunc()
	require.Error(t, ctx.Err())
	d, ok := ctx.Deadline()
	require.True(t, ok)
	require.Equal(t, deadline, d)

	nctx := context.WithoutCancel(ctx)
	require.NoError(t, nctx.Err())
	d, ok = nctx.Deadline()
	assert.False(t, ok)
	assert.True(t, d.IsZero())
}

func TestQueueConfig_Validate(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	require.NoError(t, qCfg.Validate())

	qCfg.QueueSize = 0
	require.EqualError(t, qCfg.Validate(), "`queue_size` must be positive")

	qCfg = NewDefaultQueueConfig()
	qCfg.NumConsumers = 0

	require.EqualError(t, qCfg.Validate(), "`num_consumers` must be positive")

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
				WithMarshaler(mockRequestMarshaler),
				WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
				func() Option {
					qs := NewDefaultQueueConfig()
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
					return WithRequestQueue(qs, exporterqueue.NewMemoryQueueFactory[internal.Request]())
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings()
			logger, observed := observer.New(zap.ErrorLevel)
			set.Logger = zap.New(logger)
			be, err := NewBaseExporter(set, pipeline.SignalLogs, newObservabilityConsumerSender, tt.queueOptions...)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			ocs := be.ObsrepSender.(*observabilityConsumerSender)
			mockR := newMockRequest(2, errors.New("some error"))
			ocs.run(func() {
				require.Error(t, be.Send(context.Background(), mockR))
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
	set := exportertest.NewNopSettings()
	logger, observed := observer.New(zap.ErrorLevel)
	set.Logger = zap.New(logger)
	be, err := NewBaseExporter(set, pipeline.SignalLogs, newNoopObsrepSender,
		WithRequestQueue(exporterqueue.NewDefaultConfig(), exporterqueue.NewMemoryQueueFactory[internal.Request]()))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	mockR := newMockRequest(2, errors.New("some error"))
	require.NoError(t, be.Send(context.Background(), mockR))
	require.NoError(t, be.Shutdown(context.Background()))
	mockR.checkNumRequests(t, 1)
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
}

func TestQueuedRetryPersistenceEnabled(t *testing.T) {
	tt, err := componenttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.Settings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
	be, err := NewBaseExporter(set, defaultSignal, newObservabilityConsumerSender,
		WithMarshaler(mockRequestMarshaler), WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})),
		WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	}
	host := &MockHost{Ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	tt, err := componenttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueConfig()
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.Settings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings(), BuildInfo: component.NewDefaultBuildInfo()}
	be, err := NewBaseExporter(set, defaultSignal, newObservabilityConsumerSender, WithMarshaler(mockRequestMarshaler),
		WithUnmarshaler(mockRequestUnmarshaler(&mockRequest{})), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(storageError),
	}
	host := &MockHost{Ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestQueuedRetryPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	qCfg.NumConsumers = 1
	storageID := component.MustNewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence to ensure data is re-queued on shutdown

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered

	mockReq := newErrorRequest()
	be, err := NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender, WithMarshaler(mockRequestMarshaler),
		WithUnmarshaler(mockRequestUnmarshaler(mockReq)), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	extensions := map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(nil),
	}
	host := &MockHost{Ext: extensions}

	require.NoError(t, be.Start(context.Background(), host))

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, be.Send(context.Background(), mockReq))

	// first wait for the item to be consumed from the queue
	assert.Eventually(t, func() bool {
		return be.QueueSender.(*QueueSender).queue.Size() == 0
	}, time.Second, 1*time.Millisecond)

	// shuts down the exporter, unsent data should be preserved as in-flight data in the persistent queue.
	require.NoError(t, be.Shutdown(context.Background()))

	// start the exporter again replacing the preserved mockRequest in the unmarshaler with a new one that doesn't fail.
	replacedReq := newMockRequest(1, nil)
	be, err = NewBaseExporter(defaultSettings, defaultSignal, newNoopObsrepSender, WithMarshaler(mockRequestMarshaler),
		WithUnmarshaler(mockRequestUnmarshaler(replacedReq)), WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), host))
	t.Cleanup(func() {
		require.NoError(t, be.Shutdown(context.Background()))
	})

	// wait for the item to be consumed from the queue
	replacedReq.checkNumRequests(t, 1)
}

func TestQueueSenderNoStartShutdown(t *testing.T) {
	set := exportertest.NewNopSettings()
	set.ID = exporterID
	queue := exporterqueue.NewMemoryQueueFactory[internal.Request]()(
		context.Background(),
		exporterqueue.Settings{
			Signal:           pipeline.SignalTraces,
			ExporterSettings: set,
		},
		exporterqueue.NewDefaultConfig())
	obsrep, err := NewObsReport(ObsReportSettings{
		ExporterSettings: set,
		Signal:           pipeline.SignalTraces,
	})
	require.NoError(t, err)
	qs := NewQueueSender(queue, set, 1, "", obsrep, exporterbatcher.NewDefaultConfig())
	assert.NoError(t, qs.Shutdown(context.Background()))
}
