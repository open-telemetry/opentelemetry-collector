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
	"go.opentelemetry.io/collector/consumer/pdata"
)

// TracesSink is a consumer.TracesConsumer that acts like a sink that
// stores all traces and allows querying them for testing.
type TracesSink struct {
	mu         sync.Mutex
	traces     []pdata.Traces
	spansCount int
}

var _ consumer.TracesConsumer = (*TracesSink)(nil)

// ConsumeTraces stores traces to this sink.
func (ste *TracesSink) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)
	ste.spansCount += td.SpanCount()

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

// SpansCount return the number of spans sent to this sink.
func (ste *TracesSink) SpansCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.spansCount
}

// Reset deletes any stored data.
func (ste *TracesSink) Reset() {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = nil
	ste.spansCount = 0
}

// MetricsSink is a consumer.MetricsConsumer that acts like a sink that
// stores all metrics and allows querying them for testing.
type MetricsSink struct {
	mu           sync.Mutex
	metrics      []pdata.Metrics
	metricsCount int
}

var _ consumer.MetricsConsumer = (*MetricsSink)(nil)

// ConsumeMetrics stores metrics to this sink.
func (sme *MetricsSink) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)
	sme.metricsCount += md.MetricCount()

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

// MetricsCount return the number of metrics stored by this sink since last Reset.
func (sme *MetricsSink) MetricsCount() int {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	return sme.metricsCount
}

// Reset deletes any stored data.
func (sme *MetricsSink) Reset() {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = nil
	sme.metricsCount = 0
}

// LogsSink is a consumer.LogsConsumer that acts like a sink that
// stores all logs and allows querying them for testing.
type LogsSink struct {
	mu              sync.Mutex
	logs            []pdata.Logs
	logRecordsCount int
}

var _ consumer.LogsConsumer = (*LogsSink)(nil)

// ConsumeLogs stores logs to this sink.
func (sle *LogsSink) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = append(sle.logs, ld)
	sle.logRecordsCount += ld.LogRecordCount()

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

// LogRecordsCount return the number of log records stored by this sink since last Reset.
func (sle *LogsSink) LogRecordsCount() int {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	return sle.logRecordsCount
}

// Reset deletes any stored data.
func (sle *LogsSink) Reset() {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = nil
	sle.logRecordsCount = 0
}
