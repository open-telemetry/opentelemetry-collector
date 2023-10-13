// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	firstMockR := newErrorRequest(context.Background())
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(firstMockR))
	})

	// Enqueue another request to ensure when calling shutdown we drain the queue.
	secondMockR := newMockRequest(context.Background(), 3, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(secondMockR))
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
	rCfg := NewDefaultRetrySettings()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	ctx, cancelFunc := context.WithCancel(context.Background())
	cancelFunc()
	mockR := newMockRequest(ctx, 2, nil)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()

	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 2)
	ocs.checkDroppedItemsCount(t, 0)
	require.Zero(t, be.queueSender.(*queueSender).queue.Size())
}

func TestQueuedRetry_DropOnFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.QueueSize = 0
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	require.Error(t, be.send(newMockRequest(context.Background(), 2, nil)))
}

func TestQueuedRetryHappyPath(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	rCfg := NewDefaultRetrySettings()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	wantRequests := 10
	reqs := make([]*mockRequest, 0, 10)
	for i := 0; i < wantRequests; i++ {
		ocs.run(func() {
			req := newMockRequest(context.Background(), 2, nil)
			reqs = append(reqs, req)
			require.NoError(t, be.send(req))
		})
	}

	// Wait until all batches received
	ocs.awaitAsyncProcessing()

	require.Len(t, reqs, wantRequests)
	for _, req := range reqs {
		req.checkNumRequests(t, 1)
	}

	ocs.checkSendItemsCount(t, 2*wantRequests)
	ocs.checkDroppedItemsCount(t, 0)
}

// Force the state of feature gate for a test
func setFeatureGateForTest(t testing.TB, gate *featuregate.Gate, enabled bool) func() {
	originalValue := gate.IsEnabled()
	require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
	return func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), originalValue))
	}
}

func TestQueuedRetry_QueueMetricsReported(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := NewDefaultRetrySettings()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	checkValueForGlobalManager(t, defaultExporterTags, int64(defaultQueueSize), "exporter/queue_capacity")
	for i := 0; i < 7; i++ {
		require.NoError(t, be.send(newErrorRequest(context.Background())))
	}
	checkValueForGlobalManager(t, defaultExporterTags, int64(7), "exporter/queue_size")

	assert.NoError(t, be.Shutdown(context.Background()))
	checkValueForGlobalManager(t, defaultExporterTags, int64(0), "exporter/queue_size")
}

func TestQueuedRetry_QueueMetricsReportedUsingOTel(t *testing.T) {
	resetFlag := setFeatureGateForTest(t, obsreportconfig.UseOtelForInternalMetricsfeatureGate, true)
	defer resetFlag()

	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := NewDefaultRetrySettings()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	require.NoError(t, tt.CheckExporterMetricGauge("exporter_queue_capacity", int64(defaultQueueSize)))

	for i := 0; i < 7; i++ {
		require.NoError(t, be.send(newErrorRequest(context.Background())))
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

// if requeueing is enabled, we eventually retry even if we failed at first
func TestQueuedRetry_RequeuingEnabled(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	be.queueSender.(*queueSender).requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.NoError(t, be.send(mockR))
		ocs.waitGroup.Add(1) // necessary because we'll call send() again after requeueing
	})
	ocs.awaitAsyncProcessing()

	// In the newMockConcurrentExporter we count requests and items even for failed requests
	mockR.checkNumRequests(t, 2)
	ocs.checkSendItemsCount(t, 1)
	ocs.checkDroppedItemsCount(t, 1) // not actually dropped, but ocs counts each failed send here
}

// if requeueing is enabled, but the queue is full, we get an error
func TestQueuedRetry_RequeuingEnabledQueueFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0
	qCfg.QueueSize = 0
	rCfg := NewDefaultRetrySettings()
	rCfg.MaxElapsedTime = time.Nanosecond // we don't want to retry at all, but requeue instead
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	be.queueSender.(*queueSender).requeuingEnabled = true
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})

	traceErr := consumererror.NewTraces(errors.New("some error"), testdata.GenerateTraces(1))
	mockR := newMockRequest(context.Background(), 1, traceErr)

	ocs := be.obsrepSender.(*observabilityConsumerSender)
	ocs.run(func() {
		require.Error(t, be.retrySender.send(mockR), "sending_queue is full")
	})
	mockR.checkNumRequests(t, 1)
}

func TestQueueRetryWithDisabledQueue(t *testing.T) {
	qs := NewDefaultQueueSettings()
	qs.Enabled = false
	be, err := newBaseExporter(exportertest.NewNopCreateSettings(), component.DataTypeLogs, false, nil, nil, newObservabilityConsumerSender, WithQueue(qs))
	require.Nil(t, be.queueSender.(*queueSender).queue)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(context.Background(), 2, errors.New("some error"))
	ocs.run(func() {
		// This is asynchronous so it should just enqueue, no errors expected.
		require.Error(t, be.send(mockR))
	})
	ocs.awaitAsyncProcessing()
	mockR.checkNumRequests(t, 1)
	ocs.checkSendItemsCount(t, 0)
	ocs.checkDroppedItemsCount(t, 2)
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabled(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(nil),
	}
	host := &mockHost{ext: extensions}

	// we start correctly with a file storage extension
	require.NoError(t, be.Start(context.Background(), host))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestQueuedRetryPersistenceEnabledStorageError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence
	rCfg := NewDefaultRetrySettings()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, mockRequestMarshaler, mockRequestUnmarshaler(&mockRequest{}), newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(storageError),
	}
	host := &mockHost{ext: extensions}

	// we fail to start if we get an error creating the storage client
	require.Error(t, be.Start(context.Background(), host), "could not get storage client")
}

func TestQueuedRetryPersistentEnabled_shutdown_dataIsRequeued(t *testing.T) {

	produceCounter := &atomic.Uint32{}

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := NewDefaultRetrySettings()
	rCfg.InitialInterval = 100 * time.Millisecond
	rCfg.MaxInterval = rCfg.InitialInterval
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered

	set := exportertest.NewNopCreateSettings()
	set.ID = defaultID
	set.Logger, _ = zap.NewDevelopment()
	be, err := newBaseExporter(set, "", false, nil, nil, newNoopObsrepSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), &mockHost{}))

	// wraps original queue so we can count operations
	be.queueSender.(*queueSender).queue = &producerConsumerQueueWithCounter{
		ProducerConsumerQueue: be.queueSender.(*queueSender).queue,
		produceCounter:        produceCounter,
	}
	be.queueSender.(*queueSender).requeuingEnabled = true

	// Invoke queuedRetrySender so the producer will put the item for consumer to poll
	require.NoError(t, be.send(newErrorRequest(context.Background())))
	// first wait for the item to be produced to the queue initially
	assert.True(t, produceCounter.Load() == uint32(1))
	// buffer some time so the failure is pushed back.
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, be.Shutdown(context.Background()))
	require.Eventually(t, func() bool {
		return produceCounter.Load() == uint32(2)
	}, 1*time.Second, 1*time.Millisecond)
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}
