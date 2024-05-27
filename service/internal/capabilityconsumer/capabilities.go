// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package capabilityconsumer // import "go.opentelemetry.io/collector/service/internal/capabilityconsumer"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/cmetric"
	"go.opentelemetry.io/collector/consumer/conslog"
	"go.opentelemetry.io/collector/consumer/ctrace"
)

func NewLogs(logs conslog.Logs, cap consumer.Capabilities) conslog.Logs {
	if logs.Capabilities() == cap {
		return logs
	}
	return capLogs{Logs: logs, cap: cap}
}

type capLogs struct {
	conslog.Logs
	cap consumer.Capabilities
}

func (mts capLogs) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewMetrics(metrics cmetric.Metrics, cap consumer.Capabilities) cmetric.Metrics {
	if metrics.Capabilities() == cap {
		return metrics
	}
	return capMetrics{Metrics: metrics, cap: cap}
}

type capMetrics struct {
	cmetric.Metrics
	cap consumer.Capabilities
}

func (mts capMetrics) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewTraces(traces ctrace.Traces, cap consumer.Capabilities) ctrace.Traces {
	if traces.Capabilities() == cap {
		return traces
	}
	return capTraces{Traces: traces, cap: cap}
}

type capTraces struct {
	ctrace.Traces
	cap consumer.Capabilities
}

func (mts capTraces) Capabilities() consumer.Capabilities {
	return mts.cap
}
