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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

// SinkTraceExporterOld acts as a trace receiver for use in tests.
type SinkTraceExporterOld struct {
	mu     sync.Mutex
	traces []consumerdata.TraceData
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (ste *SinkTraceExporterOld) Start(host component.Host) error {
	return nil
}

// ConsumeTraceData stores traces for tests.
func (ste *SinkTraceExporterOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporterOld) AllTraces() []consumerdata.TraceData {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	return ste.traces
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ste *SinkTraceExporterOld) Shutdown() error {
	return nil
}

// SinkTraceExporter acts as a trace receiver for use in tests.
type SinkTraceExporter struct {
	consumeTraceError error // to be returned by ConsumeTraceData, if set
	mu                sync.Mutex
	traces            []data.TraceData
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (ste *SinkTraceExporter) Start(host component.Host) error {
	return nil
}

// ConsumeTraceData stores traces for tests.
func (ste *SinkTraceExporter) ConsumeTrace(ctx context.Context, td data.TraceData) error {
	if ste.consumeTraceError != nil {
		return ste.consumeTraceError
	}

	ste.mu.Lock()
	defer ste.mu.Unlock()

	ste.traces = append(ste.traces, td)

	return nil
}

// AllTraces returns the traces sent to the test sink.
func (ste *SinkTraceExporter) AllTraces() []data.TraceData {
	ste.mu.Lock()
	defer ste.mu.Unlock()

	return ste.traces
}

// SetConsumeTraceError sets an error that will be returned by ConsumeTrace
func (ste *SinkTraceExporter) SetConsumeTraceError(err error) {
	ste.consumeTraceError = err
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ste *SinkTraceExporter) Shutdown() error {
	return nil
}

// SinkMetricsExporterOld acts as a metrics receiver for use in tests.
type SinkMetricsExporterOld struct {
	mu      sync.Mutex
	metrics []consumerdata.MetricsData
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (sme *SinkMetricsExporterOld) Start(host component.Host) error {
	return nil
}

// ConsumeMetricsData stores traces for tests.
func (sme *SinkMetricsExporterOld) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporterOld) AllMetrics() []consumerdata.MetricsData {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	return sme.metrics
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sme *SinkMetricsExporterOld) Shutdown() error {
	return nil
}

// SinkMetricsExporter acts as a metrics receiver for use in tests.
type SinkMetricsExporter struct {
	consumeMetricsError error // to be returned by ConsumeTraceData, if set
	mu                  sync.Mutex
	metrics             []data.MetricData
}

// SetConsumeMetricsError sets an error that will be returned by ConsumeMetrics
func (sme *SinkMetricsExporter) SetConsumeMetricsError(err error) {
	sme.consumeMetricsError = err
}

// Start tells the exporter to start. The exporter may prepare for exporting
// by connecting to the endpoint. Host parameter can be used for communicating
// with the host after Start() has already returned.
func (sme *SinkMetricsExporter) Start(host component.Host) error {
	return nil
}

// ConsumeMetricsData stores traces for tests.
func (sme *SinkMetricsExporter) ConsumeMetrics(ctx context.Context, md data.MetricData) error {
	if sme.consumeMetricsError != nil {
		return sme.consumeMetricsError
	}

	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)

	return nil
}

// AllMetrics returns the metrics sent to the test sink.
func (sme *SinkMetricsExporter) AllMetrics() []data.MetricData {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	return sme.metrics
}

// Shutdown stops the exporter and is invoked during shutdown.
func (sme *SinkMetricsExporter) Shutdown() error {
	return nil
}
