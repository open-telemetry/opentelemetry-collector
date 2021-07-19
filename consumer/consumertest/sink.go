// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumertest

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/model/pdata"
)

// TracesSink is a consumer.Traces that acts like a sink that
// stores all traces and allows querying them for testing.
type TracesSink struct {
	nonMutatingConsumer
	mu        sync.Mutex
	traces    []pdata.Traces
	spanCount int
}

var _ consumer.Traces = (*TracesSink)(nil)

// ConsumeTraces stores traces to this sink.
func (ste *TracesSink) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)
	ste.spanCount += td.SpanCount()

	return nil
}

// AllTraces returns the traces stored by this sink since last Reset.
func (ste *TracesSink) AllTraces() []pdata.Traces {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyTraces := make([]pdata.Traces, len(ste.traces))
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

// MetricsSink is a consumer.Metrics that acts like a sink that
// stores all metrics and allows querying them for testing.
type MetricsSink struct {
	nonMutatingConsumer
	mu             sync.Mutex
	metrics        []pdata.Metrics
	dataPointCount int
}

var _ consumer.Metrics = (*MetricsSink)(nil)

// ConsumeMetrics stores metrics to this sink.
func (sme *MetricsSink) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)
	sme.dataPointCount += md.DataPointCount()

	return nil
}

// AllMetrics returns the metrics stored by this sink since last Reset.
func (sme *MetricsSink) AllMetrics() []pdata.Metrics {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyMetrics := make([]pdata.Metrics, len(sme.metrics))
	copy(copyMetrics, sme.metrics)
	return copyMetrics
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
	sme.dataPointCount = 0
}

// LogsSink is a consumer.Logs that acts like a sink that
// stores all logs and allows querying them for testing.
type LogsSink struct {
	nonMutatingConsumer
	mu             sync.Mutex
	logs           []pdata.Logs
	logRecordCount int
}

var _ consumer.Logs = (*LogsSink)(nil)

// ConsumeLogs stores logs to this sink.
func (sle *LogsSink) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = append(sle.logs, ld)
	sle.logRecordCount += ld.LogRecordCount()

	return nil
}

// AllLogs returns the logs stored by this sink since last Reset.
func (sle *LogsSink) AllLogs() []pdata.Logs {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyLogs := make([]pdata.Logs, len(sle.logs))
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
	sle.logRecordCount = 0
}
