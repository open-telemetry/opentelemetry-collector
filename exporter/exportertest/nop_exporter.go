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

type nopExporter struct{}

func (ne *nopExporter) Start(context.Context, component.Host) error {
	return nil
}

func (ne *nopExporter) ConsumeTraces(context.Context, pdata.Traces) error {
	return nil
}

func (ne *nopExporter) ConsumeMetrics(context.Context, pdata.Metrics) error {
	return nil
}

func (ne *nopExporter) ConsumeLogs(context.Context, pdata.Logs) error {
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (ne *nopExporter) Shutdown(context.Context) error {
	return nil
}

// NewNopTraceExporter creates an TracesExporter that just drops the received data.
// Deprecated: Use consumertest.NewNopTraces.
func NewNopTraceExporter() component.TracesExporter {
	return &nopExporter{}
}

// NewNopMetricsExporter creates an MetricsExporter that just drops the received data.
// Deprecated: Use consumertest.NewNopMetrics.
func NewNopMetricsExporter() component.MetricsExporter {
	return &nopExporter{}
}

// NewNopLogsExporter creates an LogsExporter that just drops the received data.
// Deprecated: Use consumertest.NewNopLogs.
func NewNopLogsExporter() component.LogsExporter {
	return &nopExporter{}
}
