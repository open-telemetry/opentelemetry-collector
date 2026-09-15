// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	errRegisterCallback = errors.New("register callback failed")
	errCreateInstrument = errors.New("create instrument failed")
)

type failInstrumentMeterProvider struct {
	metric.MeterProvider
}

func (p failInstrumentMeterProvider) Meter(name string, options ...metric.MeterOption) metric.Meter {
	return failInstrumentMeter{Meter: p.MeterProvider.Meter(name, options...)}
}

type failInstrumentMeter struct {
	metric.Meter
}

func (failInstrumentMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errCreateInstrument
}

type failSecondRegistrationMeterProvider struct {
	metric.MeterProvider
	registrations int
}

func (p *failSecondRegistrationMeterProvider) Meter(name string, options ...metric.MeterOption) metric.Meter {
	return &failSecondRegistrationMeter{
		Meter:         p.MeterProvider.Meter(name, options...),
		registrations: &p.registrations,
	}
}

type failSecondRegistrationMeter struct {
	metric.Meter
	registrations *int
}

func (m *failSecondRegistrationMeter) RegisterCallback(
	callback metric.Callback,
	instruments ...metric.Observable,
) (metric.Registration, error) {
	registrationCount := *m.registrations + 1
	*m.registrations = registrationCount
	if registrationCount == 2 {
		return nil, errRegisterCallback
	}
	return m.Meter.RegisterCallback(callback, instruments...)
}

func TestObsQueueRegistrationRollback(t *testing.T) {
	tt := componenttest.NewTelemetry()
	t.Cleanup(func() { require.NoError(t, tt.Shutdown(context.Background())) })

	settings := tt.NewTelemetrySettings()
	settings.MeterProvider = &failSecondRegistrationMeterProvider{MeterProvider: settings.MeterProvider}
	_, err := newObsQueue[request.Request](Settings[request.Request]{
		Signal:    pipeline.SignalTraces,
		ID:        exporterID,
		Telemetry: settings,
	}, newFakeQueue[request.Request](nil, 7, 9))
	require.ErrorIs(t, err, errRegisterCallback)

	_, err = tt.GetMetric("otelcol_exporter_queue_size")
	require.Error(t, err, "first callback must be unregistered")
}

func TestNewExporterObsMetricsError(t *testing.T) {
	settings := componenttest.NewNopTelemetrySettings()
	settings.MeterProvider = failInstrumentMeterProvider{MeterProvider: settings.MeterProvider}

	_, err := NewExporterObsMetrics(settings, exporterID, pipeline.SignalTraces)
	require.ErrorIs(t, err, errCreateInstrument)
}
