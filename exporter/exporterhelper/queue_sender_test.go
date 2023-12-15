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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/obsreportconfig"
	"go.opentelemetry.io/collector/obsreport/obsreporttest"
)

func TestQueuedRetry_StopWhileWaiting(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	rCfg := configretry.NewDefaultBackOffConfig()
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
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
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
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

func TestQueuedRetry_DropOnFull(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.QueueSize = 0
	qCfg.NumConsumers = 0
	be, err := newBaseExporter(defaultSettings, "", false, nil, nil, newNoopObsrepSender, WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() {
		assert.NoError(t, be.Shutdown(context.Background()))
	})
	require.Error(t, be.send(context.Background(), newMockRequest(2, nil)))
}

func TestQueuedRetryHappyPath(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	qCfg := NewDefaultQueueSettings()
	rCfg := configretry.NewDefaultBackOffConfig()
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
			req := newMockRequest(2, nil)
			reqs = append(reqs, req)
			require.NoError(t, be.send(context.Background(), req))
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
	resetFlag := setFeatureGateForTest(t, obsreportconfig.UseOtelForInternalMetricsfeatureGate, false)
	defer resetFlag()
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))

	checkValueForGlobalManager(t, defaultExporterTags, int64(defaultQueueSize), "exporter/queue_capacity")
	for i := 0; i < 7; i++ {
		require.NoError(t, be.send(context.Background(), newErrorRequest()))
	}
	checkValueForGlobalManager(t, defaultExporterTags, int64(7), "exporter/queue_size")

	assert.NoError(t, be.Shutdown(context.Background()))
	checkValueForGlobalManager(t, defaultExporterTags, int64(0), "exporter/queue_size")
}

func TestQueuedRetry_QueueMetricsReportedUsingOTel(t *testing.T) {
	tt, err := obsreporttest.SetupTelemetry(defaultID)
	require.NoError(t, err)

	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 0 // to make every request go straight to the queue
	rCfg := configretry.NewDefaultBackOffConfig()
	set := exporter.CreateSettings{ID: defaultID, TelemetrySettings: tt.TelemetrySettings, BuildInfo: component.NewDefaultBuildInfo()}
	be, err := newBaseExporter(set, "", false, nil, nil, newObservabilityConsumerSender, WithRetry(rCfg), WithQueue(qCfg))
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
	qs := NewDefaultQueueSettings()
	qs.Enabled = false
	be, err := newBaseExporter(exportertest.NewNopCreateSettings(), component.DataTypeLogs, false, nil, nil, newObservabilityConsumerSender, WithQueue(qs))
	require.IsType(t, &errorLoggingRequestSender{}, be.queueSender)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	ocs := be.obsrepSender.(*observabilityConsumerSender)
	mockR := newMockRequest(2, errors.New("some error"))
	ocs.run(func() {
		require.Error(t, be.send(context.Background(), mockR))
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
	rCfg := configretry.NewDefaultBackOffConfig()
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
	rCfg := configretry.NewDefaultBackOffConfig()
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

func TestQueuedRetryPersistentEnabled_NoDataLossOnShutdown(t *testing.T) {
	qCfg := NewDefaultQueueSettings()
	qCfg.NumConsumers = 1
	storageID := component.NewIDWithName("file_storage", "storage")
	qCfg.StorageID = &storageID // enable persistence to ensure data is re-queued on shutdown

	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.InitialInterval = time.Millisecond
	rCfg.MaxElapsedTime = 0 // retry infinitely so shutdown can be triggered

	mockReq := newErrorRequest()
	be, err := newBaseExporter(defaultSettings, "", false, mockRequestMarshaler, mockRequestUnmarshaler(mockReq),
		newNoopObsrepSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)

	var extensions = map[component.ID]component.Component{
		storageID: internal.NewMockStorageExtension(nil),
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
	be, err = newBaseExporter(defaultSettings, "", false, mockRequestMarshaler, mockRequestUnmarshaler(replacedReq),
		newNoopObsrepSender, WithRetry(rCfg), WithQueue(qCfg))
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), host))

	// wait for the item to be consumed from the queue
	replacedReq.checkNumRequests(t, 1)
}

func TestQueueSenderNoStartShutdown(t *testing.T) {
	qs := newQueueSender(NewDefaultQueueSettings(), exportertest.NewNopCreateSettings(), "", nil, nil)
	assert.NoError(t, qs.Shutdown(context.Background()))
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (nh *mockHost) GetExtensions() map[component.ID]component.Component {
	return nh.ext
}
