// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
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
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
	"go.opentelemetry.io/collector/pipeline"

	"sync/atomic"
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

type mockRequestMiddleware struct {
	extensioncapabilities.RequestMiddleware
	handled  atomic.Int64
	shutdown atomic.Int64
}

func (m *mockRequestMiddleware) Handle(ctx context.Context, next func(context.Context) error) error {
	m.handled.Add(1)
	return next(ctx)
}

func (m *mockRequestMiddleware) Shutdown(ctx context.Context) error {
	m.shutdown.Add(1)
	return nil
}

type mockRequestMiddlewareFactory struct {
	component.Component
	mw *mockRequestMiddleware
}

func (m *mockRequestMiddlewareFactory) CreateRequestMiddleware(id component.ID, sig pipeline.Signal) (extensioncapabilities.RequestMiddleware, error) {
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
	qCfg.RequestMiddlewareID = &mwID

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))

	// Assertions AFTER Shutdown to avoid flakes with async worker processing.
	require.NoError(t, qs.Shutdown(context.Background()))

	// Ensure Handle was called at least once
	assert.GreaterOrEqual(t, mockMw.handled.Load(), int64(1))
	assert.Equal(t, int64(1), mockMw.shutdown.Load())
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
				cfg.RequestMiddlewareID = &mwID
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
				cfg.RequestMiddlewareID = &mwID
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
				cfg.RequestMiddlewareID = nil
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

	// 2. do NOT configure the RequestMiddlewareID in the queue settings.
	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.RequestMiddlewareID = nil

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
