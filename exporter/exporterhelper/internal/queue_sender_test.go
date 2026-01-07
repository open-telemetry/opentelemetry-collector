// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension/xextension/extensionmiddleware"
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewQueueSenderFailedRequestDropped(t *testing.T) {
	qSet := queuebatch.AllSettings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	logger, observed := observer.New(zap.ErrorLevel)
	qSet.Telemetry.Logger = zap.New(logger)
	qCfg := NewDefaultQueueConfig()
	be, err := NewQueueSender(
		qSet, qCfg, "", sender.NewSender(func(context.Context, request.Request) error { return errors.New("some error") }))
	require.NoError(t, err)

	require.NoError(t, be.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 2}))
	require.NoError(t, be.Shutdown(context.Background()))
	assert.Len(t, observed.All(), 1)
	assert.Equal(t, "Exporting failed. Dropping data.", observed.All()[0].Message)
}

func TestQueueConfig_Validate(t *testing.T) {
	qCfg := NewDefaultQueueConfig()
	require.NoError(t, qCfg.Validate())

	qCfg.NumConsumers = 0
	require.EqualError(t, qCfg.Validate(), "`num_consumers` must be positive")

	qCfg = NewDefaultQueueConfig()
	qCfg.QueueSize = 0
	require.EqualError(t, qCfg.Validate(), "`queue_size` must be positive")

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	noCfg := configoptional.None[queuebatch.Config]()
	assert.NoError(t, noCfg.Validate())
}

// --- New Tests for Request Middleware ---

// nopComponent satisfies component.Component but does not implement RequestMiddlewareFactory.
// Used for testing error cases where the extension exists but is the wrong type.
type nopComponent struct{}

func (nopComponent) Start(context.Context, component.Host) error { return nil }
func (nopComponent) Shutdown(context.Context) error              { return nil }

type mockRequestMiddleware struct {
	extensionmiddleware.RequestMiddleware
	handled    atomic.Int64
	shutdown   atomic.Int64
	onHandle   func()
	onShutdown func()
}

func (m *mockRequestMiddleware) Handle(ctx context.Context, next func(context.Context) error) error {
	m.handled.Add(1)
	if m.onHandle != nil {
		m.onHandle()
	}
	return next(ctx)
}

func (m *mockRequestMiddleware) Shutdown(ctx context.Context) error {
	m.shutdown.Add(1)
	if m.onShutdown != nil {
		m.onShutdown()
	}
	return nil
}

type mockRequestMiddlewareFactory struct {
	component.Component
	mw  *mockRequestMiddleware
	err error
}

func (m *mockRequestMiddlewareFactory) CreateRequestMiddleware(id component.ID, sig pipeline.Signal) (extensionmiddleware.RequestMiddleware, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.mw, nil
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (m *mockHost) GetExtensions() map[component.ID]component.Component {
	return m.ext
}

func TestQueueSenderWithRequestMiddleware(t *testing.T) {
	mwID := component.MustNewID("request_middleware")
	mockMw := &mockRequestMiddleware{}
	mockFact := &mockRequestMiddlewareFactory{mw: mockMw}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: mockFact,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mwID}
	// Make test deterministic: wait for result ensures Send blocks until middleware runs
	qCfg.WaitForResult = true

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))
	// Because WaitForResult is true, this blocks until the export (and middleware) completes.
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))

	require.NoError(t, qs.Shutdown(context.Background()))

	// Ensure Handle was called at least once
	assert.GreaterOrEqual(t, mockMw.handled.Load(), int64(1))
	assert.Equal(t, int64(1), mockMw.shutdown.Load())
}

func TestQueueSender_MultipleMiddlewares_Ordering(t *testing.T) {
	// Verify that middlewares are chained correctly: m1 wraps m2 wraps base.
	var callOrder []string

	mw1ID := component.MustNewIDWithName("request_middleware", "1")
	mw1 := &mockRequestMiddleware{
		onHandle:   func() { callOrder = append(callOrder, "mw1_handle") },
		onShutdown: func() { callOrder = append(callOrder, "mw1_shutdown") },
	}
	fact1 := &mockRequestMiddlewareFactory{mw: mw1}

	mw2ID := component.MustNewIDWithName("request_middleware", "2")
	mw2 := &mockRequestMiddleware{
		onHandle:   func() { callOrder = append(callOrder, "mw2_handle") },
		onShutdown: func() { callOrder = append(callOrder, "mw2_shutdown") },
	}
	fact2 := &mockRequestMiddlewareFactory{mw: mw2}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mw1ID: fact1,
			mw2ID: fact2,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mw1ID, mw2ID} // Config order
	// Make test deterministic
	qCfg.WaitForResult = true

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		callOrder = append(callOrder, "base")
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))
	require.NoError(t, qs.Shutdown(context.Background()))

	// Expected Execution order: mw1 -> mw2 -> base
	// Expected Shutdown order: mw2 -> mw1 (Reverse/LIFO)
	expected := []string{
		"mw1_handle", "mw2_handle", "base",
		"mw2_shutdown", "mw1_shutdown",
	}
	require.Equal(t, expected, callOrder)
}

func TestQueueSender_Start_Errors_And_Cleanup(t *testing.T) {
	mw1ID := component.MustNewIDWithName("request_middleware", "1")
	mw2ID := component.MustNewIDWithName("request_middleware", "2")

	tests := []struct {
		name        string
		setupHost   func() component.Host
		expectError string
		// Verification that mw1 was cleaned up if mw2 failed
		verifyCleanup func(t *testing.T, host component.Host)
	}{
		{
			name: "extension_not_found",
			setupHost: func() component.Host {
				return &mockHost{ext: map[component.ID]component.Component{}}
			},
			expectError: `request middleware extension "request_middleware/1" not found`,
		},
		{
			name: "extension_not_factory",
			setupHost: func() component.Host {
				return &mockHost{ext: map[component.ID]component.Component{
					mw1ID: &nopComponent{}, // Not a factory
				}}
			},
			expectError: `extension "request_middleware/1" does not implement RequestMiddlewareFactory`,
		},
		{
			name: "factory_creation_fails_cleanup_previous",
			setupHost: func() component.Host {
				mw1 := &mockRequestMiddleware{}
				fact1 := &mockRequestMiddlewareFactory{mw: mw1}

				// Fact2 returns error
				fact2 := &mockRequestMiddlewareFactory{err: errors.New("factory failed")}

				return &mockHost{ext: map[component.ID]component.Component{
					mw1ID: fact1,
					mw2ID: fact2,
				}}
			},
			expectError: `failed to create request middleware for "request_middleware/2": factory failed`,
			verifyCleanup: func(t *testing.T, h component.Host) {
				host := h.(*mockHost)
				fact1 := host.ext[mw1ID].(*mockRequestMiddlewareFactory)
				// Ensure mw1 was shut down even though Start() failed overall
				assert.Equal(t, int64(1), fact1.mw.shutdown.Load(), "First middleware should be shutdown if second fails")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := tt.setupHost()
			qSet := queuebatch.AllSettings[request.Request]{
				ID:        component.MustNewID("otlp"),
				Signal:    pipeline.SignalTraces,
				Telemetry: componenttest.NewNopTelemetrySettings(),
			}
			qCfg := NewDefaultQueueConfig()
			qCfg.RequestMiddlewares = []component.ID{mw1ID, mw2ID}

			nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error { return nil })
			qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
			require.NoError(t, err)

			err = qs.Start(context.Background(), host)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)

			if tt.verifyCleanup != nil {
				tt.verifyCleanup(t, host)
			}
		})
	}
}

func TestNewQueueSender_RequestMiddleware_LogsWarningOnDefaultConsumers(t *testing.T) {
	mwID := component.MustNewID("request_middleware")

	tests := []struct {
		name          string
		inputConfig   queuebatch.Config
		wantConsumers int
		wantLog       bool
	}{
		{
			name: "warn_on_default_consumer_count",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.RequestMiddlewares = []component.ID{mwID}
				cfg.NumConsumers = 10
				return cfg
			}(),
			wantConsumers: 10, // Should NOT change
			wantLog:       true,
		},
		{
			name: "respect_high_consumer_count",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.RequestMiddlewares = []component.ID{mwID}
				cfg.NumConsumers = 500
				return cfg
			}(),
			wantConsumers: 500,
			wantLog:       false,
		},
		{
			name: "ignore_if_middleware_disabled",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.RequestMiddlewares = nil
				cfg.NumConsumers = 10
				return cfg
			}(),
			wantConsumers: 10,
			wantLog:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, observed := observer.New(zap.WarnLevel)
			cfg := tt.inputConfig

			warnIfNumConsumersMayCapMiddleware(&cfg, zap.New(logger))

			assert.Equal(t, tt.wantConsumers, cfg.NumConsumers)
			if tt.wantLog {
				require.Len(t, observed.All(), 1, "Expected a warning log")
				assert.Contains(t, observed.All()[0].Message, "sending_queue.num_consumers is at the default")
			} else {
				assert.Len(t, observed.All(), 0, "Expected no warning logs")
			}
		})
	}
}

func TestQueueSender_RequestMiddleware_Disabled_NoInteraction(t *testing.T) {
	// 1. Setup a host with the extension available, but do NOT use it.
	mwID := component.MustNewID("request_middleware")
	mockMw := &mockRequestMiddleware{}
	mockFact := &mockRequestMiddlewareFactory{mw: mockMw}
	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: mockFact,
		},
	}

	// 2. do NOT configure the RequestMiddlewares in the queue settings.
	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = nil

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	// 3. Start and Send data
	require.NoError(t, qs.Start(context.Background(), host))
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))
	require.NoError(t, qs.Shutdown(context.Background()))

	// 4. Verification: Assert ABSOLUTELY NO interaction with the mock middleware
	assert.Equal(t, int64(0), mockMw.handled.Load(), "Handle should not be called when middleware is disabled")
	assert.Equal(t, int64(0), mockMw.shutdown.Load(), "Shutdown should not be called on the middleware when disabled")
}
