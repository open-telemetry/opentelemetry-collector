// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/hosttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/storagetest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/otel/codes"
)

func TestBaseExporter(t *testing.T) {
	be, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport)
	require.NoError(t, err)
	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Shutdown(context.Background()))
}

func TestBaseExporterWithOptions(t *testing.T) {
	want := errors.New("my error")
	be, err := NewBaseExporter(
		exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithTimeout(NewDefaultTimeoutConfig()),
	)
	require.NoError(t, err)
	require.Equal(t, want, be.Start(context.Background(), componenttest.NewNopHost()))
	require.Equal(t, want, be.Shutdown(context.Background()))
}

func TestQueueOptionsWithRequestExporter(t *testing.T) {
	bs, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithRetry(configretry.NewDefaultBackOffConfig()))
	require.NoError(t, err)
	require.Nil(t, bs.queueBatchSettings.Encoding)
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithRetry(configretry.NewDefaultBackOffConfig()), WithQueue(NewDefaultQueueConfig()))
	require.Error(t, err)

	qCfg := NewDefaultQueueConfig()
	storageID := component.NewID(component.MustNewType("test"))
	qCfg.StorageID = &storageID
	_, err = NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithRetry(configretry.NewDefaultBackOffConfig()),
		WithQueueBatch(qCfg, QueueBatchSettings[request.Request]{}))
	require.Error(t, err)
}

func TestBaseExporterLogging(t *testing.T) {
	set := exportertest.NewNopSettings(exportertest.NopType)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	qCfg := NewDefaultQueueConfig()
	qCfg.WaitForResult = true
	bs, err := NewBaseExporter(set, pipeline.SignalMetrics, errExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(qCfg),
		WithRetry(rCfg))
	require.NoError(t, err)
	require.NoError(t, bs.Start(context.Background(), componenttest.NewNopHost()))
	sendErr := bs.Send(context.Background(), &requesttest.FakeRequest{Items: 2})
	require.Error(t, sendErr)

	errorLogs := observed.FilterLevelExact(zap.ErrorLevel).All()
	require.Len(t, errorLogs, 2)
	assert.Contains(t, errorLogs[0].Message, "Exporting failed. Dropping data.")
	assert.Equal(t, "my error", errorLogs[0].ContextMap()["error"])
	assert.Contains(t, errorLogs[1].Message, "Exporting failed. Rejecting data.")
	assert.Equal(t, "my error", errorLogs[1].ContextMap()["error"])
	require.NoError(t, bs.Shutdown(context.Background()))
}

func TestExporterStartupDebugLogging(t *testing.T) {
	tests := []struct {
		name             string
		queueEnabled     bool
		expectedLogs     int
		expectedMessages []string
	}{
		{
			name:         "queue enabled",
			queueEnabled: true,
			expectedLogs: 3,
			expectedMessages: []string{
				"Starting exporter",
				"Starting QueueSender",
				"Started exporter",
			},
		},
		{
			name:         "queue disabled",
			queueEnabled: false,
			expectedLogs: 2,
			expectedMessages: []string{
				"Starting exporter",
				"Started exporter",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings(exportertest.NopType)
			logger, observed := observer.New(zap.DebugLevel)
			set.Logger = zap.New(logger)
			rCfg := configretry.NewDefaultBackOffConfig()
			rCfg.Enabled = false
			qCfg := NewDefaultQueueConfig()
			qCfg.Enabled = tt.queueEnabled

			bs, err := NewBaseExporter(set, pipeline.SignalMetrics, errExport,
				WithQueueBatchSettings(newFakeQueueBatch()),
				WithQueue(qCfg),
				WithRetry(rCfg))
			require.NoError(t, err)
			require.NoError(t, bs.Start(context.Background(), componenttest.NewNopHost()))

			// Debug Logs for Startup
			debugLogs := observed.FilterLevelExact(zap.DebugLevel).All()
			require.Len(t, debugLogs, tt.expectedLogs)

			for i, expectedMessage := range tt.expectedMessages {
				assert.Contains(t, debugLogs[i].Message, expectedMessage)
				assert.Contains(t, debugLogs[i].Context[0].Key, "exporter")
				assert.Contains(t, debugLogs[i].Context[0].String, set.ID.String())
			}

			require.NoError(t, bs.Shutdown(context.Background()))
		})
	}
}

func TestExporterStartupQueueSenderError(t *testing.T) {
	storageError := errors.New("could not get storage client")
	set := exportertest.NewNopSettings(exportertest.NopType)
	logger, observed := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)
	rCfg := configretry.NewDefaultBackOffConfig()
	rCfg.Enabled = false
	qCfg := NewDefaultQueueConfig()
	qCfg.Enabled = true
	storageID := component.NewIDWithName(component.MustNewType("file_storage"), "storage")
	qCfg.StorageID = &storageID

	bs, err := NewBaseExporter(set, pipeline.SignalMetrics, errExport,
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(qCfg),
		WithRetry(rCfg))
	require.NoError(t, err)

	// Create a host with a mock storage extension that returns an error
	host := hosttest.NewHost(map[component.ID]component.Component{
		storageID: storagetest.NewMockStorageExtension(storageError),
	})

	// Start should fail due to QueueSender error
	startErr := bs.Start(context.Background(), host)
	require.Error(t, startErr)
	assert.Contains(t, startErr.Error(), "could not get storage client")

	// Check debug logs
	debugLogs := observed.FilterLevelExact(zap.DebugLevel).All()
	require.Len(t, debugLogs, 2)
	assert.Contains(t, debugLogs[0].Message, "Starting exporter")
	assert.Contains(t, debugLogs[1].Message, "Starting QueueSender")

	// Check error logs
	errorLogs := observed.FilterLevelExact(zap.ErrorLevel).All()
	require.Len(t, errorLogs, 1)
	assert.Contains(t, errorLogs[0].Message, "Failed to start QueueSender")
	assert.Contains(t, errorLogs[0].ContextMap()["error"], "could not get storage client")
	assert.Contains(t, errorLogs[0].ContextMap()["exporter"], set.ID.String())
}

func TestExporterStartupTraceSpans(t *testing.T) {
	// Create telemetry with span recorder
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	set := exportertest.NewNopSettings(exportertest.NopType)
	set.TelemetrySettings = tt.NewTelemetrySettings()
	logger, _ := observer.New(zap.DebugLevel)
	set.Logger = zap.New(logger)

	// Test case 1: Successful start with QueueSender
	t.Run("successful_start_with_queue", func(t *testing.T) {
		// Reset span recorder
		tt.SpanRecorder.Reset()

		rCfg := configretry.NewDefaultBackOffConfig()
		rCfg.Enabled = false
		qCfg := NewDefaultQueueConfig()
		qCfg.Enabled = true
		storageID := component.NewIDWithName(component.MustNewType("file_storage"), "storage")
		qCfg.StorageID = &storageID

		bs, err := NewBaseExporter(set, pipeline.SignalMetrics, noopExport,
			WithQueueBatchSettings(newFakeQueueBatch()),
			WithQueue(qCfg),
			WithRetry(rCfg))
		require.NoError(t, err)

		// Create a host with a mock storage extension that succeeds
		host := hosttest.NewHost(map[component.ID]component.Component{
			storageID: storagetest.NewMockStorageExtension(nil),
		})

		// Start should succeed
		startErr := bs.Start(context.Background(), host)
		require.NoError(t, startErr)

		// Verify spans were recorded
		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, 2)

		// Find spans by name
		var startSpan, queueSpan sdktrace.ReadOnlySpan
		for _, span := range spans {
			switch span.Name() {
			case "Start":
				startSpan = span
			case "StartQueueSender":
				queueSpan = span
			}
		}
		require.NotNil(t, startSpan, "Start span not found")
		require.NotNil(t, queueSpan, "StartQueueSender span not found")

		// Check the main "Start" span
		assert.Equal(t, "exporterhelper", startSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Unset, startSpan.Status().Code)

		// Check the "StartQueueSender" span
		assert.Equal(t, "exporterhelper", queueSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Unset, queueSpan.Status().Code)

		// Verify span hierarchy
		assert.Equal(t, startSpan.SpanContext(), queueSpan.Parent())

		require.NoError(t, bs.Shutdown(context.Background()))
	})

	// Test case 2: Successful start without QueueSender
	t.Run("successful_start_without_queue", func(t *testing.T) {
		// Reset span recorder
		tt.SpanRecorder.Reset()

		bs, err := NewBaseExporter(set, pipeline.SignalMetrics, noopExport)
		require.NoError(t, err)

		// Start should succeed
		startErr := bs.Start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, startErr)

		// Verify only one span was recorded (no QueueSender)
		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, 1)

		// Check the main "Start" span
		startSpan := spans[0]
		assert.Equal(t, "Start", startSpan.Name())
		assert.Equal(t, "exporterhelper", startSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Unset, startSpan.Status().Code)

		require.NoError(t, bs.Shutdown(context.Background()))
	})

	// Test case 3: Failed start due to QueueSender error
	t.Run("failed_start_queue_sender_error", func(t *testing.T) {
		// Reset span recorder
		tt.SpanRecorder.Reset()

		storageError := errors.New("could not get storage client")
		rCfg := configretry.NewDefaultBackOffConfig()
		rCfg.Enabled = false
		qCfg := NewDefaultQueueConfig()
		qCfg.Enabled = true
		storageID := component.NewIDWithName(component.MustNewType("file_storage"), "storage")
		qCfg.StorageID = &storageID

		bs, err := NewBaseExporter(set, pipeline.SignalMetrics, noopExport,
			WithQueueBatchSettings(newFakeQueueBatch()),
			WithQueue(qCfg),
			WithRetry(rCfg))
		require.NoError(t, err)

		// Create a host with a mock storage extension that returns an error
		host := hosttest.NewHost(map[component.ID]component.Component{
			storageID: storagetest.NewMockStorageExtension(storageError),
		})

		// Start should fail due to QueueSender error
		startErr := bs.Start(context.Background(), host)
		require.Error(t, startErr)
		assert.Contains(t, startErr.Error(), "could not get storage client")

		// Verify spans were recorded
		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, 2)

		// Find spans by name
		var startSpan, queueSpan sdktrace.ReadOnlySpan
		for _, span := range spans {
			switch span.Name() {
			case "Start":
				startSpan = span
			case "StartQueueSender":
				queueSpan = span
			}
		}
		require.NotNil(t, startSpan, "Start span not found")
		require.NotNil(t, queueSpan, "StartQueueSender span not found")

		// Check the main "Start" span
		assert.Equal(t, "exporterhelper", startSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Unset, startSpan.Status().Code)

		// Check the "StartQueueSender" span with error
		assert.Equal(t, "exporterhelper", queueSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Error, queueSpan.Status().Code)
		assert.Equal(t, storageError.Error(), queueSpan.Status().Description)

		// Verify span hierarchy
		assert.Equal(t, startSpan.SpanContext(), queueSpan.Parent())

	})

	// Test case 4: Failed start due to wrapped exporter error
	t.Run("failed_start_wrapped_exporter_error", func(t *testing.T) {
		// Reset span recorder
		tt.SpanRecorder.Reset()

		wrappedError := errors.New("wrapped exporter start failed")
		bs, err := NewBaseExporter(
			set, pipeline.SignalMetrics, noopExport,
			WithStart(func(context.Context, component.Host) error { return wrappedError }),
		)
		require.NoError(t, err)

		// Start should fail due to wrapped exporter error
		startErr := bs.Start(context.Background(), componenttest.NewNopHost())
		require.Error(t, startErr)
		assert.Equal(t, wrappedError, startErr)

		// Verify only one span was recorded (no QueueSender span created)
		spans := tt.SpanRecorder.Ended()
		require.Len(t, spans, 1)

		// Check the main "Start" span
		startSpan := spans[0]
		assert.Equal(t, "Start", startSpan.Name())
		assert.Equal(t, "exporterhelper", startSpan.InstrumentationLibrary().Name)
		assert.Equal(t, codes.Error, startSpan.Status().Code)
		assert.Equal(t, wrappedError.Error(), startSpan.Status().Description)
	})
}

func TestExporterShutdownDebugLogging(t *testing.T) {
	tests := []struct {
		name             string
		queueEnabled     bool
		retryEnabled     bool
		expectedLogs     int
		expectedMessages []string
	}{
		{
			name:         "queue and retry enabled",
			queueEnabled: true,
			retryEnabled: true,
			expectedLogs: 7,
			expectedMessages: []string{
				"Begin exporter shutdown sequence.",
				"Shutting down exporter retry sender...",
				"Shutdown exporter retry sender",
				"Shutting down queue sender...",
				"Shutdown queue sender",
				"Shutting down exporter...",
				"Shutdown exporter",
			},
		},
		{
			name:         "queue enabled, retry disabled",
			queueEnabled: true,
			retryEnabled: false,
			expectedLogs: 5,
			expectedMessages: []string{
				"Begin exporter shutdown sequence.",
				"Shutting down queue sender...",
				"Shutdown queue sender",
				"Shutting down exporter...",
				"Shutdown exporter",
			},
		},
		{
			name:         "queue disabled, retry enabled",
			queueEnabled: false,
			retryEnabled: true,
			expectedLogs: 5,
			expectedMessages: []string{
				"Begin exporter shutdown sequence.",
				"Shutting down exporter retry sender...",
				"Shutdown exporter retry sender",
				"Shutting down exporter...",
				"Shutdown exporter",
			},
		},
		{
			name:         "queue and retry disabled",
			queueEnabled: false,
			retryEnabled: false,
			expectedLogs: 3,
			expectedMessages: []string{
				"Begin exporter shutdown sequence.",
				"Shutting down exporter...",
				"Shutdown exporter",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings(exportertest.NopType)
			logger, observed := observer.New(zap.DebugLevel)
			set.Logger = zap.New(logger)
			rCfg := configretry.NewDefaultBackOffConfig()
			rCfg.Enabled = tt.retryEnabled
			qCfg := NewDefaultQueueConfig()
			qCfg.Enabled = tt.queueEnabled

			bs, err := NewBaseExporter(set, pipeline.SignalMetrics, errExport,
				WithQueueBatchSettings(newFakeQueueBatch()),
				WithQueue(qCfg),
				WithRetry(rCfg))
			require.NoError(t, err)
			require.NoError(t, bs.Start(context.Background(), componenttest.NewNopHost()))

			// Clear the startup logs
			observed.TakeAll()

			// Perform shutdown
			require.NoError(t, bs.Shutdown(context.Background()))

			// Debug Logs for Shutdown
			debugLogs := observed.FilterLevelExact(zap.DebugLevel).All()
			require.Len(t, debugLogs, tt.expectedLogs)

			for i, expectedMessage := range tt.expectedMessages {
				assert.Contains(t, debugLogs[i].Message, expectedMessage)
				assert.Contains(t, debugLogs[i].Context[0].Key, "exporter")
				assert.Contains(t, debugLogs[i].Context[0].String, set.ID.String())
			}
		})
	}
}

func TestQueueRetryWithDisabledQueue(t *testing.T) {
	tests := []struct {
		name         string
		queueOptions []Option
	}{
		{
			name: "WithQueue",
			queueOptions: []Option{
				WithQueueBatchSettings(newFakeQueueBatch()),
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
					qs := NewDefaultQueueConfig()
					qs.Enabled = false
					return WithQueueBatch(qs, newFakeQueueBatch())
				}(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := exportertest.NewNopSettings(exportertest.NopType)
			logger, observed := observer.New(zap.ErrorLevel)
			set.Logger = zap.New(logger)
			be, err := NewBaseExporter(set, pipeline.SignalLogs, errExport, tt.queueOptions...)
			require.NoError(t, err)
			require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
			mockR := &requesttest.FakeRequest{Items: 2}
			require.Error(t, be.Send(context.Background(), mockR))
			assert.Len(t, observed.All(), 1)
			assert.Equal(t, "Exporting failed. Rejecting data. Try enabling sending_queue to survive temporary failures.", observed.All()[0].Message)
			require.NoError(t, be.Shutdown(context.Background()))
		})
	}
}

func errExport(context.Context, request.Request) error {
	return errors.New("my error")
}

func noopExport(context.Context, request.Request) error {
	return nil
}

func newFakeQueueBatch() QueueBatchSettings[request.Request] {
	return QueueBatchSettings[request.Request]{
		Encoding:   fakeEncoding{},
		ItemsSizer: request.NewItemsSizer(),
		BytesSizer: requesttest.NewBytesSizer(),
	}
}

type fakeEncoding struct{}

func (f fakeEncoding) Marshal(context.Context, request.Request) ([]byte, error) {
	return []byte("mockRequest"), nil
}

func (f fakeEncoding) Unmarshal([]byte) (context.Context, request.Request, error) {
	return context.Background(), &requesttest.FakeRequest{}, nil
}
