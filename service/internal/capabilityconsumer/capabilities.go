// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package capabilityconsumer // import "go.opentelemetry.io/collector/service/internal/capabilityconsumer"

import (
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerlogs"
	"go.opentelemetry.io/collector/consumer/consumermetrics"
	"go.opentelemetry.io/collector/consumer/consumertraces"
)

func NewLogs(logs consumerlogs.Logs, cap consumer.Capabilities) consumerlogs.Logs {
	if logs.Capabilities() == cap {
		return logs
	}
	return capLogs{Logs: logs, cap: cap}
}

type capLogs struct {
	consumerlogs.Logs
	cap consumer.Capabilities
}

func (mts capLogs) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewMetrics(metrics consumermetrics.Metrics, cap consumer.Capabilities) consumermetrics.Metrics {
	if metrics.Capabilities() == cap {
		return metrics
	}
	return capMetrics{Metrics: metrics, cap: cap}
}

type capMetrics struct {
	consumermetrics.Metrics
	cap consumer.Capabilities
}

func (mts capMetrics) Capabilities() consumer.Capabilities {
	return mts.cap
}

func NewTraces(traces consumertraces.Traces, cap consumer.Capabilities) consumertraces.Traces {
	if traces.Capabilities() == cap {
		return traces
	}
	return capTraces{Traces: traces, cap: cap}
}

type capTraces struct {
	consumertraces.Traces
	cap consumer.Capabilities
}

func (mts capTraces) Capabilities() consumer.Capabilities {
	return mts.cap
}
