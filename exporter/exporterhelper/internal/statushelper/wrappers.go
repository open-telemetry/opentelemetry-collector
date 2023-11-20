// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package statushelper // import "go.opentelemetry.io/collector/exporterhelper/internal/statushelper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// WrapStart wraps a component.StartFunc and will report StatusOK if Start returns without an error.
// If Start returns an error, the automatic status reporting in graph will report
// StatusPermanentError. Components that wish to report a recoverable error from start should not
// use this wrapper and report status manually (or by other means).
func WrapStart(startFunc component.StartFunc, telemetry component.TelemetrySettings) component.StartFunc {
	return func(ctx context.Context, host component.Host) error {
		if err := startFunc.Start(ctx, host); err != nil {
			// automatic status reporting for errors returned from Start will be handled by graph
			return err
		}
		_ = telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
		return nil
	}
}

// WrapConsumeTraces wraps a consumer.ConsumeTracesFunc to automatically report status based on the
// return value ConsumeTraces. If it returns without an error, StatusOK will be reported. If it
// with an error, StatusRecoverableError will be reported. If a component wants to report a more
// severe error status (e.g. StatusPermamentError or StatusFatalError), it can report it manually
// while still using this wrapper. The more severe statuses will transition the component state
// ahead of the wrapper, making the automatic status reporting effectively a no-op.
func WrapConsumeTraces(consumeFunc consumer.ConsumeTracesFunc, telemetry component.TelemetrySettings) consumer.ConsumeTracesFunc {
	return func(ctx context.Context, td ptrace.Traces) error {
		if err := consumeFunc.ConsumeTraces(ctx, td); err != nil {
			_ = telemetry.ReportComponentStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		_ = telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
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
			_ = telemetry.ReportComponentStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		_ = telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
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
			_ = telemetry.ReportComponentStatus(component.NewRecoverableErrorEvent(err))
			return err
		}
		_ = telemetry.ReportComponentStatus(component.NewStatusEvent(component.StatusOK))
		return nil
	}
}
