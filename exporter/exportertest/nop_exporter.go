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

package exportertest

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	nopTraceExporterName   = "nop_trace"
	nopMetricsExporterName = "nop_metrics"
	nopLogsExporterName    = "nop_log"
)

type nopExporter struct {
	name     string
	retError error
}

func (ne *nopExporter) Start(context.Context, component.Host) error {
	return nil
}

func (ne *nopExporter) ConsumeTraces(context.Context, pdata.Traces) error {
	return ne.retError
}

func (ne *nopExporter) ConsumeMetrics(context.Context, pdata.Metrics) error {
	return ne.retError
}

func (ne *nopExporter) ConsumeLogs(context.Context, pdata.Logs) error {
	return ne.retError
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ne *nopExporter) Shutdown(context.Context) error {
	return nil
}

// NewNopTraceExporterOld creates an TraceExporter that just drops the received data.
func NewNopTraceExporter() component.TraceExporter {
	ne := &nopExporter{
		name: nopTraceExporterName,
	}
	return ne
}

// NewNopMetricsExporterOld creates an MetricsExporter that just drops the received data.
func NewNopMetricsExporter() component.MetricsExporter {
	ne := &nopExporter{
		name: nopMetricsExporterName,
	}
	return ne
}

// NewNopLogsExporterOld creates an LogsExporter that just drops the received data.
func NewNopLogsExporter() component.LogsExporter {
	ne := &nopExporter{
		name: nopLogsExporterName,
	}
	return ne
}
