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

package otlpexporter

import (
	"context"
	"fmt"
	"sync"

	otlpmetrics "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/metrics/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/collector/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

type otlpExporter struct {
	exporters chan *exporterImp
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
	defaultNumWorkers int = 8

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

// createOTLPExporter creates an OTLP exporter.
func createOTLPExporter(config configmodels.Exporter) (*otlpExporter, error) {
	oCfg := config.(*Config)

	if oCfg.Endpoint == "" {
		return nil, &exporterError{
			code: errEndpointRequired,
			msg:  "OTLP exporter config requires an Endpoint",
		}
	}

	numWorkers := defaultNumWorkers
	if oCfg.NumWorkers > 0 {
		numWorkers = oCfg.NumWorkers
	}

	exportersChan := make(chan *exporterImp, numWorkers)
	for exporterIndex := 0; exporterIndex < numWorkers; exporterIndex++ {
		// TODO: newExporter blocks for connection. Now that we have ability
		// to report errors asynchronously using Host.ReportFatalError we can move this
		// code to Start() and do it in background to avoid blocking Collector startup
		// as we do now.
		exporter, serr := newExporter(oCfg)
		if serr != nil {
			return nil, fmt.Errorf("cannot configure OTLP exporter: %v", serr)
		}
		exportersChan <- exporter
	}
	oce := &otlpExporter{exporters: exportersChan}
	return oce, nil
}

func (oce *otlpExporter) Shutdown(context.Context) error {
	// Stop all exporters. Will wait until all are stopped.
	wg := &sync.WaitGroup{}
	var errors []error
	var errorsMu sync.Mutex
	visitedCnt := 0
	for currExporter := range oce.exporters {
		wg.Add(1)
		go func(exporter *exporterImp) {
			defer wg.Done()
			err := exporter.stop()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			}
		}(currExporter)
		visitedCnt++
		if visitedCnt == cap(oce.exporters) {
			// Visited and concurrently executed stop() on all exporters.
			break
		}
	}

	// Wait for all stop() calls to finish.
	wg.Wait()
	close(oce.exporters)

	return componenterror.CombineErrors(errors)
}

func (oce *otlpExporter) pushTraceData(ctx context.Context, td pdata.Traces) (int, error) {
	// Get first available exporter.
	exporter, ok := <-oce.exporters
	if !ok {
		err := &exporterError{
			code: errAlreadyStopped,
			msg:  "OpenTelemetry exporter was already stopped.",
		}
		return td.SpanCount(), err
	}

	// Perform the request.
	request := &otlptrace.ExportTraceServiceRequest{
		ResourceSpans: pdata.TracesToOtlp(td),
	}
	err := exporter.exportTrace(ctx, request)

	// Return the exporter to the pool.
	oce.exporters <- exporter
	if err != nil {
		return td.SpanCount(), err
	}
	return 0, nil
}

func (oce *otlpExporter) pushMetricsData(ctx context.Context, md pdata.Metrics) (int, error) {
	imd := pdatautil.MetricsToInternalMetrics(md)
	// Get first available exporter.
	exporter, ok := <-oce.exporters
	if !ok {
		err := &exporterError{
			code: errAlreadyStopped,
			msg:  "OpenTelemetry exporter was already stopped.",
		}
		return imd.MetricCount(), err
	}

	// Perform the request.
	request := &otlpmetrics.ExportMetricsServiceRequest{
		ResourceMetrics: data.MetricDataToOtlp(imd),
	}
	err := exporter.exportMetrics(ctx, request)

	// Return the exporter to the pool.
	oce.exporters <- exporter
	if err != nil {
		return imd.MetricCount(), err
	}
	return 0, nil
}
