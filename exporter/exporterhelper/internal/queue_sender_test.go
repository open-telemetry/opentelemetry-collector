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
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestNewQueueSenderFailedRequestDropped(t *testing.T) {
	qSet := queuebatch.Settings[request.Request]{
		Signal:    pipeline.SignalMetrics,
		ID:        component.NewID(exportertest.NopType),
		Telemetry: componenttest.NewNopTelemetrySettings(),
	}
	logger, observed := observer.New(zap.ErrorLevel)
	qSet.Telemetry.Logger = zap.New(logger)
	be, err := NewQueueSender(
		qSet, NewDefaultQueueConfig(), BatcherConfig{}, "", sender.NewSender(func(context.Context, request.Request) error { return errors.New("some error") }))
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
	qCfg.Enabled = false
	assert.NoError(t, qCfg.Validate())
}

func TestBatcherConfig_Validate(t *testing.T) {
	cfg := NewDefaultBatcherConfig()
	require.NoError(t, cfg.Validate())

	cfg = NewDefaultBatcherConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, cfg.Validate(), "`flush_timeout` must be greater than zero")
}

func TestSizeConfig_Validate(t *testing.T) {
	cfg := BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeBytes,
			MinSize: 100,
			MaxSize: 1000,
		},
	}
	require.EqualError(t, cfg.Validate(), "unsupported sizer type: {\"bytes\"}")

	cfg = BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeItems,
			MinSize: 100,
			MaxSize: -1000,
		},
	}
	require.EqualError(t, cfg.Validate(), "`max_size` must be greater than or equal to zero")

	cfg = BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeItems,
			MinSize: -100,
			MaxSize: 1000,
		},
	}
	require.EqualError(t, cfg.Validate(), "`min_size` must be greater than or equal to zero")

	cfg = BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeItems,
			MinSize: 1000,
			MaxSize: 100,
		},
	}
	require.EqualError(t, cfg.Validate(), "`max_size` must be greater than or equal to mix_size")
}
