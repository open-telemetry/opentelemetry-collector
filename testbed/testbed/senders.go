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

package testbed

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
	"go.opentelemetry.io/collector/internal/data"
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
type TraceDataSenderOld interface {
	DataSender
	SendSpans(traces consumerdata.TraceData) error
}

// MetricDataSender defines the interface that allows sending metric data. It adds ability
// to send a batch of Metrics to the DataSender interface.
type MetricDataSenderOld interface {
	DataSender
	SendMetrics(metrics consumerdata.MetricsData) error
}

// DataSenderOverTraceExporter partially implements TraceDataSender via a TraceExporter.
type DataSenderOverTraceExporterOld struct {
	exporter component.TraceExporterOld
	Host     string
	Port     int
}

// NewDataSenderOverExporter creates a new sender that will send
// to the specified port after Start is called.
func NewDataSenderOverExporterOld(host string, port int) *DataSenderOverTraceExporterOld {
	return &DataSenderOverTraceExporterOld{
		Host: host,
		Port: port,
	}
}

func (ds *DataSenderOverTraceExporterOld) SendSpans(traces consumerdata.TraceData) error {
	return ds.exporter.ConsumeTraceData(context.Background(), traces)
}

func (ds *DataSenderOverTraceExporterOld) Flush() {
	// TraceExporter interface does not support Flush, so nothing to do.
}

func (ds *DataSenderOverTraceExporterOld) GetCollectorPort() int {
	return ds.Port
}

// TraceDataSender defines the interface that allows sending trace data. It adds ability
// to send a batch of Spans to the DataSender interface.
type TraceDataSender interface {
	DataSender
	SendSpans(traces pdata.Traces) error
}

// MetricDataSender defines the interface that allows sending metric data. It adds ability
// to send a batch of Metrics to the DataSender interface.
type MetricDataSender interface {
	DataSender
	SendMetrics(metrics data.MetricData) error
}

// DataSenderOverTraceExporter partially implements TraceDataSender via a TraceExporter.
type DataSenderOverTraceExporter struct {
	exporter component.TraceExporter
	Host     string
	Port     int
}

// NewDataSenderOverExporter creates a new sender that will send
// to the specified port after Start is called.
func NewDataSenderOverExporter(host string, port int) *DataSenderOverTraceExporter {
	return &DataSenderOverTraceExporter{
		Host: host,
		Port: port,
	}
}

func (ds *DataSenderOverTraceExporter) SendSpans(traces pdata.Traces) error {
	return ds.exporter.ConsumeTraces(context.Background(), traces)
}

func (ds *DataSenderOverTraceExporter) Flush() {
	// TraceExporter interface does not support Flush, so nothing to do.
}

func (ds *DataSenderOverTraceExporter) GetCollectorPort() int {
	return ds.Port
}

// JaegerGRPCDataSender implements TraceDataSender for Jaeger thrift_http protocol.
type JaegerGRPCDataSender struct {
	DataSenderOverTraceExporter
}

// Ensure JaegerGRPCDataSender implements TraceDataSender.
var _ TraceDataSender = (*JaegerGRPCDataSender)(nil)

// NewJaegerGRPCDataSender creates a new Jaeger protocol sender that will send
// to the specified port after Start is called.
func NewJaegerGRPCDataSender(host string, port int) *JaegerGRPCDataSender {
	return &JaegerGRPCDataSender{DataSenderOverTraceExporter{
		Host: host,
		Port: port,
	}}
}

func (je *JaegerGRPCDataSender) Start() error {
	cfg := &jaegerexporter.Config{
		// Use standard endpoint for Jaeger.
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: fmt.Sprintf("%s:%d", je.Host, je.Port),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	var err error
	factory := jaegerexporter.Factory{}
	params := component.ExporterCreateParams{Logger: zap.L()}
	exporter, err := factory.CreateTraceExporter(context.Background(), params, cfg)

	if err != nil {
		return err
	}

	je.exporter = exporter
	return err
}

func (je *JaegerGRPCDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	// We only need to enable gRPC protocol because that's what we use in tests.
	// Due to bug in Jaeger receiver (https://go.opentelemetry.io/collector/issues/445)
	// which makes it impossible to disable protocols that we don't need to receive on we
	// have to use fake ports for all endpoints except gRPC, otherwise it is
	// impossible to start the Collector because the standard ports for those protocols
	// are already listened by mock Jaeger backend that is part of the tests.
	// As soon as the bug is fixed remove the endpoints and use "disabled: true" setting
	// instead.
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "%s:%d"
      thrift_tchannel:
        endpoint: "localhost:8372"
      thrift_compact:
        endpoint: "localhost:8373"
      thrift_binary:
        endpoint: "localhost:8374"
      thrift_http:
        endpoint: "localhost:8375"`, je.Host, je.Port)
}

func (je *JaegerGRPCDataSender) ProtocolName() string {
	return "jaeger"
}

// OCTraceDataSender implements TraceDataSender for OpenCensus trace protocol.
type OCTraceDataSender struct {
	DataSenderOverTraceExporterOld
}

// Ensure OCTraceDataSender implements TraceDataSender.
var _ TraceDataSenderOld = (*OCTraceDataSender)(nil)

// NewOCTraceDataSender creates a new OCTraceDataSender that will send
// to the specified port after Start is called.
func NewOCTraceDataSender(host string, port int) *OCTraceDataSender {
	return &OCTraceDataSender{DataSenderOverTraceExporterOld{
		Host: host,
		Port: port,
	}}
}

func (ote *OCTraceDataSender) Start() error {
	cfg := &opencensusexporter.Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: fmt.Sprintf("%s:%d", ote.Host, ote.Port),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
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
    endpoint: "%s:%d"`, ote.Host, ote.Port)
}

func (ote *OCTraceDataSender) ProtocolName() string {
	return "opencensus"
}

// OCMetricsDataSender implements MetricDataSender for OpenCensus metrics protocol.
type OCMetricsDataSender struct {
	exporter component.MetricsExporterOld
	host     string
	port     int
}

// Ensure OCMetricsDataSender implements MetricDataSender.
var _ MetricDataSenderOld = (*OCMetricsDataSender)(nil)

// NewOCMetricDataSender creates a new OpenCensus metric protocol sender that will send
// to the specified port after Start is called.
func NewOCMetricDataSender(host string, port int) *OCMetricsDataSender {
	return &OCMetricsDataSender{
		host: host,
		port: port,
	}
}

func (ome *OCMetricsDataSender) Start() error {
	cfg := &opencensusexporter.Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: fmt.Sprintf("%s:%d", ome.host, ome.port),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
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
    endpoint: "%s:%d"`, ome.host, ome.port)
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
func NewOTLPTraceDataSender(host string, port int) *OTLPTraceDataSender {
	return &OTLPTraceDataSender{DataSenderOverTraceExporter{
		Host: host,
		Port: port,
	}}
}

func (ote *OTLPTraceDataSender) Start() error {
	cfg := &otlpexporter.Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: fmt.Sprintf("%s:%d", ote.Host, ote.Port),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	factory := otlpexporter.Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.L()}
	exporter, err := factory.CreateTraceExporter(context.Background(), creationParams, cfg)

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
    endpoint: "%s:%d"`, ote.Host, ote.Port)
}

func (ote *OTLPTraceDataSender) ProtocolName() string {
	return "otlp"
}

// OTLPMetricsDataSender implements MetricDataSender for OpenCensus metrics protocol.
type OTLPMetricsDataSender struct {
	exporter component.MetricsExporter
	host     string
	port     int
}

// Ensure OTLPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPMetricsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OpenCensus metric protocol sender that will send
// to the specified port after Start is called.
func NewOTLPMetricDataSender(host string, port int) *OTLPMetricsDataSender {
	return &OTLPMetricsDataSender{
		host: host,
		port: port,
	}
}

func (ome *OTLPMetricsDataSender) Start() error {
	cfg := &otlpexporter.Config{
		GRPCClientSettings: configgrpc.GRPCClientSettings{
			Endpoint: fmt.Sprintf("%s:%d", ome.host, ome.port),
			TLSSetting: configtls.TLSClientSetting{
				Insecure: true,
			},
		},
	}

	factory := otlpexporter.Factory{}
	creationParams := component.ExporterCreateParams{Logger: zap.L()}
	exporter, err := factory.CreateMetricsExporter(context.Background(), creationParams, cfg)

	if err != nil {
		return err
	}

	ome.exporter = exporter
	return nil
}

func (ome *OTLPMetricsDataSender) SendMetrics(metrics data.MetricData) error {
	return ome.exporter.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(metrics))
}

func (ome *OTLPMetricsDataSender) Flush() {
}

func (ome *OTLPMetricsDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  otlp:
    endpoint: "%s:%d"`, ome.host, ome.port)
}

func (ome *OTLPMetricsDataSender) GetCollectorPort() int {
	return ome.port
}

func (ome *OTLPMetricsDataSender) ProtocolName() string {
	return "otlp"
}

// ZipkinDataSender implements TraceDataSender for Zipkin http protocol.
type ZipkinDataSender struct {
	DataSenderOverTraceExporterOld
}

// Ensure ZipkinDataSender implements TraceDataSender.
var _ TraceDataSenderOld = (*ZipkinDataSender)(nil)

// NewZipkinDataSender creates a new Zipkin protocol sender that will send
// to the specified port after Start is called.
func NewZipkinDataSender(host string, port int) *ZipkinDataSender {
	return &ZipkinDataSender{DataSenderOverTraceExporterOld{
		Host: host,
		Port: port,
	}}
}

func (zs *ZipkinDataSender) Start() error {
	spansURL := fmt.Sprintf("http://localhost:%d/api/v2/spans", zs.Port)

	cfg := &zipkinexporter.Config{
		URL:    spansURL,
		Format: "json",
	}

	factory := zipkinexporter.Factory{}
	exporter, err := factory.CreateTraceExporter(zap.L(), cfg)

	if err != nil {
		return err
	}

	zs.exporter = exporter
	return err
}

func (zs *ZipkinDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  zipkin:
    endpoint: %s:%d`, zs.Host, zs.Port)
}

func (zs *ZipkinDataSender) ProtocolName() string {
	return "zipkin"
}
