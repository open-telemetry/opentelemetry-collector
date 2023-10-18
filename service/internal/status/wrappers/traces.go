// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package wrappers // import "go.opentelemetry.io/collector/service/internal/status/wrappers"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
)

type tracesConsumer struct {
	consumer.Traces
	telemetry *component.TelemetrySettings
}

func (tc *tracesConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	err := tc.Traces.ConsumeTraces(ctx, td)
	_ = tc.telemetry.ReportComponentStatus(
		component.NewEventFromError(err, component.StatusRecoverableError))
	return err
}

func WrapTracesConsumer(tc consumer.Traces, telemetry *component.TelemetrySettings) consumer.Traces {
	return &tracesConsumer{
		Traces:    tc,
		telemetry: telemetry,
	}
}

func WrapTracesExporter(te exporter.Traces, telemetry *component.TelemetrySettings) exporter.Traces {
	return &struct {
		component.Component
		consumer.Traces
	}{
		Component: WrapComponent(te, telemetry),
		Traces:    WrapTracesConsumer(te, telemetry),
	}
}

func WrapTracesProcessor(tp processor.Traces, telemetry *component.TelemetrySettings) processor.Traces {
	return &struct {
		component.Component
		consumer.Traces
	}{
		Component: WrapComponent(tp, telemetry),
		Traces:    WrapTracesConsumer(tp, telemetry),
	}
}

func WrapTracesConnector(tc connector.Traces, telemetry *component.TelemetrySettings) connector.Traces {
	return &struct {
		component.Component
		consumer.Traces
	}{
		Component: WrapComponent(tc, telemetry),
		Traces:    WrapTracesConsumer(tc, telemetry),
	}
}
