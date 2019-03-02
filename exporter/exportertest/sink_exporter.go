// Copyright 2018, OpenCensus Authors
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

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

// SinkTraceExporter acts as a trace receiver for use in tests.
type SinkTraceExporter struct {
	mu     sync.Mutex
	traces []data.TraceData
}

var _ exporter.TraceExporter = (*SinkTraceExporter)(nil)

// ProcessTraceData stores traces for tests.
func (ste *SinkTraceExporter) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)

	return nil
}

const sinkExportFormat = "SinkExporter"

// ExportFormat retruns the name of this TraceExporter
func (ste *SinkTraceExporter) ExportFormat() string {
	return sinkExportFormat
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporter) AllTraces() []data.TraceData {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	return ste.traces[:]
}

// SinkMetricsExporter acts as a metrics receiver for use in tests.
type SinkMetricsExporter struct {
	mu      sync.Mutex
	metrics []data.MetricsData
}

var _ exporter.MetricsExporter = (*SinkMetricsExporter)(nil)

// ProcessMetricsData stores traces for tests.
func (sme *SinkMetricsExporter) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)

	return nil
}

// ExportFormat retruns the name of this TraceExporter
func (sme *SinkMetricsExporter) ExportFormat() string {
	return sinkExportFormat
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporter) AllMetrics() []data.MetricsData {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	return sme.metrics[:]
}
