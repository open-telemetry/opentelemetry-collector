// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrappers // import "go.opentelemetry.io/collector/service/internal/status/wrappers"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
)

type logsConsumer struct {
	consumer.Logs
	telemetry *component.TelemetrySettings
}

func (lc *logsConsumer) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	err := lc.Logs.ConsumeLogs(ctx, ld)
	_ = lc.telemetry.ReportComponentStatus(
		component.NewEventFromError(err, component.StatusRecoverableError))
	return err
}

func WrapLogsConsumer(lc consumer.Logs, telemetry *component.TelemetrySettings) consumer.Logs {
	return &logsConsumer{
		Logs:      lc,
		telemetry: telemetry,
	}
}

func WrapLogsExporter(le exporter.Logs, telemetry *component.TelemetrySettings) exporter.Logs {
	return &struct {
		component.Component
		consumer.Logs
	}{
		Component: WrapComponent(le, telemetry),
		Logs:      WrapLogsConsumer(le, telemetry),
	}
}

func WrapLogsProcessor(lp processor.Logs, telemetry *component.TelemetrySettings) processor.Logs {
	return &struct {
		component.Component
		consumer.Logs
	}{
		Component: WrapComponent(lp, telemetry),
		Logs:      WrapLogsConsumer(lp, telemetry),
	}
}

func WrapLogsConnector(lc connector.Logs, telemetry *component.TelemetrySettings) connector.Logs {
	return &struct {
		component.Component
		consumer.Logs
	}{
		Component: WrapComponent(lc, telemetry),
		Logs:      WrapLogsConsumer(lc, telemetry),
	}
}
