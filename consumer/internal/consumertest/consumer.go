// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/internal/consumertest"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
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

type NonMutatingConsumer struct{}

// Capabilities returns the base consumer capabilities.
func (bc NonMutatingConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
