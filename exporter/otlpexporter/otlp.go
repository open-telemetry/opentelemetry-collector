// Copyright The OpenTelemetry Authors
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

package otlpexporter

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal/data"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/metrics/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/collector/trace/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

type otlpExporter struct {
	exporter *exporterImp
	stopOnce sync.Once
}

type exporterErrorCode int
type exporterError struct {
	code exporterErrorCode
	msg  string
}

var _ error = (*exporterError)(nil)

func (e *exporterError) Error() string {
	return e.msg
}

const (
	_ exporterErrorCode = iota // skip 0
	// errEndpointRequired indicates that this exporter was not provided with an endpoint in its config.
	errEndpointRequired
	// errAlreadyStopped indicates that the exporter was already stopped.
	errAlreadyStopped
)

// NewTraceExporter creates an OTLP trace exporter.
func NewTraceExporter(
	ctx context.Context,
	params component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.TraceExporter, error) {
	oce, err := createOTLPExporter(config)
	if err != nil {
		return nil, err
	}
	oexp, err := exporterhelper.NewTraceExporter(
		config,
		oce.pushTraceData,
		exporterhelper.WithShutdown(oce.Shutdown))
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

// NewMetricsExporter creates an OTLP metrics exporter.
func NewMetricsExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.MetricsExporter, error) {
	oce, err := createOTLPExporter(config)
	if err != nil {
		return nil, err
	}
	oexp, err := exporterhelper.NewMetricsExporter(
		config,
		oce.pushMetricsData,
		exporterhelper.WithShutdown(oce.Shutdown),
	)
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

// NewLogExporter creates an OTLP log exporter.
func NewLogExporter(
	_ context.Context,
	_ component.ExporterCreateParams,
	config configmodels.Exporter,
) (component.LogExporter, error) {
	oce, err := createOTLPExporter(config)
	if err != nil {
		return nil, err
	}
	oexp, err := exporterhelper.NewLogsExporter(
		config,
		oce.pushLogData,
		exporterhelper.WithShutdown(oce.Shutdown),
	)
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

// createOTLPExporter creates an OTLP exporter.
func createOTLPExporter(config configmodels.Exporter) (*otlpExporter, error) {
	oCfg := config.(*Config)

	if oCfg.Endpoint == "" {
		return nil, &exporterError{
			code: errEndpointRequired,
			msg:  "OTLP exporter config requires an Endpoint",
		}
	}

	exporter, serr := newExporter(oCfg)
	if serr != nil {
		return nil, fmt.Errorf("cannot configure OTLP exporter: %v", serr)
	}

	oce := &otlpExporter{exporter: exporter}
	return oce, nil
}

func (oce *otlpExporter) Shutdown(context.Context) error {
	err := error(&exporterError{
		code: errAlreadyStopped,
		msg:  "OpenTelemetry exporter was already stopped.",
	})
	oce.stopOnce.Do(func() {
		err = oce.exporter.stop()
	})
	return err
}

func (oce *otlpExporter) pushTraceData(ctx context.Context, td pdata.Traces) (int, error) {
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	err := oce.exporter.exportTrace(ctx, request)

	if err != nil {
		return td.SpanCount(), err
	}
	return 0, nil
}

func (oce *otlpExporter) pushMetricsData(ctx context.Context, md pdata.Metrics) (int, error) {
	imd := pdatautil.MetricsToInternalMetrics(md)
	request := &otlpmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: data.MetricDataToOtlp(imd),
	}
	err := oce.exporter.exportMetrics(ctx, request)

	if err != nil {
		return imd.MetricCount(), err
	}
	return 0, nil
}

func (oce *otlpExporter) pushLogData(ctx context.Context, logs data.Logs) (int, error) {
	request := &otlplogs.ExportLogServiceRequest{
		ResourceLogs: data.LogsToProto(logs),
	}
	err := oce.exporter.exportLogs(ctx, request)

	if err != nil {
		return logs.LogRecordCount(), err
	}
	return 0, nil
}
