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

package testbed

import (
	"context"
	"fmt"

	"contrib.go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
)

// Exporter defines the interface that allows sending data. This is an interface
// that must be implemented by all protocols that want to be used in LoadGenerator.
// Note the terminology: testbed.Exporter is something that sends data to Collector
// and the corresponding entity that receives the data in the Collector is a receiver.
type Exporter interface {
	// Start exporter and connect to the configured endpoint. Must be called before
	// exporting data.
	Start() error

	// Send any accumulated data.
	Flush()

	// Return the port to which this exporter will send data.
	GetCollectorPort() int

	// Generate a config string to place in receiver part of collector config
	// so that it can receive data from this exporter.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

// TraceExporter defines the interface that allows sending trace data. It adds ability
// to export a batch of Spans to the Exporter interface.
type TraceExporter interface {
	Exporter
	ExportSpans(spans []*trace.SpanData) error
}

// MetricExporter defines the interface that allows sending metric data. It adds ability
// to export a batch of Metrics to the Exporter interface.
type MetricExporter interface {
	Exporter
	ExportMetrics(metrics consumerdata.MetricsData) error
}

// jaegerExporter implements TraceExporter for Jaeger Thrift-HTTP protocol.
type jaegerExporter struct {
	exporter *jaeger.Exporter
	port     int
}

// Create a new Jaeger protocol exporter.
func NewJaegerExporter(port int) *jaegerExporter {
	return &jaegerExporter{port: port}
}

func (je *jaegerExporter) Start() error {
	opts := jaeger.Options{
		// Use standard URL for Jaeger.
		CollectorEndpoint: fmt.Sprintf("http://localhost:%d/api/traces", je.port),
		Process: jaeger.Process{
			ServiceName: "load-generator",
		},
	}

	var err error
	je.exporter, err = jaeger.NewExporter(opts)
	return err
}

func (je *jaegerExporter) ExportSpans(spans []*trace.SpanData) error {
	for _, span := range spans {
		je.exporter.ExportSpan(span)
	}
	return nil
}

func (je *jaegerExporter) Flush() {
	je.exporter.Flush()
}

func (je *jaegerExporter) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	// We only need to enable thrift-http protocol because that's what we use in tests.
	// Due to bug in Jaeger receiver (https://github.com/open-telemetry/opentelemetry-collector/issues/445)
	// which makes it impossible to disable protocols that we don't need to receive on we
	// have to use fake ports for all endpoints except thrift-http, otherwise it is
	// impossible to start the Collector because the standard ports for those protocols
	// are already listened by mock Jaeger backend that is part of the tests.
	// As soon as the bug is fixed remove the endpoints and use "disabled: true" setting
	// instead.
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "localhost:8371"
      thrift-tchannel:
        endpoint: "localhost:8372"
      thrift-compact:
        endpoint: "localhost:8373"
      thrift-binary:
        endpoint: "localhost:8374"
      thrift-http:
        endpoint: "localhost:%d"`, je.port)
}

func (je *jaegerExporter) GetCollectorPort() int {
	return je.port
}

func (je *jaegerExporter) ProtocolName() string {
	return "jaeger"
}

// ocMetricsExporter implements MetricExporter for OpenCensus metrics protocol.
type ocMetricsExporter struct {
	exporter exporter.MetricsExporter
	port     int
}

// Create a new OpenCensus metric protocol exporter.
func NewOcMetricExporter(port int) *ocMetricsExporter {
	return &ocMetricsExporter{port: port}
}

func (ome *ocMetricsExporter) Start() error {
	cfg := &opencensusexporter.Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: fmt.Sprintf("localhost:%d", ome.port),
		},
	}

	factory := opencensusexporter.Factory{}
	exporter, err := factory.CreateMetricsExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	ome.exporter = exporter
	return nil
}

func (ome *ocMetricsExporter) ExportMetrics(metrics consumerdata.MetricsData) error {
	return ome.exporter.ConsumeMetricsData(context.Background(), metrics)
}

func (ome *ocMetricsExporter) Flush() {
}

func (ome *ocMetricsExporter) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"`, ome.port)
}

func (ome *ocMetricsExporter) GetCollectorPort() int {
	return ome.port
}

func (ome *ocMetricsExporter) ProtocolName() string {
	return "opencensus"
}
