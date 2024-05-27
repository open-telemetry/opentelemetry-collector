// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ctracetest // import "go.opentelemetry.io/collector/consumer/ctrace/ctracetest"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer/ctrace"
	"go.opentelemetry.io/collector/consumer/internal/consumertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
) // TracesSink is a ctrace.Traces that acts like a sink that
// stores all traces and allows querying them for testing.
type TracesSink struct {
	consumertest.NonMutatingConsumer
	mu        sync.Mutex
	traces    []ptrace.Traces
	spanCount int
}

var _ ctrace.Traces = (*TracesSink)(nil)

// ConsumeTraces stores traces to this sink.
func (ste *TracesSink) ConsumeTraces(_ context.Context, td ptrace.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)
	ste.spanCount += td.SpanCount()

	return nil
}

// AllTraces returns the traces stored by this sink since last Reset.
func (ste *TracesSink) AllTraces() []ptrace.Traces {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyTraces := make([]ptrace.Traces, len(ste.traces))
	copy(copyTraces, ste.traces)
	return copyTraces
}

// SpanCount returns the number of spans sent to this sink.
func (ste *TracesSink) SpanCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.spanCount
}

// Reset deletes any stored data.
func (ste *TracesSink) Reset() {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = nil
	ste.spanCount = 0
}
