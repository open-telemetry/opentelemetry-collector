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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requestmiddleware"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
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

// nopComponent satisfies component.Component but does not implement RequestMiddleware.
type nopComponent struct{}

func (nopComponent) Start(context.Context, component.Host) error { return nil }
func (nopComponent) Shutdown(context.Context) error              { return nil }

type mockRequestMiddleware struct {
	component.StartFunc
	component.ShutdownFunc
	onWrap func() error
	// Used for assertions
	wrapper *mockWrapperSender
}

// WrapSender creates a new wrapper that intercepts calls
func (m *mockRequestMiddleware) WrapSender(_ requestmiddleware.RequestMiddlewareSettings, next sender.Sender[request.Request]) (sender.Sender[request.Request], error) {
	if m.onWrap != nil {
		if err := m.onWrap(); err != nil {
			return nil, err
		}
	}
	m.wrapper = &mockWrapperSender{next: next}
	return m.wrapper, nil
}

// mockWrapperSender intercepts Send calls to verify execution order
type mockWrapperSender struct {
	component.StartFunc // Implements Start (no-op)
	next                sender.Sender[request.Request]
	handled             atomic.Int64
	shutdown            atomic.Int64
	onHandle            func()
	onShutdown          func()
}

func (m *mockWrapperSender) Send(ctx context.Context, req request.Request) error {
	m.handled.Add(1)
	if m.onHandle != nil {
		m.onHandle()
	}
	return m.next.Send(ctx, req)
}

func (m *mockWrapperSender) Shutdown(_ context.Context) error {
	m.shutdown.Add(1)
	if m.onShutdown != nil {
		m.onShutdown()
	}
	return nil
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

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mwID: mockMw,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewares = []component.ID{mwID}
	// Make test deterministic using WaitForResult.
	qCfg.WaitForResult = true

	nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))
	// Because WaitForResult is true, this blocks until the export (and middleware) completes.
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))

	require.NoError(t, qs.Shutdown(context.Background()))

	// Ensure the wrapper was created and called
	require.NotNil(t, mockMw.wrapper)
	assert.Equal(t, int64(1), mockMw.wrapper.handled.Load())
	assert.Equal(t, int64(1), mockMw.wrapper.shutdown.Load())
}

func TestQueueSender_MultipleMiddlewares_Ordering(t *testing.T) {
	// Verify that middlewares are chained correctly: m1 wraps m2 wraps base.
	var callOrder []string

	mw1ID := component.MustNewIDWithName("request_middleware", "1")
	mw1 := &mockRequestMiddleware{}
	mw2ID := component.MustNewIDWithName("request_middleware", "2")
	mw2 := &mockRequestMiddleware{}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			mw1ID: mw1,
			mw2ID: mw2,
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

	nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error {
		callOrder = append(callOrder, "base")
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))

	// Set hooks after Start, because wrappers are created in Start
	require.NotNil(t, mw1.wrapper)
	mw1.wrapper.onHandle = func() { callOrder = append(callOrder, "mw1_handle") }
	mw1.wrapper.onShutdown = func() { callOrder = append(callOrder, "mw1_shutdown") }

	require.NotNil(t, mw2.wrapper)
	mw2.wrapper.onHandle = func() { callOrder = append(callOrder, "mw2_handle") }
	mw2.wrapper.onShutdown = func() { callOrder = append(callOrder, "mw2_shutdown") }

	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))
	require.NoError(t, qs.Shutdown(context.Background()))

	// Expected Execution order: mw1 -> mw2 -> base
	// Expected Shutdown order: mw1 -> mw2 (LIFO)
	expected := []string{
		"mw1_handle", "mw2_handle", "base",
		"mw1_shutdown", "mw2_shutdown",
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
		// Verification that cleanup happened
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
			name: "extension_does_not_implement_interface",
			setupHost: func() component.Host {
				return &mockHost{ext: map[component.ID]component.Component{
					mw1ID: &nopComponent{},
				}}
			},
			expectError: `extension "request_middleware/1" does not implement RequestMiddleware`,
		},
		{
			name: "wrap_sender_fails_cleanup_previous",
			setupHost: func() component.Host {
				// mw1 (outer) fails on wrap
				mw1 := &mockRequestMiddleware{
					onWrap: func() error { return errors.New("wrap failed") },
				}
				// mw2 (inner, processed first) succeeds
				mw2 := &mockRequestMiddleware{}

				return &mockHost{ext: map[component.ID]component.Component{
					mw1ID: mw1,
					mw2ID: mw2,
				}}
			},
			// Expect error from mw1 (index 0 in loop, but loop is reverse so it runs 2nd)
			expectError: `failed to wrap sender for "request_middleware/1": wrap failed`,
			verifyCleanup: func(t *testing.T, h component.Host) {
				host := h.(*mockHost)
				mw2 := host.ext[mw2ID].(*mockRequestMiddleware)
				// Ensure mw2 wrapper was shut down even though Start() failed overall
				require.NotNil(t, mw2.wrapper)
				assert.Equal(t, int64(1), mw2.wrapper.shutdown.Load(), "Inner middleware wrapper should be shutdown if outer fails")
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

			nextSender := sender.NewSender(func(_ context.Context, _ request.Request) error { return nil })
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
