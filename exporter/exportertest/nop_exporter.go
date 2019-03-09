// Copyright 2019, OpenCensus Authors
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

	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

// NopExporterOption represents options that can be applied to a NopExporter.
type NopExporterOption func(*nopExporter)

type nopExporter struct{ retError error }

var _ exporter.TraceExporter = (*nopExporter)(nil)
var _ exporter.MetricsExporter = (*nopExporter)(nil)

func (ne *nopExporter) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	return ne.retError
}

func (ne *nopExporter) ConsumeMetricsData(ctx context.Context, md data.MetricsData) error {
	return ne.retError
}

const (
	nopTraceExportFormat   = "nop_trace"
	nopMetricsExportFormat = "nop_metrics"
)

func (ne *nopExporter) TraceExportFormat() string {
	return nopTraceExportFormat
}

func (ne *nopExporter) MetricsExportFormat() string {
	return nopMetricsExportFormat
}

// NewNopTraceExporter creates an TraceExporter that just drops the received data.
func NewNopTraceExporter(options ...NopExporterOption) exporter.TraceExporter {
	return newNopExporter(options...)
}

// NewNopMetricsExporter creates an MetricsExporter that just drops the received data.
func NewNopMetricsExporter(options ...NopExporterOption) exporter.MetricsExporter {
	return newNopExporter(options...)
}

// WithReturnError returns a NopExporterOption that enforces the nop Exporters to return the given error.
func WithReturnError(retError error) NopExporterOption {
	return func(ne *nopExporter) {
		ne.retError = retError
	}
}

func newNopExporter(options ...NopExporterOption) *nopExporter {
	ne := new(nopExporter)
	for _, opt := range options {
		opt(ne)
	}
	return ne
}
