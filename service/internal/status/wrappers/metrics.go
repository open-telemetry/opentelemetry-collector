// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrappers // import "go.opentelemetry.io/collector/service/internal/status/wrappers"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/processor"
)

type metricsConsumer struct {
	consumer.Metrics
	telemetry *component.TelemetrySettings
}

func (mc *metricsConsumer) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	err := mc.Metrics.ConsumeMetrics(ctx, md)
	_ = mc.telemetry.ReportComponentStatus(
		component.NewEventFromError(err, component.StatusRecoverableError))
	return err
}

func WrapMetricsConsumer(mc consumer.Metrics, telemetry *component.TelemetrySettings) consumer.Metrics {
	return &metricsConsumer{
		Metrics:   mc,
		telemetry: telemetry,
	}
}

func WrapMetricsExporter(me exporter.Metrics, telemetry *component.TelemetrySettings) exporter.Metrics {
	return &struct {
		component.Component
		consumer.Metrics
	}{
		Component: WrapComponent(me, telemetry),
		Metrics:   WrapMetricsConsumer(me, telemetry),
	}
}

func WrapMetricsProcessor(mp processor.Metrics, telemetry *component.TelemetrySettings) processor.Metrics {
	return &struct {
		component.Component
		consumer.Metrics
	}{
		Component: WrapComponent(mp, telemetry),
		Metrics:   WrapMetricsConsumer(mp, telemetry),
	}
}

func WrapMetricsConnector(mc connector.Metrics, telemetry *component.TelemetrySettings) connector.Metrics {
	return &struct {
		component.Component
		consumer.Metrics
	}{
		Component: WrapComponent(mc, telemetry),
		Metrics:   WrapMetricsConsumer(mc, telemetry),
	}
}
