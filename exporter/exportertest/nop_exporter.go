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

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
)

// NopExporterOption represents options that can be applied to a NopExporter.
type NopExporterOption func(*nopExporter)

type nopExporter struct {
	name     string
	retError error
}

var _ exporter.TraceExporter = (*nopExporter)(nil)
var _ exporter.MetricsExporter = (*nopExporter)(nil)

func (ne *nopExporter) Start(host component.Host) error {
	return nil
}

func (ne *nopExporter) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	return ne.retError
}

func (ne *nopExporter) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	return ne.retError
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ne *nopExporter) Shutdown() error {
	return nil
}

const (
	nopTraceExporterName   = "nop_trace"
	nopMetricsExporterName = "nop_metrics"
)

// NewNopTraceExporter creates an TraceExporter that just drops the received data.
func NewNopTraceExporter(options ...NopExporterOption) exporter.TraceExporter {
	return newNopTraceExporter(options...)
}

// NewNopMetricsExporter creates an MetricsExporter that just drops the received data.
func NewNopMetricsExporter(options ...NopExporterOption) exporter.MetricsExporter {
	return newNopMetricsExporter(options...)
}

// WithReturnError returns a NopExporterOption that enforces the nop Exporters to return the given error.
func WithReturnError(retError error) NopExporterOption {
	return func(ne *nopExporter) {
		ne.retError = retError
	}
}

func newNopTraceExporter(options ...NopExporterOption) *nopExporter {
	ne := &nopExporter{
		name: nopTraceExporterName,
	}
	for _, opt := range options {
		opt(ne)
	}
	return ne
}

func newNopMetricsExporter(options ...NopExporterOption) *nopExporter {
	ne := &nopExporter{
		name: nopMetricsExporterName,
	}
	for _, opt := range options {
		opt(ne)
	}
	return ne
}
