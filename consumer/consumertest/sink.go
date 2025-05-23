// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package consumertest // import "go.opentelemetry.io/collector/consumer/consumertest"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// TracesSink is a consumer.Traces that acts like a sink that
// stores all traces and allows querying them for testing.
type TracesSink struct {
	nonMutatingConsumer
	mu        sync.Mutex
	traces    []ptrace.Traces
	contexts  []context.Context
	spanCount int
}

var _ consumer.Traces = (*TracesSink)(nil)

// ConsumeTraces stores traces to this sink.
func (ste *TracesSink) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)
	ste.contexts = append(ste.contexts, ctx)
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

// Contexts returns the contexts stored by this sink since last Reset.
func (ste *TracesSink) Contexts() []context.Context {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyContexts := make([]context.Context, len(ste.contexts))
	copy(copyContexts, ste.contexts)
	return copyContexts
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
	ste.contexts = nil
	ste.spanCount = 0
}

// MetricsSink is a consumer.Metrics that acts like a sink that
// stores all metrics and allows querying them for testing.
type MetricsSink struct {
	nonMutatingConsumer
	mu             sync.Mutex
	metrics        []pmetric.Metrics
	contexts       []context.Context
	dataPointCount int
}

var _ consumer.Metrics = (*MetricsSink)(nil)

// ConsumeMetrics stores metrics to this sink.
func (sme *MetricsSink) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)
	sme.contexts = append(sme.contexts, ctx)
	sme.dataPointCount += md.DataPointCount()

	return nil
}

// AllMetrics returns the metrics stored by this sink since last Reset.
func (sme *MetricsSink) AllMetrics() []pmetric.Metrics {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyMetrics := make([]pmetric.Metrics, len(sme.metrics))
	copy(copyMetrics, sme.metrics)
	return copyMetrics
}

// Contexts returns the contexts stored by this sink since last Reset.
func (sme *MetricsSink) Contexts() []context.Context {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyContexts := make([]context.Context, len(sme.contexts))
	copy(copyContexts, sme.contexts)
	return copyContexts
}

// DataPointCount returns the number of metrics stored by this sink since last Reset.
func (sme *MetricsSink) DataPointCount() int {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	return sme.dataPointCount
}

// Reset deletes any stored data.
func (sme *MetricsSink) Reset() {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = nil
	sme.contexts = nil
	sme.dataPointCount = 0
}

// LogsSink is a consumer.Logs that acts like a sink that
// stores all logs and allows querying them for testing.
type LogsSink struct {
	nonMutatingConsumer
	mu             sync.Mutex
	logs           []plog.Logs
	contexts       []context.Context
	logRecordCount int
}

var _ consumer.Logs = (*LogsSink)(nil)

// ConsumeLogs stores logs to this sink.
func (sle *LogsSink) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = append(sle.logs, ld)
	sle.logRecordCount += ld.LogRecordCount()
	sle.contexts = append(sle.contexts, ctx)
	return nil
}

// AllLogs returns the logs stored by this sink since last Reset.
func (sle *LogsSink) AllLogs() []plog.Logs {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyLogs := make([]plog.Logs, len(sle.logs))
	copy(copyLogs, sle.logs)
	return copyLogs
}

// LogRecordCount returns the number of log records stored by this sink since last Reset.
func (sle *LogsSink) LogRecordCount() int {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	return sle.logRecordCount
}

// Reset deletes any stored data.
func (sle *LogsSink) Reset() {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = nil
	sle.contexts = nil
	sle.logRecordCount = 0
}

// Contexts returns the contexts stored by this sink since last Reset.
func (sle *LogsSink) Contexts() []context.Context {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyContexts := make([]context.Context, len(sle.contexts))
	copy(copyContexts, sle.contexts)
	return copyContexts
}

// ProfilesSink is a xconsumer.Profiles that acts like a sink that
// stores all profiles and allows querying them for testing.
type ProfilesSink struct {
	nonMutatingConsumer
	mu          sync.Mutex
	profiles    []pprofile.Profiles
	contexts    []context.Context
	sampleCount int
}

var _ xconsumer.Profiles = (*ProfilesSink)(nil)

// ConsumeProfiles stores profiles to this sink.
func (ste *ProfilesSink) ConsumeProfiles(ctx context.Context, td pprofile.Profiles) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.profiles = append(ste.profiles, td)
	ste.contexts = append(ste.contexts, ctx)
	ste.sampleCount += td.SampleCount()

	return nil
}

// AllProfiles returns the profiles stored by this sink since last Reset.
func (ste *ProfilesSink) AllProfiles() []pprofile.Profiles {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyProfiles := make([]pprofile.Profiles, len(ste.profiles))
	copy(copyProfiles, ste.profiles)
	return copyProfiles
}

// SampleCount returns the number of profiles stored by this sink since last Reset.
func (ste *ProfilesSink) SampleCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.sampleCount
}

// Reset deletes any stored data.
func (ste *ProfilesSink) Reset() {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.profiles = nil
	ste.contexts = nil
	ste.sampleCount = 0
}

// Contexts returns the contexts stored by this sink since last Reset.
func (ste *ProfilesSink) Contexts() []context.Context {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyContexts := make([]context.Context, len(ste.contexts))
	copy(copyContexts, ste.contexts)
	return copyContexts
}
