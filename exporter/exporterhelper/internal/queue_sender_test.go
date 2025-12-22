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

type mockConcurrencyController struct {
	extensioncapabilities.ConcurrencyController
	acquired atomic.Int64
	recorded atomic.Int64
	shutdown atomic.Int64
}

func (m *mockConcurrencyController) Acquire(ctx context.Context) error {
	m.acquired.Add(1)
	return nil
}

func (m *mockConcurrencyController) Record(ctx context.Context, dur time.Duration, err error) {
	m.recorded.Add(1)
}

func (m *mockConcurrencyController) Shutdown(ctx context.Context) error {
	m.shutdown.Add(1)
	return nil
}

type mockConcurrencyFactory struct {
	component.Component
	ctrl *mockConcurrencyController
}

func (m *mockConcurrencyFactory) CreateConcurrencyController(id component.ID, sig pipeline.Signal) (extensioncapabilities.ConcurrencyController, error) {
	return m.ctrl, nil
}

type mockHost struct {
	component.Host
	ext map[component.ID]component.Component
}

func (m *mockHost) GetExtensions() map[component.ID]component.Component {
	return m.ext
}

func TestQueueSenderWithConcurrencyController(t *testing.T) {
	ctrlID := component.MustNewID("adaptive_concurrency")
	mockCtrl := &mockConcurrencyController{}
	mockFact := &mockConcurrencyFactory{ctrl: mockCtrl}

	host := &mockHost{
		ext: map[component.ID]component.Component{
			ctrlID: mockFact,
		},
	}

	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.ConcurrencyControllerID = &ctrlID

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	require.NoError(t, qs.Start(context.Background(), host))
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))

	// Assertions AFTER Shutdown to avoid flakes with async worker processing.
	require.NoError(t, qs.Shutdown(context.Background()))

	assert.Equal(t, int64(1), mockCtrl.acquired.Load())
	assert.Equal(t, int64(1), mockCtrl.recorded.Load())
	assert.Equal(t, int64(1), mockCtrl.shutdown.Load())
}

func TestNewQueueSender_ConcurrencyControllerEnforcesMinConsumers(t *testing.T) {
	ctrlID := component.MustNewID("adaptive_concurrency")

	tests := []struct {
		name          string
		inputConfig   queuebatch.Config
		wantConsumers int
	}{
		{
			name: "upgrade_low_consumer_count",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.ConcurrencyControllerID = &ctrlID
				cfg.NumConsumers = 10
				return cfg
			}(),
			wantConsumers: 200,
		},
		{
			name: "respect_high_consumer_count",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.ConcurrencyControllerID = &ctrlID
				cfg.NumConsumers = 500
				return cfg
			}(),
			wantConsumers: 500,
		},
		{
			name: "ignore_if_controller_disabled",
			inputConfig: func() queuebatch.Config {
				cfg := NewDefaultQueueConfig()
				cfg.ConcurrencyControllerID = nil
				cfg.NumConsumers = 10
				return cfg
			}(),
			wantConsumers: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.inputConfig
			enforceMinConsumers(&cfg, zap.NewNop())
			assert.Equal(t, tt.wantConsumers, cfg.NumConsumers)
		})
	}
}

func TestQueueSender_ConcurrencyController_Disabled_NoInteraction(t *testing.T) {
	// 1. Setup a host with the extension available, but do NOT use it.
	ctrlID := component.MustNewID("adaptive_concurrency")
	mockCtrl := &mockConcurrencyController{}
	mockFact := &mockConcurrencyFactory{ctrl: mockCtrl}
	host := &mockHost{
		ext: map[component.ID]component.Component{
			ctrlID: mockFact,
		},
	}

	// 2. do NOT configure the ConcurrencyControllerID in the queue settings.
	qSet := queuebatch.AllSettings[request.Request]{
		ID:        component.MustNewID("otlp"),
		Signal:    pipeline.SignalTraces,
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	qCfg := NewDefaultQueueConfig()
	qCfg.ConcurrencyControllerID = nil

	nextSender := sender.NewSender(func(ctx context.Context, req request.Request) error {
		return nil
	})

	qs, err := NewQueueSender(qSet, qCfg, "", nextSender)
	require.NoError(t, err)

	// 3. Start and Send data
	require.NoError(t, qs.Start(context.Background(), host))
	require.NoError(t, qs.Send(context.Background(), &requesttest.FakeRequest{Items: 10}))
	require.NoError(t, qs.Shutdown(context.Background()))

	// 4. Verification: Assert ABSOLUTELY NO interaction with the controller
	// If the interface was accidentally active, these would be > 0
	assert.Equal(t, int64(0), mockCtrl.acquired.Load(), "Acquire should not be called when controller is disabled")
	assert.Equal(t, int64(0), mockCtrl.recorded.Load(), "Record should not be called when controller is disabled")
	assert.Equal(t, int64(0), mockCtrl.shutdown.Load(), "Shutdown should not be called on the controller when disabled")

	// Internal verification: Ensure the struct field is actually nil
	assert.Nil(t, qs.(*queueSender).ctrl, "Internal controller field must be nil")
}
