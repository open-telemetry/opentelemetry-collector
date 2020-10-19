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

type baseErrorConsumer struct {
	mu           sync.Mutex
	consumeError error // to be returned by ConsumeTraces, if set
}

// SetConsumeError sets an error that will be returned by the Consume function.
func (bec *baseErrorConsumer) SetConsumeError(err error) {
	bec.mu.Lock()
	defer bec.mu.Unlock()
	bec.consumeError = err
}

// TracesSink acts as a trace receiver for use in tests.
type TracesSink struct {
	baseErrorConsumer
	traces     []pdata.Traces
	spansCount int
}

var _ consumer.TracesConsumer = (*TracesSink)(nil)

// ConsumeTraceData stores traces for tests.
func (ste *TracesSink) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	if ste.consumeError != nil {
		return ste.consumeError
	}

	ste.traces = append(ste.traces, td)
	ste.spansCount += td.SpanCount()

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (ste *TracesSink) AllTraces() []pdata.Traces {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	copyTraces := make([]pdata.Traces, len(ste.traces))
	copy(copyTraces, ste.traces)
	return copyTraces
}

// SpansCount return the number of spans sent to the test sing.
func (ste *TracesSink) SpansCount() int {
	ste.mu.Lock()
	defer ste.mu.Unlock()
	return ste.spansCount
}

// Reset deletes any existing metrics.
func (ste *TracesSink) Reset() {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = nil
	ste.spansCount = 0
}

// MetricsSink acts as a metrics receiver for use in tests.
type MetricsSink struct {
	baseErrorConsumer
	metrics      []pdata.Metrics
	metricsCount int
}

var _ consumer.MetricsConsumer = (*MetricsSink)(nil)

// ConsumeMetricsData stores traces for tests.
func (sme *MetricsSink) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	if sme.consumeError != nil {
		return sme.consumeError
	}

	sme.metrics = append(sme.metrics, md)
	sme.metricsCount += md.MetricCount()

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *MetricsSink) AllMetrics() []pdata.Metrics {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyMetrics := make([]pdata.Metrics, len(sme.metrics))
	copy(copyMetrics, sme.metrics)
	return copyMetrics
}

// MetricsCount return the number of metrics sent to the test sing.
func (sme *MetricsSink) MetricsCount() int {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	return sme.metricsCount
}

// Reset deletes any existing metrics.
func (sme *MetricsSink) Reset() {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = nil
	sme.metricsCount = 0
}

// LogsSink acts as a metrics receiver for use in tests.
type LogsSink struct {
	baseErrorConsumer
	logs            []pdata.Logs
	logRecordsCount int
}

var _ consumer.LogsConsumer = (*LogsSink)(nil)

// ConsumeLogData stores traces for tests.
func (sle *LogsSink) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	if sle.consumeError != nil {
		return sle.consumeError
	}

	sle.logs = append(sle.logs, ld)
	sle.logRecordsCount += ld.LogRecordCount()

	return nil
}

// AllLog returns the metrics sent to the test sink.
func (sle *LogsSink) AllLogs() []pdata.Logs {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	copyLogs := make([]pdata.Logs, len(sle.logs))
	copy(copyLogs, sle.logs)
	return copyLogs
}

// LogRecordsCount return the number of log records sent to the test sing.
func (sle *LogsSink) LogRecordsCount() int {
	sle.mu.Lock()
	defer sle.mu.Unlock()
	return sle.logRecordsCount
}

// Reset deletes any existing logs.
func (sle *LogsSink) Reset() {
	sle.mu.Lock()
	defer sle.mu.Unlock()

	sle.logs = nil
	sle.logRecordsCount = 0
}
