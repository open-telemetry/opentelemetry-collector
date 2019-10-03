// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exportertest

import (
	"context"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
)

// SinkTraceExporter acts as a trace receiver for use in tests.
type SinkTraceExporter struct {
	mu     sync.Mutex
	traces []consumerdata.TraceData
}

var _ exporter.TraceExporter = (*SinkTraceExporter)(nil)

// ConsumeTraceData stores traces for tests.
func (ste *SinkTraceExporter) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)

	return nil
}

const (
	sinkTraceExportFormat   = "sink_trace"
	sinkMetricsExportFormat = "sink_metrics"
)

// Name returns the name of this TraceExporter.
func (ste *SinkTraceExporter) Name() string {
	return sinkTraceExportFormat
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporter) AllTraces() []consumerdata.TraceData {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	return ste.traces[:]
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ste *SinkTraceExporter) Shutdown() error {
	return nil
}

// SinkMetricsExporter acts as a metrics receiver for use in tests.
type SinkMetricsExporter struct {
	mu      sync.Mutex
	metrics []consumerdata.MetricsData
}

var _ exporter.MetricsExporter = (*SinkMetricsExporter)(nil)

// ConsumeMetricsData stores traces for tests.
func (sme *SinkMetricsExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)

	return nil
}

// Name returns the name of this MetricsExporter.
func (sme *SinkMetricsExporter) Name() string {
	return sinkMetricsExportFormat
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporter) AllMetrics() []consumerdata.MetricsData {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	return sme.metrics[:]
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sme *SinkMetricsExporter) Shutdown() error {
	return nil
}
