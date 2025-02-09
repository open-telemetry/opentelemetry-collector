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
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/exporter/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/internal/storagetest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestQueuedBatchStopWhileWaiting(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			qCfg := exporterqueue.NewDefaultConfig()
			qCfg.NumConsumers = 1
			be, err := newQueueBatchExporter(qCfg, exporterbatcher.Config{})
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			sink := requesttest.NewSink()
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 4, Sink: sink, ExportErr: errors.New("transient error")}))
			// Enqueue another request to ensure when calling shutdown we drain the queue.
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3, Sink: sink, Delay: 100 * time.Millisecond}))
			require.LessOrEqual(t, int64(1), be.queue.Size())

			require.NoError(t, be.Shutdown(context.Background()))
			assert.EqualValues(t, 1, sink.RequestsCount())
			assert.EqualValues(t, 3, sink.ItemsCount())
			require.Zero(t, be.queue.Size())
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchDoNotPreserveCancellation(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			qCfg := exporterqueue.NewDefaultConfig()
			qCfg.NumConsumers = 1
			be, err := newQueueBatchExporter(qCfg, exporterbatcher.Config{})
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				resetFeatureGate()
			})

			ctx, cancelFunc := context.WithCancel(context.Background())
			cancelFunc()

			sink := requesttest.NewSink()
			require.NoError(t, be.Send(ctx, &requesttest.FakeRequest{Items: 4, Sink: sink}))
			require.NoError(t, be.Shutdown(context.Background()))

			assert.EqualValues(t, 1, sink.RequestsCount())
			assert.EqualValues(t, 4, sink.ItemsCount())
			require.Zero(t, be.queue.Size())
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchHappyPath(t *testing.T) {
	tests := []struct {
		name string
		qCfg exporterqueue.Config
		qf   exporterqueue.Factory[internal.Request]
	}{
		{
			name: "WithRequestQueue/MemoryQueueFactory",
			qCfg: exporterqueue.Config{
				Enabled:      true,
				QueueSize:    10,
				NumConsumers: 1,
			},
			qf: exporterqueue.NewMemoryQueueFactory[internal.Request](),
		},
		{
			name: "WithRequestQueue/PersistentQueueFactory",
			qCfg: exporterqueue.Config{
				Enabled:      true,
				QueueSize:    10,
				NumConsumers: 1,
			},
			qf: exporterqueue.NewPersistentQueueFactory[internal.Request](nil, exporterqueue.PersistentQueueSettings[internal.Request]{}),
		},
	}

	runTest := func(testName string, enableQueueBatcher bool, tt struct {
		name string
		qCfg exporterqueue.Config
		qf   exporterqueue.Factory[internal.Request]
	},
	) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			be, err := newQueueBatchExporter(tt.qCfg, exporterbatcher.Config{})
			require.NoError(t, err)

			sink := requesttest.NewSink()
			for i := 0; i < 10; i++ {
				require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: i, Sink: sink}))
			}

			// expect queue to be full
			require.Error(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, Sink: sink}))

			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() {
				assert.NoError(t, be.Shutdown(context.Background()))
				resetFeatureGate()
			})

			assert.Eventually(t, func() bool {
				return sink.RequestsCount() == 10 && sink.ItemsCount() == 45
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
	for _, tt := range tests {
		runTest(tt.name+"_enable_queue_batcher", true, tt)
		runTest(tt.name+"_disable_queue_batcher", false, tt)
	}
}

func TestQueueConfig_Validate(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
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
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueFailedRequestDropped(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			qSet := exporterqueue.Settings{
				Signal:           defaultSignal,
				ExporterSettings: defaultSettings,
			}
			logger, observed := observer.New(zap.ErrorLevel)
			qSet.ExporterSettings.Logger = zap.New(logger)
			be, err := NewQueueSender(
				exporterqueue.NewMemoryQueueFactory[internal.Request](),
				qSet, exporterqueue.NewDefaultConfig(), exporterbatcher.Config{}, "", newNoopExportSender())

			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2, ExportErr: errors.New("some error")}))
			require.NoError(t, be.Shutdown(context.Background()))
			assert.Len(t, observed.All(), 1)
			assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchPersistenceEnabled(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			qSet := exporterqueue.Settings{
				Signal:           defaultSignal,
				ExporterSettings: defaultSettings,
			}
			storageID := component.MustNewIDWithName("file_storage", "storage")
			be, err := NewQueueSender(
				exporterqueue.NewPersistentQueueFactory[internal.Request](&storageID, exporterqueue.PersistentQueueSettings[internal.Request]{
					Marshaler:   mockRequestMarshaler,
					Unmarshaler: mockRequestUnmarshaler(&requesttest.FakeRequest{}),
				}),
				qSet, exporterqueue.NewDefaultConfig(), exporterbatcher.Config{}, "", newNoopExportSender())
			require.NoError(t, err)

			extensions := map[component.ID]component.Component{
				storageID: storagetest.NewMockStorageExtension(nil),
			}
			host := &MockHost{Ext: extensions}

			// we start correctly with a file storage extension
			require.NoError(t, be.Start(context.Background(), host))
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchPersistenceEnabledStorageError(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			storageError := errors.New("could not get storage client")

			qSet := exporterqueue.Settings{
				Signal:           defaultSignal,
				ExporterSettings: defaultSettings,
			}
			storageID := component.MustNewIDWithName("file_storage", "storage")
			be, err := NewQueueSender(
				exporterqueue.NewPersistentQueueFactory[internal.Request](&storageID, exporterqueue.PersistentQueueSettings[internal.Request]{
					Marshaler:   mockRequestMarshaler,
					Unmarshaler: mockRequestUnmarshaler(&requesttest.FakeRequest{}),
				}),
				qSet, exporterqueue.NewDefaultConfig(), exporterbatcher.Config{}, "", newNoopExportSender())
			require.NoError(t, err)

			extensions := map[component.ID]component.Component{
				storageID: storagetest.NewMockStorageExtension(storageError),
			}
			host := &MockHost{Ext: extensions}

			// we fail to start if we get an error creating the storage client
			require.Error(t, be.Start(context.Background(), host), "could not get storage client")
		})
	}

	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			resetFeatureGate := setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)
			qCfg := exporterqueue.NewDefaultConfig()
			qCfg.NumConsumers = 1

			rCfg := configretry.NewDefaultBackOffConfig()
			rCfg.InitialInterval = time.Millisecond
			rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered
			rs := newRetrySender(rCfg, defaultSettings, newNoopExportSender())
			require.NoError(t, rs.Start(context.Background(), componenttest.NewNopHost()))

			qSet := exporterqueue.Settings{
				Signal:           defaultSignal,
				ExporterSettings: defaultSettings,
			}
			storageID := component.MustNewIDWithName("file_storage", "storage")
			mockReq := newErrorRequest(errors.New("transient error"))
			be, err := NewQueueSender(
				exporterqueue.NewPersistentQueueFactory[internal.Request](&storageID, exporterqueue.PersistentQueueSettings[internal.Request]{
					Marshaler:   mockRequestMarshaler,
					Unmarshaler: mockRequestUnmarshaler(mockReq),
				}),
				qSet, qCfg, exporterbatcher.Config{}, "", rs)
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
				return be.queue.Size() == 0
			}, 1*time.Second, 10*time.Millisecond)

			// shuts down the exporter, unsent data should be preserved as in-flight data in the persistent queue.
			require.NoError(t, rs.Shutdown(context.Background()))
			require.NoError(t, be.Shutdown(context.Background()))

			// start the exporter again replacing the preserved mockRequest in the unmarshaler with a new one that doesn't fail.
			sink := requesttest.NewSink()
			replacedReq := &requesttest.FakeRequest{Items: 7, Sink: sink}
			be, err = NewQueueSender(
				exporterqueue.NewPersistentQueueFactory[internal.Request](&storageID, exporterqueue.PersistentQueueSettings[internal.Request]{
					Marshaler:   mockRequestMarshaler,
					Unmarshaler: mockRequestUnmarshaler(replacedReq),
				}),
				qSet, qCfg, exporterbatcher.Config{}, "", newNoopExportSender())
			require.NoError(t, err)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), host))
			t.Cleanup(func() {
				require.NoError(t, be.Shutdown(context.Background()))
				resetFeatureGate()
			})

			assert.Eventually(t, func() bool {
				return sink.ItemsCount() == 7 && sink.RequestsCount() == 1
			}, 1*time.Second, 10*time.Millisecond)
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func TestQueueBatchNoStartShutdown(t *testing.T) {
	runTest := func(testName string, enableQueueBatcher bool) {
		t.Run(testName, func(t *testing.T) {
			defer setFeatureGateForTest(t, usePullingBasedExporterQueueBatcher, enableQueueBatcher)()
			set := exportertest.NewNopSettings()
			set.ID = exporterID
			qSet := exporterqueue.Settings{
				Signal:           pipeline.SignalTraces,
				ExporterSettings: set,
			}
			qs, err := NewQueueSender(exporterqueue.NewMemoryQueueFactory[internal.Request](), qSet, exporterqueue.NewDefaultConfig(), exporterbatcher.NewDefaultConfig(), "", newNoopExportSender())
			require.NoError(t, err)
			assert.NoError(t, qs.Shutdown(context.Background()))
		})
	}
	runTest("enable_queue_batcher", true)
	runTest("disable_queue_batcher", false)
}

func newQueueBatchExporter(qCfg exporterqueue.Config, bCfg exporterbatcher.Config) (*QueueSender, error) {
	qSet := exporterqueue.Settings{
		Signal:           defaultSignal,
		ExporterSettings: defaultSettings,
	}
	return NewQueueSender(exporterqueue.NewMemoryQueueFactory[internal.Request](), qSet, qCfg, bCfg, "", newNoopExportSender())
}
