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

// jaegerGRPCDataSender implements TraceDataSender for Jaeger thrift_http exporter.
type jaegerGRPCDataSender struct {
	DataSenderBase
	consumer.Traces
}

// Ensure jaegerGRPCDataSender implements TraceDataSender.
var _ TraceDataSender = (*jaegerGRPCDataSender)(nil)

// NewJaegerGRPCDataSender creates a new Jaeger exporter sender that will send
// to the specified port after Start is called.
func NewJaegerGRPCDataSender(host string, port int) TraceDataSender {
	return &jaegerGRPCDataSender{
		DataSenderBase: DataSenderBase{Port: port, Host: host},
	}
}

func (je *jaegerGRPCDataSender) Start() error {
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

func (je *jaegerGRPCDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  jaeger:
    protocols:
      grpc:
        endpoint: "%s"`, je.GetEndpoint())
}

func (je *jaegerGRPCDataSender) ProtocolName() string {
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

// ocTracesDataSender implements TraceDataSender for OpenCensus trace exporter.
type ocTracesDataSender struct {
	ocDataSender
	consumer.Traces
}

// NewOCTraceDataSender creates a new ocTracesDataSender that will send
// to the specified port after Start is called.
func NewOCTraceDataSender(host string, port int) TraceDataSender {
	return &ocTracesDataSender{
		ocDataSender: ocDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *ocTracesDataSender) Start() error {
	factory := opencensusexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*opencensusexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// ocMetricsDataSender implements MetricDataSender for OpenCensus metrics exporter.
type ocMetricsDataSender struct {
	ocDataSender
	consumer.Metrics
}

// NewOCMetricDataSender creates a new OpenCensus metric exporter sender that will send
// to the specified port after Start is called.
func NewOCMetricDataSender(host string, port int) MetricDataSender {
	return &ocMetricsDataSender{
		ocDataSender: ocDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *ocMetricsDataSender) Start() error {
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

// otlpHTTPTraceDataSender implements TraceDataSender for OTLP/HTTP trace exporter.
type otlpHTTPTraceDataSender struct {
	otlpHTTPDataSender
	consumer.Traces
}

// NewOTLPHTTPTraceDataSender creates a new TraceDataSender for OTLP/HTTP traces exporter.
func NewOTLPHTTPTraceDataSender(host string, port int) TraceDataSender {
	return &otlpHTTPTraceDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *otlpHTTPTraceDataSender) Start() error {
	factory := otlphttpexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*otlphttpexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// otlpHTTPMetricsDataSender implements MetricDataSender for OTLP/HTTP metrics exporter.
type otlpHTTPMetricsDataSender struct {
	otlpHTTPDataSender
	consumer.Metrics
}

// NewOTLPHTTPMetricDataSender creates a new OTLP/HTTP metrics exporter sender that will send
// to the specified port after Start is called.
func NewOTLPHTTPMetricDataSender(host string, port int) MetricDataSender {
	return &otlpHTTPMetricsDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *otlpHTTPMetricsDataSender) Start() error {
	factory := otlphttpexporter.NewFactory()
	cfg := ome.fillConfig(factory.CreateDefaultConfig().(*otlphttpexporter.Config))
	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ome.Metrics = exp
	return exp.Start(context.Background(), ome)
}

// otlpHTTPLogsDataSender implements LogsDataSender for OTLP/HTTP logs exporter.
type otlpHTTPLogsDataSender struct {
	otlpHTTPDataSender
	consumer.Logs
}

// NewOTLPHTTPLogsDataSender creates a new OTLP/HTTP logs exporter sender that will send
// to the specified port after Start is called.
func NewOTLPHTTPLogsDataSender(host string, port int) LogDataSender {
	return &otlpHTTPLogsDataSender{
		otlpHTTPDataSender: otlpHTTPDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (olds *otlpHTTPLogsDataSender) Start() error {
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

// otlpTraceDataSender implements TraceDataSender for OTLP traces exporter.
type otlpTraceDataSender struct {
	otlpDataSender
	consumer.Traces
}

// NewOTLPTraceDataSender creates a new TraceDataSender for OTLP traces exporter.
func NewOTLPTraceDataSender(host string, port int) TraceDataSender {
	return &otlpTraceDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ote *otlpTraceDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := ote.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ote.Traces = exp
	return exp.Start(context.Background(), ote)
}

// otlpMetricsDataSender implements MetricDataSender for OTLP metrics exporter.
type otlpMetricsDataSender struct {
	otlpDataSender
	consumer.Metrics
}

// NewOTLPMetricDataSender creates a new OTLP metric exporter sender that will send
// to the specified port after Start is called.
func NewOTLPMetricDataSender(host string, port int) MetricDataSender {
	return &otlpMetricsDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (ome *otlpMetricsDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := ome.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	ome.Metrics = exp
	return exp.Start(context.Background(), ome)
}

// otlpLogsDataSender implements LogsDataSender for OTLP logs exporter.
type otlpLogsDataSender struct {
	otlpDataSender
	consumer.Logs
}

// NewOTLPLogsDataSender creates a new OTLP logs exporter sender that will send
// to the specified port after Start is called.
func NewOTLPLogsDataSender(host string, port int) LogDataSender {
	return &otlpLogsDataSender{
		otlpDataSender: otlpDataSender{
			DataSenderBase: DataSenderBase{
				Port: port,
				Host: host,
			},
		},
	}
}

func (olds *otlpLogsDataSender) Start() error {
	factory := otlpexporter.NewFactory()
	cfg := olds.fillConfig(factory.CreateDefaultConfig().(*otlpexporter.Config))
	exp, err := factory.CreateLogsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	olds.Logs = exp
	return exp.Start(context.Background(), olds)
}

// zipkinDataSender implements TraceDataSender for Zipkin http exporter.
type zipkinDataSender struct {
	DataSenderBase
	consumer.Traces
}

// NewZipkinDataSender creates a new Zipkin exporter sender that will send
// to the specified port after Start is called.
func NewZipkinDataSender(host string, port int) TraceDataSender {
	return &zipkinDataSender{
		DataSenderBase: DataSenderBase{
			Port: port,
			Host: host,
		},
	}
}

func (zs *zipkinDataSender) Start() error {
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

func (zs *zipkinDataSender) GenConfigYAMLStr() string {
	return fmt.Sprintf(`
  zipkin:
    endpoint: %s`, zs.GetEndpoint())
}

func (zs *zipkinDataSender) ProtocolName() string {
	return "zipkin"
}

// prometheus

type prometheusDataSender struct {
	DataSenderBase
	consumer.Metrics
	namespace string
}

// NewPrometheusDataSender creates a new Prometheus sender that will expose data
// on the specified port after Start is called.
func NewPrometheusDataSender(host string, port int) MetricDataSender {
	return &prometheusDataSender{
		DataSenderBase: DataSenderBase{
			Port: port,
			Host: host,
		},
	}
}

func (pds *prometheusDataSender) Start() error {
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

func (pds *prometheusDataSender) GenConfigYAMLStr() string {
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

func (pds *prometheusDataSender) ProtocolName() string {
	return "prometheus"
}

func defaultExporterParams() component.ExporterCreateSettings {
	return component.ExporterCreateSettings{Logger: zap.L()}
}
