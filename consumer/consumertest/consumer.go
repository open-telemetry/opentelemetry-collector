// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/clog"
	"go.opentelemetry.io/collector/consumer/cmetric"
	"go.opentelemetry.io/collector/consumer/ctrace"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Consumer is a convenience interface that implements all consumer interfaces.
// It has a private function on it to forbid external users from implementing it
// and, as a result, to allow us to add extra functions without breaking
// compatibility.
type Consumer interface {
	// Capabilities to implement the base consumer functionality.
	Capabilities() consumer.Capabilities

	// ConsumeTraces to implement the trace.Traces.
	ConsumeTraces(context.Context, ptrace.Traces) error

	// ConsumeMetrics to implement the metric.Metrics.
	ConsumeMetrics(context.Context, pmetric.Metrics) error

	// ConsumeLogs to implement the log.Logs.
	ConsumeLogs(context.Context, plog.Logs) error

	unexported()
}

var _ clog.Logs = (Consumer)(nil)
var _ cmetric.Metrics = (Consumer)(nil)
var _ ctrace.Traces = (Consumer)(nil)

type nonMutatingConsumer struct{}

// Capabilities returns the base consumer capabilities.
func (bc nonMutatingConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type baseConsumer struct {
	nonMutatingConsumer
	ctrace.ConsumeTracesFunc
	cmetric.ConsumeMetricsFunc
	clog.ConsumeLogsFunc
}

func (bc baseConsumer) unexported() {}
