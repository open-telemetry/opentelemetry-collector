// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
)

func countingObsMetrics(shutdowns *int) ObsMetrics {
	return ObsMetrics{
		ShutdownFunc: func() {
			*shutdowns++
		},
	}
}

func TestBaseExporterLeavesInjectedObsMetricsOnOptionFailure(t *testing.T) {
	shutdowns := 0

	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		withObsMetrics(countingObsMetrics(&shutdowns)),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.Error(t, err)
	require.Equal(t, 0, shutdowns)
}

func TestBaseExporterUsesAndShutsDownInjectedObsMetrics(t *testing.T) {
	shutdowns := 0
	sent := int64(0)
	inFlight := int64(0)
	metrics := countingObsMetrics(&shutdowns)
	metrics.RecordIntFunc = func(_ context.Context, m queuebatchtelemetry.Metric, value int64, _ ...metric.AddOption) {
		switch m {
		case queuebatchtelemetry.MetricSent:
			sent += value
		case queuebatchtelemetry.MetricInFlight:
			inFlight += value
		}
	}

	be, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalMetrics, noopExport,
		withObsMetrics(metrics))
	require.NoError(t, err)
	require.NoError(t, be.Send(context.Background(), &requesttest.FakeRequest{Items: 3}))
	require.Equal(t, int64(3), sent)
	require.Zero(t, inFlight)

	require.NoError(t, be.Shutdown(context.Background()))
	require.NoError(t, be.Shutdown(context.Background()))
	require.Equal(t, 1, shutdowns)
}

func TestInjectedObsMetricsReleasedOnQueueRegistrationFailure(t *testing.T) {
	shutdowns := 0
	metrics := countingObsMetrics(&shutdowns)
	wantErr := errors.New("register callback failed")
	metrics.RegisterIntFunc = func(queuebatchtelemetry.Metric, func() int64) error {
		return wantErr
	}

	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalTraces, noopExport,
		withObsMetrics(metrics),
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, shutdowns)
}
