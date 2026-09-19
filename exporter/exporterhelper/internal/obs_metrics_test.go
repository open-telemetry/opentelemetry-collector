// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/requesttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	queuebatchtelemetry "go.opentelemetry.io/collector/internal/telemetry/queuebatch"
	"go.opentelemetry.io/collector/pipeline"
)

type failSecondMeterProvider struct {
	metric.MeterProvider
	calls int
}

func (p *failSecondMeterProvider) Meter(name string, options ...metric.MeterOption) metric.Meter {
	p.calls++
	meter := p.MeterProvider.Meter(name, options...)
	if p.calls == 2 {
		return failInstrumentMeter{Meter: meter}
	}
	return meter
}

type failInstrumentMeter struct {
	metric.Meter
}

func (failInstrumentMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errCreateInstrument
}

var errCreateInstrument = errors.New("create instrument failed")

func countingObsMetrics(shutdowns *int) ObsMetrics {
	return ObsMetrics{
		QueueMetrics: queuebatchtelemetry.QueueMetrics{
			ShutdownFunc: func() {
				*shutdowns++
			},
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
	metrics.SentFunc = func(_ context.Context, items int64) {
		sent += items
	}
	metrics.InFlightFunc = func(_ context.Context, delta int64) {
		inFlight += delta
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
	metrics.RegisterQueueFunc = func(func() int64, func() int64) error {
		return wantErr
	}

	_, err := NewBaseExporter(exportertest.NewNopSettings(exportertest.NopType), pipeline.SignalTraces, noopExport,
		withObsMetrics(metrics),
		WithQueueBatchSettings(newFakeQueueBatch()),
		WithQueue(configoptional.Some(NewDefaultQueueConfig())))
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, shutdowns)
}

func TestNewExporterObsMetricsSendBuilderError(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = &failSecondMeterProvider{MeterProvider: settings.MeterProvider}

	_, err := newExporterObsMetrics(exporter.Settings{
		ID:                component.NewID(exportertest.NopType),
		TelemetrySettings: settings,
	}, pipeline.SignalTraces, nil)
	require.ErrorIs(t, err, errCreateInstrument)
}
