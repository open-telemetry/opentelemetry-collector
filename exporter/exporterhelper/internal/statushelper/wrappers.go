// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statushelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/statushelper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// WrapConsumeTraces wraps a consumer.ConsumeTracesFunc to automatically report status based on the
// return value ConsumeTraces. If it returns without an error, StatusOK will be reported. If it
// with an error, StatusRecoverableError will be reported. If a component wants to report a more
// severe error status (e.g. StatusPermamentError or StatusFatalError), it can report it manually
// while still using this wrapper. The more severe statuses will transition the component state
// ahead of the wrapper, making the automatic status reporting effectively a no-op.
func WrapConsumeTraces(consumeFunc consumer.ConsumeTracesFunc, telemetry component.TelemetrySettings) consumer.ConsumeTracesFunc {
	return func(ctx context.Context, td ptrace.Traces) error {
		if err := consumeFunc.ConsumeTraces(ctx, td); err != nil {
			telemetry.ReportStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		telemetry.ReportStatus(component.NewStatusEvent(component.StatusOK))
		return nil
	}
}

// WrapConsumeMetrics wraps a consumer.ConsumeMetricsFunc to automatically report status based on the
// return value ConsumeMetrics. If it returns without an error, StatusOK will be reported. If it
// with an error, StatusRecoverableError will be reported. If a component wants to report a more
// severe error status (e.g. StatusPermamentError or StatusFatalError), it can report it manually
// while still using this wrapper. The more severe statuses will transition the component state
// ahead of the wrapper, making the automatic status reporting effectively a no-op.
func WrapConsumeMetrics(consumeFunc consumer.ConsumeMetricsFunc, telemetry component.TelemetrySettings) consumer.ConsumeMetricsFunc {
	return func(ctx context.Context, md pmetric.Metrics) error {
		if err := consumeFunc.ConsumeMetrics(ctx, md); err != nil {
			telemetry.ReportStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		telemetry.ReportStatus(component.NewStatusEvent(component.StatusOK))
		return nil
	}
}

// WrapConsumeLogs wraps a consumer.ConsumeLogsFunc to automatically report status based on the
// return value ConsumeLogs. If it returns without an error, StatusOK will be reported. If it
// with an error, StatusRecoverableError will be reported. If a component wants to report a more
// severe error status (e.g. StatusPermamentError or StatusFatalError), it can report it manually
// while still using this wrapper. The more severe statuses will transition the component state
// ahead of the wrapper, making the automatic status reporting effectively a no-op.
func WrapConsumeLogs(consumeFunc consumer.ConsumeLogsFunc, telemetry component.TelemetrySettings) consumer.ConsumeLogsFunc {
	return func(ctx context.Context, ld plog.Logs) error {
		if err := consumeFunc.ConsumeLogs(ctx, ld); err != nil {
			telemetry.ReportStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		telemetry.ReportStatus(component.NewStatusEvent(component.StatusOK))
		return nil
	}
}
