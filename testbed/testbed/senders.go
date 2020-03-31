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
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configgrpc"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/jaeger/jaegerthrifthttpexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/opencensusexporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/otlpexporter"
)

// DataSender defines the interface that allows sending data. This is an interface
// that must be implemented by all protocols that want to be used in LoadGenerator.
// Note the terminology: testbed.DataSender is something that sends data to Collector
// and the corresponding entity that receives the data in the Collector is a receiver.
type DataSender interface {
	// Start sender and connect to the configured endpoint. Must be called before
	// sending data.
	Start() error

	// Send any accumulated data.
	Flush()

	// Return the port to which this sender will send data.
	GetCollectorPort() int

	// Generate a config string to place in receiver part of collector config
	// so that it can receive data from this sender.
	GenConfigYAMLStr() string

	// Return protocol name to use in collector config pipeline.
	ProtocolName() string
}

// TraceDataSender defines the interface that allows sending trace data. It adds ability
// to send a batch of Spans to the DataSender interface.
type TraceDataSender interface {
	DataSender
	SendSpans(traces consumerdata.TraceData) error
}

// MetricDataSender defines the interface that allows sending metric data. It adds ability
// to send a batch of Metrics to the DataSender interface.
type MetricDataSender interface {
	DataSender
	SendMetrics(metrics consumerdata.MetricsData) error
}

// DataSenderOverTraceExporter partially implements TraceDataSender via a TraceExporter.
type DataSenderOverTraceExporter struct {
	exporter component.TraceExporterOld
	Port     int
}

// NewDataSenderOverExporter creates a new sender that will send
// to the specified port after Start is called.
func NewDataSenderOverExporter(port int) *DataSenderOverTraceExporter {
	return &DataSenderOverTraceExporter{Port: port}
}

func (ds *DataSenderOverTraceExporter) SendSpans(traces consumerdata.TraceData) error {
	return ds.exporter.ConsumeTraceData(context.Background(), traces)
}

func (ds *DataSenderOverTraceExporter) Flush() {
	// TraceExporter interface does not support Flush, so nothing to do.
}

func (ds *DataSenderOverTraceExporter) GetCollectorPort() int {
	return ds.Port
}

// JaegerThriftDataSender implements TraceDataSender for Jaeger thrift_http protocol.
type JaegerThriftDataSender struct {
	DataSenderOverTraceExporter
}

// Ensure JaegerThriftDataSender implements TraceDataSender.
var _ TraceDataSender = (*JaegerThriftDataSender)(nil)

// NewJaegerThriftDataSender creates a new Jaeger protocol sender that will send
// to the specified port after Start is called.
func NewJaegerThriftDataSender(port int) *JaegerThriftDataSender {
	return &JaegerThriftDataSender{DataSenderOverTraceExporter{Port: port}}
}

func (je *JaegerThriftDataSender) Start() error {
	cfg := &jaegerthrifthttpexporter.Config{
		// Use standard URL for Jaeger.
		URL:     fmt.Sprintf("http://localhost:%d/api/traces", je.Port),
		Timeout: 5 * time.Second,
	}

	var err error
	factory := jaegerthrifthttpexporter.Factory{}
	exporter, err := factory.CreateTraceExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	je.exporter = exporter
	return err
}

func (je *JaegerThriftDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	// We only need to enable thrift_http protocol because that's what we use in tests.
	// Due to bug in Jaeger receiver (https://github.com/open-telemetry/opentelemetry-collector/issues/445)
	// which makes it impossible to disable protocols that we don't need to receive on we
	// have to use fake ports for all endpoints except thrift_http, otherwise it is
	// impossible to start the Collector because the standard ports for those protocols
	// are already listened by mock Jaeger backend that is part of the tests.
	// As soon as the bug is fixed remove the endpoints and use "disabled: true" setting
	// instead.
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "localhost:8371"
      thrift_tchannel:
        endpoint: "localhost:8372"
      thrift_compact:
        endpoint: "localhost:8373"
      thrift_binary:
        endpoint: "localhost:8374"
      thrift_http:
        endpoint: "localhost:%d"`, je.Port)
}

func (je *JaegerThriftDataSender) ProtocolName() string {
	return "jaeger"
}

// OCTraceDataSender implements TraceDataSender for OpenCensus trace protocol.
type OCTraceDataSender struct {
	DataSenderOverTraceExporter
}

// Ensure OCTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OCTraceDataSender)(nil)

// NewOCTraceDataSender creates a new OCTraceDataSender that will send
// to the specified port after Start is called.
func NewOCTraceDataSender(port int) *OCTraceDataSender {
	return &OCTraceDataSender{DataSenderOverTraceExporter{Port: port}}
}

func (ote *OCTraceDataSender) Start() error {
	cfg := &opencensusexporter.Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: fmt.Sprintf("localhost:%d", ote.Port),
		},
	}

	factory := opencensusexporter.Factory{}
	exporter, err := factory.CreateTraceExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	ote.exporter = exporter
	return err
}

func (ote *OCTraceDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"`, ote.Port)
}

func (ote *OCTraceDataSender) ProtocolName() string {
	return "opencensus"
}

// OCMetricsDataSender implements MetricDataSender for OpenCensus metrics protocol.
type OCMetricsDataSender struct {
	exporter component.MetricsExporterOld
	port     int
}

// Ensure OCMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OCMetricsDataSender)(nil)

// NewOCMetricDataSender creates a new OpenCensus metric protocol sender that will send
// to the specified port after Start is called.
func NewOCMetricDataSender(port int) *OCMetricsDataSender {
	return &OCMetricsDataSender{port: port}
}

func (ome *OCMetricsDataSender) Start() error {
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

func (ome *OCMetricsDataSender) SendMetrics(metrics consumerdata.MetricsData) error {
	return ome.exporter.ConsumeMetricsData(context.Background(), metrics)
}

func (ome *OCMetricsDataSender) Flush() {
}

func (ome *OCMetricsDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "localhost:%d"`, ome.port)
}

func (ome *OCMetricsDataSender) GetCollectorPort() int {
	return ome.port
}

func (ome *OCMetricsDataSender) ProtocolName() string {
	return "opencensus"
}

// OTLPTraceDataSender implements TraceDataSender for OpenCensus trace protocol.
type OTLPTraceDataSender struct {
	DataSenderOverTraceExporter
}

// Ensure OTLPTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OTLPTraceDataSender)(nil)

// NewOTLPTraceDataSender creates a new OTLPTraceDataSender that will send
// to the specified port after Start is called.
func NewOTLPTraceDataSender(port int) *OTLPTraceDataSender {
	return &OTLPTraceDataSender{DataSenderOverTraceExporter{Port: port}}
}

func (ote *OTLPTraceDataSender) Start() error {
	cfg := &otlpexporter.Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: fmt.Sprintf("localhost:%d", ote.Port),
		},
	}

	factory := otlpexporter.Factory{}
	exporter, err := factory.CreateTraceExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	ote.exporter = exporter
	return err
}

func (ote *OTLPTraceDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  otlp:
    endpoint: "localhost:%d"`, ote.Port)
}

func (ote *OTLPTraceDataSender) ProtocolName() string {
	return "otlp"
}

// OTLPMetricsDataSender implements MetricDataSender for OpenCensus metrics protocol.
type OTLPMetricsDataSender struct {
	exporter component.MetricsExporterOld
	port     int
}

// Ensure OTLPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPMetricsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OpenCensus metric protocol sender that will send
// to the specified port after Start is called.
func NewOTLPMetricDataSender(port int) *OTLPMetricsDataSender {
	return &OTLPMetricsDataSender{port: port}
}

func (ome *OTLPMetricsDataSender) Start() error {
	cfg := &otlpexporter.Config{
		GRPCSettings: configgrpc.GRPCSettings{
			Endpoint: fmt.Sprintf("localhost:%d", ome.port),
		},
	}

	factory := otlpexporter.Factory{}
	exporter, err := factory.CreateMetricsExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	ome.exporter = exporter
	return nil
}

func (ome *OTLPMetricsDataSender) SendMetrics(metrics consumerdata.MetricsData) error {
	return ome.exporter.ConsumeMetricsData(context.Background(), metrics)
}

func (ome *OTLPMetricsDataSender) Flush() {
}

func (ome *OTLPMetricsDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  otlp:
    endpoint: "localhost:%d"`, ome.port)
}

func (ome *OTLPMetricsDataSender) GetCollectorPort() int {
	return ome.port
}

func (ome *OTLPMetricsDataSender) ProtocolName() string {
	return "otlp"
}
