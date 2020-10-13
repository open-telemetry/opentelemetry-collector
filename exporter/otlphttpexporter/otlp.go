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

package otlphttpexporter

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/logs/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
)

type exporterImp struct {
	// Input configuration.
	config *Config
	w      sender
}

type sender interface {
	exportTrace(ctx context.Context, request *otlptrace.ExportTraceServiceRequest) error
	exportMetrics(ctx context.Context, request *otlpmetrics.ExportMetricsServiceRequest) error
	exportLogs(ctx context.Context, request *otlplogs.ExportLogsServiceRequest) error
	stop() error
}

// Crete new exporter.
func newExporter(cfg configmodels.Exporter) (*exporterImp, error) {
	oCfg := cfg.(*Config)

	if oCfg.Endpoint == "" {
		return nil, errors.New("OTLP exporter config requires an Endpoint")
	}

	e := &exporterImp{}
	e.config = oCfg
	w, err := newHTTPSender(oCfg)
	if err != nil {
		return nil, err
	}
	e.w = w
	return e, nil
}

func (e *exporterImp) shutdown(context.Context) error {
	return e.w.stop()
}

func (e *exporterImp) pushTraceData(ctx context.Context, td pdata.Traces) (int, error) {
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	err := e.w.exportTrace(ctx, request)

	if err != nil {
		return td.SpanCount(), fmt.Errorf("failed to push trace data via OTLP exporter: %w", err)
	}
	return 0, nil
}

func (e *exporterImp) pushMetricsData(ctx context.Context, md pdata.Metrics) (int, error) {
	request := &otlpmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: pdata.MetricsToOtlp(md),
	}
	err := e.w.exportMetrics(ctx, request)

	if err != nil {
		return md.MetricCount(), fmt.Errorf("failed to push metrics data via OTLP exporter: %w", err)
	}
	return 0, nil
}

func (e *exporterImp) pushLogData(ctx context.Context, logs pdata.Logs) (int, error) {
	request := &otlplogs.ExportLogsServiceRequest{
		ResourceLogs: internal.LogsToOtlp(logs.InternalRep()),
	}
	err := e.w.exportLogs(ctx, request)

	if err != nil {
		return logs.LogRecordCount(), fmt.Errorf("failed to push log data via OTLP exporter: %w", err)
	}
	return 0, nil
}

type httpSender struct {
}

func newHTTPSender(config *Config) (sender, error) {
	hs := &httpSender{}
	return hs, nil
}

func (hs *httpSender) stop() error {
	return nil
}

func (hs *httpSender) exportTrace(ctx context.Context, request *otlptrace.ExportTraceServiceRequest) error {
	return nil
}

func (hs *httpSender) exportMetrics(ctx context.Context, request *otlpmetrics.ExportMetricsServiceRequest) error {
	return nil
}

func (hs *httpSender) exportLogs(ctx context.Context, request *otlplogs.ExportLogsServiceRequest) error {
	return nil
}
