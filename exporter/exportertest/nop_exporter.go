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

type nopExporter int

var _ exporter.TraceExporter = (*nopExporter)(nil)
var _ exporter.MetricsExporter = (*nopExporter)(nil)

func (ne *nopExporter) ProcessTraceData(ctx context.Context, td data.TraceData) error {
	return nil
}

func (ne *nopExporter) ProcessMetricsData(ctx context.Context, md data.MetricsData) error {
	return nil
}

const nopExportFormat = "NopExporter"

func (ne *nopExporter) ExportFormat() string {
	return nopExportFormat
}

// NewNopTraceExporter creates an TraceExporter that just drops the received data.
func NewNopTraceExporter() exporter.TraceExporter {
	return new(nopExporter)
}

// NewNopMetricsExporter creates an MetricsExporter that just drops the received data.
func NewNopMetricsExporter() exporter.MetricsExporter {
	return new(nopExporter)
}
