// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerlogs"
	"go.opentelemetry.io/collector/consumer/consumermetrics"
	"go.opentelemetry.io/collector/consumer/consumertraces"
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

	// ConsumeTraces to implement the consumertraces.Traces.
	ConsumeTraces(context.Context, ptrace.Traces) error

	// ConsumeMetrics to implement the consumermetrics.Metrics.
	ConsumeMetrics(context.Context, pmetric.Metrics) error

	// ConsumeLogs to implement the consumerlogs.Logs.
	ConsumeLogs(context.Context, plog.Logs) error

	unexported()
}

var _ consumerlogs.Logs = (Consumer)(nil)
var _ consumermetrics.Metrics = (Consumer)(nil)
var _ consumertraces.Traces = (Consumer)(nil)

type nonMutatingConsumer struct{}

// Capabilities returns the base consumer capabilities.
func (bc nonMutatingConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

type baseConsumer struct {
	nonMutatingConsumer
	consumertraces.ConsumeTracesFunc
	consumermetrics.ConsumeMetricsFunc
	consumerlogs.ConsumeLogsFunc
}

func (bc baseConsumer) unexported() {}
