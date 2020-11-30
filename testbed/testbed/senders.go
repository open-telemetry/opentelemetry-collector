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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
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

	// Send any accumulated data.
	Flush()

	// Return the port to which this sender will send data.
	GetEndpoint() string

	// Generate a config string to place in receiver part of collector config
	// so that it can receive data from this sender.
	GenConfigYAMLStr() string

	// Return exporterType name to use in collector config pipeline.
	ProtocolName() string
}

// TraceDataSender defines the interface that allows sending trace data. It adds ability
// to send a batch of Spans to the DataSender interface.
type TraceDataSender interface {
	DataSender
	consumer.TracesConsumer
}

// MetricDataSender defines the interface that allows sending metric data. It adds ability
// to send a batch of Metrics to the DataSender interface.
type MetricDataSender interface {
	DataSender
	consumer.MetricsConsumer
}

// LogDataSender defines the interface that allows sending log data. It adds ability
// to send a batch of Logs to the DataSender interface.
type LogDataSender interface {
	DataSender
	consumer.LogsConsumer
}

type DataSenderBase struct {
	Port int
	Host string
}

func (dsb *DataSenderBase) GetEndpoint() string {
	return fmt.Sprintf("%s:%d", dsb.Host, dsb.Port)
}

func (dsb *DataSenderBase) ReportFatalError(err error) {
	log.Printf("Fatal error reported: %v", err)
}

// GetFactory of the specified kind. Returns the factory for a component type.
func (dsb *DataSenderBase) GetFactory(_ component.Kind, _ configmodels.Type) component.Factory {
	return nil
}

// Return map of extensions. Only enabled and created extensions will be returned.
func (dsb *DataSenderBase) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return nil
}

func (dsb *DataSenderBase) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	return nil
}

func (dsb *DataSenderBase) Flush() {
	// Exporter interface does not support Flush, so nothing to do.
}

// JaegerGRPCDataSender implements TraceDataSender for Jaeger thrift_http exporterType.
type JaegerGRPCDataSender struct {
	DataSenderBase
	consumer.TracesConsumer
}

// Ensure JaegerGRPCDataSender implements TraceDataSender.
var _ TraceDataSender = (*JaegerGRPCDataSender)(nil)

// NewJaegerGRPCDataSender creates a new Jaeger exporterType sender that will send
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
	cfg.Endpoint = je.GetEndpoint()
	cfg.TLSSetting = configtls.TLSClientSetting{
		Insecure: true,
	}

	exp, err := factory.CreateTracesExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	je.TracesConsumer = exp
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
	cfg.Endpoint = ods.GetEndpoint()
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

// OCTraceDataSender implements TraceDataSender for OpenCensus trace exporterType.
type OCTraceDataSender struct {
	ocDataSender
	consumer.TracesConsumer
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

	ote.TracesConsumer = exp
	return exp.Start(context.Background(), ote)
}

// OCMetricsDataSender implements MetricDataSender for OpenCensus metrics exporterType.
type OCMetricsDataSender struct {
	ocDataSender
	consumer.MetricsConsumer
}

// Ensure OCMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OCMetricsDataSender)(nil)

// NewOCMetricDataSender creates a new OpenCensus metric exporterType sender that will send
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

	ome.MetricsConsumer = exp
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

// OTLPHTTPTraceDataSender implements TraceDataSender for OTLP/HTTP trace exporterType.
type OTLPHTTPTraceDataSender struct {
	otlpHTTPDataSender
	consumer.TracesConsumer
}

// Ensure OTLPHTTPTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OTLPHTTPTraceDataSender)(nil)

// NewOTLPHTTPTraceDataSender creates a new TraceDataSender for OTLP/HTTP traces exporterType.
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

	ote.TracesConsumer = exp
	return exp.Start(context.Background(), ote)
}

// OTLPHTTPMetricsDataSender implements MetricDataSender for OTLP/HTTP metrics exporterType.
type OTLPHTTPMetricsDataSender struct {
	otlpHTTPDataSender
	consumer.MetricsConsumer
}

// Ensure OTLPHTTPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPHTTPMetricsDataSender)(nil)

// NewOTLPHTTPMetricDataSender creates a new OTLP/HTTP metric exporterType sender that will send
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

	ome.MetricsConsumer = exp
	return exp.Start(context.Background(), ome)
}

// OTLPHTTPLogsDataSender implements LogsDataSender for OTLP/HTTP logs exporterType.
type OTLPHTTPLogsDataSender struct {
	otlpHTTPDataSender
	consumer.LogsConsumer
}

// Ensure OTLPHTTPLogsDataSender implements MetricDataSender.
var _ LogDataSender = (*OTLPHTTPLogsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OTLP/HTTP metric exporterType sender that will send
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

	olds.LogsConsumer = exp
	return exp.Start(context.Background(), olds)
}

type otlpDataSender struct {
	DataSenderBase
}

func (ods *otlpDataSender) fillConfig(cfg *otlpexporter.Config) *otlpexporter.Config {
	cfg.Endpoint = ods.GetEndpoint()
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

// OTLPTraceDataSender implements TraceDataSender for OTLP trace exporterType.
type OTLPTraceDataSender struct {
	otlpDataSender
	consumer.TracesConsumer
}

// Ensure OTLPTraceDataSender implements TraceDataSender.
var _ TraceDataSender = (*OTLPTraceDataSender)(nil)

// NewOTLPTraceDataSender creates a new TraceDataSender for OTLP traces exporterType.
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

	ote.TracesConsumer = exp
	return exp.Start(context.Background(), ote)
}

// OTLPMetricsDataSender implements MetricDataSender for OTLP metrics exporterType.
type OTLPMetricsDataSender struct {
	otlpDataSender
	consumer.MetricsConsumer
}

// Ensure OTLPMetricsDataSender implements MetricDataSender.
var _ MetricDataSender = (*OTLPMetricsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OTLP metric exporterType sender that will send
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

	ome.MetricsConsumer = exp
	return exp.Start(context.Background(), ome)
}

// OTLPLogsDataSender implements LogsDataSender for OTLP logs exporterType.
type OTLPLogsDataSender struct {
	otlpDataSender
	consumer.LogsConsumer
}

// Ensure OTLPLogsDataSender implements LogDataSender.
var _ LogDataSender = (*OTLPLogsDataSender)(nil)

// NewOTLPMetricDataSender creates a new OTLP metric exporterType sender that will send
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

	olds.LogsConsumer = exp
	return exp.Start(context.Background(), olds)
}

// ZipkinDataSender implements TraceDataSender for Zipkin http exporterType.
type ZipkinDataSender struct {
	DataSenderBase
	consumer.TracesConsumer
}

// Ensure ZipkinDataSender implements TraceDataSender.
var _ TraceDataSender = (*ZipkinDataSender)(nil)

// NewZipkinDataSender creates a new Zipkin exporterType sender that will send
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

	zs.TracesConsumer = exp
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
	consumer.MetricsConsumer
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
	cfg.Endpoint = pds.GetEndpoint()
	cfg.Namespace = pds.namespace

	exp, err := factory.CreateMetricsExporter(context.Background(), defaultExporterParams(), cfg)
	if err != nil {
		return err
	}

	pds.MetricsConsumer = exp
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

type FluentBitFileLogWriter struct {
	DataSenderBase
	file        *os.File
	parsersFile *os.File
}

// Ensure FluentBitFileLogWriter implements LogDataSender.
var _ LogDataSender = (*FluentBitFileLogWriter)(nil)

// NewFluentBitFileLogWriter creates a new data sender that will write log entries to a
// file, to be tailed by FluentBit and sent to the collector.
func NewFluentBitFileLogWriter(host string, port int) *FluentBitFileLogWriter {
	file, err := ioutil.TempFile("", "perf-logs.json")
	if err != nil {
		panic("failed to create temp file")
	}

	parsersFile, err := ioutil.TempFile("", "parsers.json")
	if err != nil {
		panic("failed to create temp file")
	}

	f := &FluentBitFileLogWriter{
		DataSenderBase: DataSenderBase{
			Port: port,
			Host: host,
		},
		file:        file,
		parsersFile: parsersFile,
	}
	f.setupParsers()
	return f
}

func (f *FluentBitFileLogWriter) Start() error {
	return nil
}

func (f *FluentBitFileLogWriter) setupParsers() {
	_, err := f.parsersFile.Write([]byte(`
[PARSER]
    Name   json
    Format json
    Time_Key time
    Time_Format %d/%m/%Y:%H:%M:%S %z
`))
	if err != nil {
		panic("failed to write parsers")
	}

	f.parsersFile.Close()
}

func (f *FluentBitFileLogWriter) ConsumeLogs(_ context.Context, logs pdata.Logs) error {
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		for j := 0; j < logs.ResourceLogs().At(i).InstrumentationLibraryLogs().Len(); j++ {
			ills := logs.ResourceLogs().At(i).InstrumentationLibraryLogs().At(j)
			for k := 0; k < ills.Logs().Len(); k++ {
				_, err := f.file.Write(append(f.convertLogToJSON(ills.Logs().At(k)), '\n'))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *FluentBitFileLogWriter) convertLogToJSON(lr pdata.LogRecord) []byte {
	rec := map[string]string{
		"time": time.Unix(0, int64(lr.Timestamp())).Format("02/01/2006:15:04:05Z"),
	}
	rec["log"] = lr.Body().StringVal()

	lr.Attributes().ForEach(func(k string, v pdata.AttributeValue) {
		switch v.Type() {
		case pdata.AttributeValueSTRING:
			rec[k] = v.StringVal()
		case pdata.AttributeValueINT:
			rec[k] = strconv.FormatInt(v.IntVal(), 10)
		case pdata.AttributeValueDOUBLE:
			rec[k] = strconv.FormatFloat(v.DoubleVal(), 'f', -1, 64)
		case pdata.AttributeValueBOOL:
			rec[k] = strconv.FormatBool(v.BoolVal())
		default:
			panic("missing case")
		}
	})
	b, err := json.Marshal(rec)
	if err != nil {
		panic("failed to write log: " + err.Error())
	}
	return b
}

func (f *FluentBitFileLogWriter) Flush() {
	_ = f.file.Sync()
}

func (f *FluentBitFileLogWriter) GenConfigYAMLStr() string {
	// Note that this generates a receiver config for agent.
	return fmt.Sprintf(`
  fluentforward:
    endpoint: "%s"`, f.GetEndpoint())
}

func (f *FluentBitFileLogWriter) Extensions() map[string]string {
	return map[string]string{
		"fluentbit": fmt.Sprintf(`
  fluentbit:
    executable_path: fluent-bit
    tcp_endpoint: "%s"
    config: |
      [SERVICE]
        parsers_file %s
      [INPUT]
        Name tail
        parser json
        path %s
`, f.GetEndpoint(), f.parsersFile.Name(), f.file.Name()),
	}
}

func (f *FluentBitFileLogWriter) ProtocolName() string {
	return "fluentforward"
}

func defaultExporterParams() component.ExporterCreateParams {
	return component.ExporterCreateParams{Logger: zap.L()}
}
