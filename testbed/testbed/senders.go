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

package testbed

import (
	"context"
	"fmt"
	"log"
	"net"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/jaegerexporter"
	"go.opentelemetry.io/collector/exporter/opencensusexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/exporter/prometheusexporter"
	"go.opentelemetry.io/collector/exporter/zipkinexporter"
)

// DataSender defines the interface that allows sending data. This is an interface
// that must be implemented by all protocols that want to be used in LoadGenerator.
// Note the terminology: testbed.DataSender is something that sends data to Collector
// and the corresponding entity that receives the data in the Collector is a receiver.
type DataSender interface {
	// Start sender and connect to the configured endpoint. Must be called before
	// sending data.
	Start() error

	// Flush sends any accumulated data.
	Flush()

	// GetEndpoint returns the address to which this sender will send data.
	GetEndpoint() net.Addr

	// GenConfigYAMLStr generates a config string to place in receiver part of collector config
	// so that it can receive data from this sender.
	GenConfigYAMLStr() string

	// ProtocolName returns exporter name to use in collector config pipeline.
	ProtocolName() string
}

// TraceDataSender defines the interface that allows sending trace data. It adds ability
// to send a batch of Spans to the DataSender interface.
type TraceDataSender interface {
	DataSender
	consumer.Traces
}

// MetricDataSender defines the interface that allows sending metric data. It adds ability
// to send a batch of Metrics to the DataSender interface.
type MetricDataSender interface {
	DataSender
	consumer.Metrics
}

// LogDataSender defines the interface that allows sending log data. It adds ability
// to send a batch of Logs to the DataSender interface.
type LogDataSender interface {
	DataSender
	consumer.Logs
}

type DataSenderBase struct {
	Port int
	Host string
}

func (dsb *DataSenderBase) GetEndpoint() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", dsb.Host, dsb.Port))
	return addr
}

func (dsb *DataSenderBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

func (dsb *DataSenderBase) GetFactory(_ component.Kind, _ config.Type) component.Factory {
	return nil
}

func (dsb *DataSenderBase) GetExtensions() map[config.ComponentID]component.Extension {
	return nil
}

func (dsb *DataSenderBase) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return nil
}

func (dsb *DataSenderBase) Flush() {
	// Exporter interface does not support Flush, so nothing to do.
}

// JaegerGRPCDataSender implements TraceDataSender for Jaeger thrift_http exporter.
type JaegerGRPCDataSender struct {
	DataSenderBase
	consumer.Traces
}

// Ensure JaegerGRPCDataSender implements TraceDataSender.
var _ TraceDataSender = (*JaegerGRPCDataSender)(nil)

// NewJaegerGRPCDataSender creates a new Jaeger exporter sender that will send
// to the specified port after Start is called.
func NewJaegerGRPCDataSender(host string, port int) *JaegerGRPCDataSender {
	return &JaegerGRPCDataSender{
		DataSenderBase: DataSenderBase{Port: port, Host: host},
	}
}

func (je *JaegerGRPCDataSender) Start() error {
	factory := jaegerexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*jaegerexporter.Config)
	// Disable retries, we should push data and if error just log it.
	cfg.RetrySettings.Enabled = false
	// Disable sending queue, we should push data from the caller goroutine.
	cfg.QueueSettings.Enabled = false
	cfg.Endpoint = je.GetEndpoint().String()
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}

	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	je.Traces = exp
	return exp.Start(context.Background(), je)
}

func (je *JaegerGRPCDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "%s"`, je.GetEndpoint())
}

func (je *JaegerGRPCDataSender) ProtocolName() string {
	return "jaeger"
}

type ocDataSender struct {
	DataSenderBase
}

func (ods *ocDataSender) fillConfig(cfg *opencensusexporter.Config) *opencensusexporter.Config {
	cfg.Endpoint = ods.GetEndpoint().String()
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}
	return cfg
}

func (ods *ocDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  opencensus:
    endpoint: "%s"`, ods.GetEndpoint())
}

func (ods *ocDataSender) ProtocolName() string {
	return "opencensus"
}

// OCTraceDataSender implements TraceDataSender for OpenCensus trace exporter.
type OCTraceDataSender struct {
	ocDataSender
	consumer.Traces
}

// Ensure OCTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OCTraceDataSender)(nil)

// NewOCTraceDataSender creates a new OCTraceDataSender that will send
// to the specified port after Start is called.
func NewOCTraceDataSender(host string, port int) *OCTraceDataSender {
	return &OCTraceDataSender{
		ocDataSender: ocDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *OCTraceDataSender) Start() error {
	factory := opencensusexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*opencensusexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// OCMetricsDataSender implements MetricDataSender for OpenCensus metrics exporter.
type OCMetricsDataSender struct {
	ocDataSender
	consumer.Metrics
}

// Ensure OCMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OCMetricsDataSender)(nil)

// NewOCMetricDataSender creates a new OpenCensus metric exporter sender that will send
// to the specified port after Start is called.
func NewOCMetricDataSender(host string, port int) *OCMetricsDataSender {
	return &OCMetricsDataSender{
		ocDataSender: ocDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *OCMetricsDataSender) Start() error {
	factory := opencensusexporter.NewFactory()
	cfg := ome.fillConfig(factory.CreateDefaultConfig().(*opencensusexporter.Config))
	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ome.Metrics = exp
	return exp.Start(context.Background(), ome)
}

type otlpHTTPDataSender struct {
	DataSenderBase
}

func (ods *otlpHTTPDataSender) fillConfig(cfg *otlphttpexporter.Config) *otlphttpexporter.Config {
	cfg.Endpoint = fmt.Sprintf("http://%s", ods.GetEndpoint())
	// Disable retries, we should push data and if error just log it.
	cfg.RetrySettings.Enabled = false
	// Disable sending queue, we should push data from the caller goroutine.
	cfg.QueueSettings.Enabled = false
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}
	return cfg
}

func (ods *otlpHTTPDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  otlp:
    protocols:
      http:
        endpoint: "%s"`, ods.GetEndpoint())
}

func (ods *otlpHTTPDataSender) ProtocolName() string {
	return "otlp"
}

// OTLPHTTPTraceDataSender implements TraceDataSender for OTLP/HTTP trace exporter.
type OTLPHTTPTraceDataSender struct {
	otlpHTTPDataSender
	consumer.Traces
}

// Ensure OTLPHTTPTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OTLPHTTPTraceDataSender)(nil)

// NewOTLPHTTPTraceDataSender creates a new TraceDataSender for OTLP/HTTP traces exporter.
func NewOTLPHTTPTraceDataSender(host string, port int) *OTLPHTTPTraceDataSender {
	return &OTLPHTTPTraceDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *OTLPHTTPTraceDataSender) Start() error {
	factory := otlphttpexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*otlphttpexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// OTLPHTTPMetricsDataSender implements MetricDataSender for OTLP/HTTP metrics exporter.
type OTLPHTTPMetricsDataSender struct {
	otlpHTTPDataSender
	consumer.Metrics
}

// Ensure OTLPHTTPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPHTTPMetricsDataSender)(nil)

// NewOTLPHTTPMetricDataSender creates a new OTLP/HTTP metrics exporter sender that will send
// to the specified port after Start is called.
func NewOTLPHTTPMetricDataSender(host string, port int) *OTLPHTTPMetricsDataSender {
	return &OTLPHTTPMetricsDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *OTLPHTTPMetricsDataSender) Start() error {
	factory := otlphttpexporter.NewFactory()
	cfg := ome.fillConfig(factory.CreateDefaultConfig().(*otlphttpexporter.Config))
	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ome.Metrics = exp
	return exp.Start(context.Background(), ome)
}

// OTLPHTTPLogsDataSender implements LogsDataSender for OTLP/HTTP logs exporter.
type OTLPHTTPLogsDataSender struct {
	otlpHTTPDataSender
	consumer.Logs
}

// Ensure OTLPHTTPLogsDataSender implements MetricDataSender.
var _ LogDataSender = (*OTLPHTTPLogsDataSender)(nil)

// NewOTLPHTTPLogsDataSender creates a new OTLP/HTTP logs exporter sender that will send
// to the specified port after Start is called.
func NewOTLPHTTPLogsDataSender(host string, port int) *OTLPHTTPLogsDataSender {
	return &OTLPHTTPLogsDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (olds *OTLPHTTPLogsDataSender) Start() error {
	factory := otlphttpexporter.NewFactory()
	cfg := olds.fillConfig(factory.CreateDefaultConfig().(*otlphttpexporter.Config))
	exp, err := factory.CreateLogsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	olds.Logs = exp
	return exp.Start(context.Background(), olds)
}

type otlpDataSender struct {
	DataSenderBase
}

func (ods *otlpDataSender) fillConfig(cfg *otlpexporter.Config) *otlpexporter.Config {
	cfg.Endpoint = ods.GetEndpoint().String()
	// Disable retries, we should push data and if error just log it.
	cfg.RetrySettings.Enabled = false
	// Disable sending queue, we should push data from the caller goroutine.
	cfg.QueueSettings.Enabled = false
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}
	return cfg
}

func (ods *otlpDataSender) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  otlp:
    protocols:
      grpc:
        endpoint: "%s"`, ods.GetEndpoint())
}

func (ods *otlpDataSender) ProtocolName() string {
	return "otlp"
}

// OTLPTraceDataSender implements TraceDataSender for OTLP traces exporter.
type OTLPTraceDataSender struct {
	otlpDataSender
	consumer.Traces
}

// Ensure OTLPTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OTLPTraceDataSender)(nil)

// NewOTLPTraceDataSender creates a new TraceDataSender for OTLP traces exporter.
func NewOTLPTraceDataSender(host string, port int) *OTLPTraceDataSender {
	return &OTLPTraceDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *OTLPTraceDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// OTLPMetricsDataSender implements MetricDataSender for OTLP metrics exporter.
type OTLPMetricsDataSender struct {
	otlpDataSender
	consumer.Metrics
}

// Ensure OTLPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPMetricsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OTLP metric exporter sender that will send
// to the specified port after Start is called.
func NewOTLPMetricDataSender(host string, port int) *OTLPMetricsDataSender {
	return &OTLPMetricsDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *OTLPMetricsDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := ome.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ome.Metrics = exp
	return exp.Start(context.Background(), ome)
}

// OTLPLogsDataSender implements LogsDataSender for OTLP logs exporter.
type OTLPLogsDataSender struct {
	otlpDataSender
	consumer.Logs
}

// Ensure OTLPLogsDataSender implements LogDataSender.
var _ LogDataSender = (*OTLPLogsDataSender)(nil)

// NewOTLPLogsDataSender creates a new OTLP logs exporter sender that will send
// to the specified port after Start is called.
func NewOTLPLogsDataSender(host string, port int) *OTLPLogsDataSender {
	return &OTLPLogsDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (olds *OTLPLogsDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := olds.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateLogsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	olds.Logs = exp
	return exp.Start(context.Background(), olds)
}

// ZipkinDataSender implements TraceDataSender for Zipkin http exporter.
type ZipkinDataSender struct {
	DataSenderBase
	consumer.Traces
}

// Ensure ZipkinDataSender implements TraceDataSender.
var _ TraceDataSender = (*ZipkinDataSender)(nil)

// NewZipkinDataSender creates a new Zipkin exporter sender that will send
// to the specified port after Start is called.
func NewZipkinDataSender(host string, port int) *ZipkinDataSender {
	return &ZipkinDataSender{
		DataSenderBase: DataSenderBase{
			Port: port,
			Host: host,
		},
	}
}

func (zs *ZipkinDataSender) Start() error {
	factory := zipkinexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*zipkinexporter.Config)
	cfg.Endpoint = fmt.Sprintf("http://%s/api/v2/spans", zs.GetEndpoint())
	// Disable retries, we should push data and if error just log it.
	cfg.RetrySettings.Enabled = false
	// Disable sending queue, we should push data from the caller goroutine.
	cfg.QueueSettings.Enabled = false

	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	zs.Traces = exp
	return exp.Start(context.Background(), zs)
}

func (zs *ZipkinDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  zipkin:
    endpoint: %s`, zs.GetEndpoint())
}

func (zs *ZipkinDataSender) ProtocolName() string {
	return "zipkin"
}

// prometheus

type PrometheusDataSender struct {
	DataSenderBase
	consumer.Metrics
	namespace string
}

var _ MetricDataSender = (*PrometheusDataSender)(nil)

func NewPrometheusDataSender(host string, port int) *PrometheusDataSender {
	return &PrometheusDataSender{
		DataSenderBase: DataSenderBase{
			Port: port,
			Host: host,
		},
	}
}

func (pds *PrometheusDataSender) Start() error {
	factory := prometheusexporter.NewFactory()
	cfg := factory.CreateDefaultConfig().(*prometheusexporter.Config)
	cfg.Endpoint = pds.GetEndpoint().String()
	cfg.Namespace = pds.namespace

	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	pds.Metrics = exp
	return exp.Start(context.Background(), pds)
}

func (pds *PrometheusDataSender) GenConfigYAMLStr() string {
	format := `
  prometheus:
    config:
      scrape_configs:
        - job_name: 'testbed'
          scrape_interval: 100ms
          static_configs:
            - targets: ['%s']
`
	return fmt.Sprintf(format, pds.GetEndpoint())
}

func (pds *PrometheusDataSender) ProtocolName() string {
	return "prometheus"
}

func defaultExporterParams() component.ExporterCreateSettings {
	return component.ExporterCreateSettings{Logger: zap.L()}
}
