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

package opencensusexporter

import (
	"context"
	"fmt"
	"sync"

	"contrib.go.opencensus.io/exporter/ocagent"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

type ocAgentExporter struct {
	exporters chan *ocagent.Exporter
}

type ocExporterErrorCode int
type ocExporterError struct {
	code ocExporterErrorCode
	msg  string
}

var _ error = (*ocExporterError)(nil)

func (e *ocExporterError) Error() string {
	return e.msg
}

const (
	defaultNumWorkers int = 2

	_ ocExporterErrorCode = iota // skip 0
	// errEndpointRequired indicates that this exporter was not provided with an endpoint in its config.
	errEndpointRequired
	// errUnsupportedCompressionType indicates that this exporter was provided with a compression protocol it does not support.
	errUnsupportedCompressionType
	// errUnableToGetTLSCreds indicates that this exporter could not read the provided TLS credentials.
	errUnableToGetTLSCreds
	// errAlreadyStopped indicates that the exporter was already stopped.
	errAlreadyStopped
)

// NewTraceExporter creates an Open Census trace exporter.
func NewTraceExporter(logger *zap.Logger, config configmodels.Exporter, opts ...ocagent.ExporterOption) (exporter.TraceExporter, error) {
	oce, err := createOCAgentExporter(logger, config, opts...)
	if err != nil {
		return nil, err
	}
	oexp, err := exporterhelper.NewTraceExporter(
		config,
		oce.PushTraceData,
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true),
		exporterhelper.WithShutdown(oce.Shutdown))
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

// createOCAgentExporter takes ocagent exporter options and create an OC exporter
func createOCAgentExporter(logger *zap.Logger, config configmodels.Exporter, opts ...ocagent.ExporterOption) (*ocAgentExporter, error) {
	oCfg := config.(*Config)
	numWorkers := defaultNumWorkers
	if oCfg.NumWorkers > 0 {
		numWorkers = oCfg.NumWorkers
	}

	exportersChan := make(chan *ocagent.Exporter, numWorkers)
	for exporterIndex := 0; exporterIndex < numWorkers; exporterIndex++ {
		// TODO: ocagent.NewExporter blocks for connection. Now that we have ability
		// to report errors asynchronously using Host.ReportFatalError we can move this
		// code to Start() and do it in background to avoid blocking Collector startup
		// as we do now.
		exporter, serr := ocagent.NewExporter(opts...)
		if serr != nil {
			return nil, fmt.Errorf("cannot configure OpenCensus exporter: %v", serr)
		}
		exportersChan <- exporter
	}
	oce := &ocAgentExporter{exporters: exportersChan}
	return oce, nil
}

// NewMetricsExporter creates an Open Census metrics exporter.
func NewMetricsExporter(logger *zap.Logger, config configmodels.Exporter, opts ...ocagent.ExporterOption) (exporter.MetricsExporter, error) {
	oce, err := createOCAgentExporter(logger, config, opts...)
	if err != nil {
		return nil, err
	}
	oexp, err := exporterhelper.NewMetricsExporter(
		config,
		oce.PushMetricsData,
		exporterhelper.WithTracing(true),
		exporterhelper.WithMetrics(true),
		exporterhelper.WithShutdown(oce.Shutdown))
	if err != nil {
		return nil, err
	}

	return oexp, nil
}

func (oce *ocAgentExporter) Shutdown() error {
	wg := &sync.WaitGroup{}
	var errors []error
	var errorsMu sync.Mutex
	visitedCnt := 0
	for currExporter := range oce.exporters {
		wg.Add(1)
		go func(exporter *ocagent.Exporter) {
			defer wg.Done()
			err := exporter.Stop()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
			}
		}(currExporter)
		visitedCnt++
		if visitedCnt == cap(oce.exporters) {
			// Visited and started Stop on all exporters, just wait for the stop to finish.
			break
		}
	}

	wg.Wait()
	close(oce.exporters)

	return oterr.CombineErrors(errors)
}

func (oce *ocAgentExporter) PushTraceData(ctx context.Context, td consumerdata.TraceData) (int, error) {
	// Get first available exporter.
	exporter, ok := <-oce.exporters
	if !ok {
		err := &ocExporterError{
			code: errAlreadyStopped,
			msg:  fmt.Sprintf("OpenCensus exporter was already stopped."),
		}
		return len(td.Spans), err
	}

	err := exporter.ExportTraceServiceRequest(
		&agenttracepb.ExportTraceServiceRequest{
			Spans:    td.Spans,
			Resource: td.Resource,
			Node:     td.Node,
		},
	)
	oce.exporters <- exporter
	if err != nil {
		return len(td.Spans), err
	}
	return 0, nil
}

func (oce *ocAgentExporter) PushMetricsData(ctx context.Context, md consumerdata.MetricsData) (int, error) {
	// Get first available exporter.
	exporter, ok := <-oce.exporters
	if !ok {
		err := &ocExporterError{
			code: errAlreadyStopped,
			msg:  fmt.Sprintf("OpenCensus exporter was already stopped."),
		}
		return exporterhelper.NumTimeSeries(md), err
	}

	req := &agentmetricspb.ExportMetricsServiceRequest{
		Metrics:  md.Metrics,
		Resource: md.Resource,
		Node:     md.Node,
	}
	err := exporter.ExportMetricsServiceRequest(req)
	oce.exporters <- exporter
	if err != nil {
		return exporterhelper.NumTimeSeries(md), err
	}
	return 0, nil
}
