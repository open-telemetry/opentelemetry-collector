// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"

import (
	"strings"

	"go.opentelemetry.io/contrib/otelconf"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
)

type TracesConfigV030 struct {
	// Level configures whether spans are emitted or not, the possible values are:
	//  - "none" indicates that no tracing data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	Level configtelemetry.Level `mapstructure:"level"`
	// Propagators is a list of TextMapPropagators from the supported propagators list. Currently,
	// tracecontext and  b3 are supported. By default, the value is set to empty list and
	// context propagation is disabled.
	Propagators []string `mapstructure:"propagators"`

	config.TracerProvider `mapstructure:",squash"`
}

func (c *TracesConfigV030) Unmarshal(conf *confmap.Conf) error {
	unmarshaled := *c
	if err := conf.Unmarshal(c); err != nil {
		v2TracesConfig := tracesConfigV020{
			Level:       unmarshaled.Level,
			Propagators: unmarshaled.Propagators,
		}
		// try unmarshaling using v0.2.0 struct
		if e := conf.Unmarshal(&v2TracesConfig); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		// TODO: add a warning log to tell users to migrate their config
		return tracesConfigV02ToV03(v2TracesConfig, c)
	}
	// ensure endpoint normalization occurs
	for _, r := range c.Processors {
		if r.Batch != nil {
			if r.Batch.Exporter.OTLP != nil {
				r.Batch.Exporter.OTLP.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLP.Endpoint, r.Batch.Exporter.OTLP.Insecure)
			}
		}
		if r.Simple != nil {
			if r.Simple.Exporter.OTLP != nil && r.Simple.Exporter.OTLP.Endpoint != nil {
				r.Simple.Exporter.OTLP.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLP.Endpoint, r.Simple.Exporter.OTLP.Insecure)
			}
		}
	}
	return nil
}

type MetricsConfigV030 struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level configtelemetry.Level `mapstructure:"level"`

	config.MeterProvider `mapstructure:",squash"`
}

func (c *MetricsConfigV030) Unmarshal(conf *confmap.Conf) error {
	unmarshaled := *c
	if err := conf.Unmarshal(c); err != nil {
		v02 := metricsConfigV020{
			Level: unmarshaled.Level,
		}
		// try unmarshaling using v0.2.0 struct
		if e := conf.Unmarshal(&v02); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		// TODO: add a warning log to tell users to migrate their config
		return metricsConfigV02ToV03(v02, c)
	}
	// ensure endpoint normalization occurs
	for _, r := range c.Readers {
		if r.Periodic != nil {
			if r.Periodic.Exporter.OTLP != nil && r.Periodic.Exporter.OTLP.Endpoint != nil {
				r.Periodic.Exporter.OTLP.Endpoint = normalizeEndpoint(*r.Periodic.Exporter.OTLP.Endpoint, r.Periodic.Exporter.OTLP.Insecure)
			}
		}
	}
	return nil
}

type logsConfigV030 struct {
	// Level is the minimum enabled logging level.
	// (default = "INFO")
	Level zapcore.Level `mapstructure:"level"`

	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	// (default = false)
	Development bool `mapstructure:"development,omitempty"`

	// Encoding sets the logger's encoding.
	// Example values are "json", "console".
	Encoding string `mapstructure:"encoding"`

	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	// (default = false)
	DisableCaller bool `mapstructure:"disable_caller,omitempty"`

	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	// (default = false)
	DisableStacktrace bool `mapstructure:"disable_stacktrace,omitempty"`

	// Sampling sets a sampling policy.
	// Default:
	// 		sampling:
	//	   		enabled: true
	//	   		tick: 10s
	//	   		initial: 10
	//	   		thereafter: 100
	// Sampling can be disabled by setting 'enabled' to false
	Sampling *LogsSamplingConfig `mapstructure:"sampling"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must be an absolute path.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	// (default = ["stderr"])
	OutputPaths []string `mapstructure:"output_paths"`

	// ErrorOutputPaths is a list of URLs or file paths to write zap internal logger errors to.
	// The URLs could only be with "file" schema or without schema.
	// The URLs with "file" schema must use absolute paths.
	// The URLs without schema are treated as local file paths.
	// "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
	// see details at Open in zap/writer.go.
	//
	// Note that this setting only affects the zap internal logger errors.
	// (default = ["stderr"])
	ErrorOutputPaths []string `mapstructure:"error_output_paths"`

	// InitialFields is a collection of fields to add to the root logger.
	// Example:
	//
	// 		initial_fields:
	//	   		foo: "bar"
	//
	// By default, there is no initial field.
	InitialFields map[string]any `mapstructure:"initial_fields,omitempty"`

	// Processors allow configuration of log record processors to emit logs to
	// any number of supported backends.
	Processors []config.LogRecordProcessor `mapstructure:"processors,omitempty"`

	// DisableZapResource disables adding resource attributes to logs exported through Zap. This does not affect logs exported through OTLP.
	DisableZapResource bool `mapstructure:"disable_zap_resource,omitempty"`
}

// LogsConfigV030 is the exported type for use in tests and external migration.
type LogsConfigV030 = logsConfigV030

func (c *logsConfigV030) Unmarshal(conf *confmap.Conf) error {
	unmarshaled := *c
	if err := conf.Unmarshal(c); err != nil {
		v02 := logsConfigV020{
			Level:             unmarshaled.Level,
			Development:       unmarshaled.Development,
			Encoding:          unmarshaled.Encoding,
			DisableCaller:     unmarshaled.DisableCaller,
			DisableStacktrace: unmarshaled.DisableStacktrace,
			Sampling:          unmarshaled.Sampling,
			OutputPaths:       unmarshaled.OutputPaths,
			ErrorOutputPaths:  unmarshaled.ErrorOutputPaths,
			InitialFields:     unmarshaled.InitialFields,
		}
		// try unmarshaling using v0.2.0 struct
		if e := conf.Unmarshal(&v02); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		// TODO: add a warning log to tell users to migrate their config
		return logsConfigV02ToV03(v02, c)
	}
	// ensure endpoint normalization occurs
	for _, r := range c.Processors {
		if r.Batch != nil {
			if r.Batch.Exporter.OTLP != nil {
				r.Batch.Exporter.OTLP.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLP.Endpoint, r.Batch.Exporter.OTLP.Insecure)
			}
		}
		if r.Simple != nil {
			if r.Simple.Exporter.OTLP != nil && r.Simple.Exporter.OTLP.Endpoint != nil {
				r.Simple.Exporter.OTLP.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLP.Endpoint, r.Simple.Exporter.OTLP.Insecure)
			}
		}
	}
	return nil
}

func headersV03ToV1(in []config.NameStringValuePair) []otelconf.NameStringValuePair {
	out := make([]otelconf.NameStringValuePair, len(in))
	for i, h := range in {
		out[i] = otelconf.NameStringValuePair{
			Name:  h.Name,
			Value: otelconf.NameStringValuePairValue(h.Value),
		}
	}
	return out
}

func grpcTLSV03ToV1(certificate, clientCertificate, clientKey *string, insecure *bool) *otelconf.GrpcTls {
	if certificate == nil && clientCertificate == nil && clientKey == nil && insecure == nil {
		return nil
	}
	return &otelconf.GrpcTls{
		CaFile:   otelconf.GrpcTlsCaFile(certificate),
		CertFile: otelconf.GrpcTlsCertFile(clientCertificate),
		KeyFile:  otelconf.GrpcTlsKeyFile(clientKey),
		Insecure: otelconf.GrpcTlsInsecure(insecure),
	}
}

func httpTLSV03ToV1(certificate, clientCertificate, clientKey *string) *otelconf.HttpTls {
	if certificate == nil && clientCertificate == nil && clientKey == nil {
		return nil
	}
	return &otelconf.HttpTls{
		CaFile:   otelconf.HttpTlsCaFile(certificate),
		CertFile: otelconf.HttpTlsCertFile(clientCertificate),
		KeyFile:  otelconf.HttpTlsKeyFile(clientKey),
	}
}

func otlpV03ToV1(in *config.OTLP) (grpc *otelconf.OTLPGrpcExporter, http *otelconf.OTLPHttpExporter) {
	if in == nil {
		return nil, nil
	}
	if in.Protocol != nil && strings.HasPrefix(*in.Protocol, "http") {
		return nil, &otelconf.OTLPHttpExporter{
			Compression: otelconf.OTLPHttpExporterCompression(in.Compression),
			Endpoint:    otelconf.OTLPHttpExporterEndpoint(in.Endpoint),
			Headers:     headersV03ToV1(in.Headers),
			HeadersList: otelconf.OTLPHttpExporterHeadersList(in.HeadersList),
			Timeout:     otelconf.OTLPHttpExporterTimeout(in.Timeout),
			Tls:         httpTLSV03ToV1(in.Certificate, in.ClientCertificate, in.ClientKey),
		}
	}
	return &otelconf.OTLPGrpcExporter{
		Compression: otelconf.OTLPGrpcExporterCompression(in.Compression),
		Endpoint:    otelconf.OTLPGrpcExporterEndpoint(in.Endpoint),
		Headers:     headersV03ToV1(in.Headers),
		HeadersList: otelconf.OTLPGrpcExporterHeadersList(in.HeadersList),
		Timeout:     otelconf.OTLPGrpcExporterTimeout(in.Timeout),
		Tls:         grpcTLSV03ToV1(in.Certificate, in.ClientCertificate, in.ClientKey, in.Insecure),
	}, nil
}

func spanExporterV03ToV1(in config.SpanExporter) otelconf.SpanExporter {
	grpc, http := otlpV03ToV1(in.OTLP)
	return otelconf.SpanExporter{
		Console:  otelconf.ConsoleExporter(in.Console),
		OTLPGrpc: grpc,
		OTLPHttp: http,
	}
}

func tracesConfigV03ToV1(v3 TracesConfigV030, v1 *TracesConfigV1) error {
	processors := make([]otelconf.SpanProcessor, len(v3.Processors))
	for idx, p := range v3.Processors {
		processors[idx] = otelconf.SpanProcessor{}
		if p.Batch != nil {
			processors[idx].Batch = &otelconf.BatchSpanProcessor{
				ExportTimeout:      otelconf.BatchSpanProcessorExportTimeout(p.Batch.ExportTimeout),
				MaxExportBatchSize: otelconf.BatchSpanProcessorMaxExportBatchSize(p.Batch.MaxExportBatchSize),
				MaxQueueSize:       otelconf.BatchSpanProcessorMaxQueueSize(p.Batch.MaxQueueSize),
				ScheduleDelay:      otelconf.BatchSpanProcessorScheduleDelay(p.Batch.ScheduleDelay),
				Exporter:           spanExporterV03ToV1(p.Batch.Exporter),
			}
		}
		if p.Simple != nil {
			processors[idx].Simple = &otelconf.SimpleSpanProcessor{
				Exporter: spanExporterV03ToV1(p.Simple.Exporter),
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v1.Level = v3.Level
	v1.Propagators = v3.Propagators
	v1.Processors = processors
	return nil
}

func otlpMetricV03ToV1(in *config.OTLPMetric) (grpc *otelconf.OTLPGrpcMetricExporter, http *otelconf.OTLPHttpMetricExporter) {
	if in == nil {
		return nil, nil
	}
	var histAgg *otelconf.ExporterDefaultHistogramAggregation
	if in.DefaultHistogramAggregation != nil {
		v := otelconf.ExporterDefaultHistogramAggregation(*in.DefaultHistogramAggregation)
		histAgg = &v
	}
	var temporality *otelconf.ExporterTemporalityPreference
	if in.TemporalityPreference != nil {
		v := otelconf.ExporterTemporalityPreference(*in.TemporalityPreference)
		temporality = &v
	}
	if in.Protocol != nil && strings.HasPrefix(*in.Protocol, "http") {
		return nil, &otelconf.OTLPHttpMetricExporter{
			Compression:                 otelconf.OTLPHttpMetricExporterCompression(in.Compression),
			Endpoint:                    otelconf.OTLPHttpMetricExporterEndpoint(in.Endpoint),
			Headers:                     headersV03ToV1(in.Headers),
			HeadersList:                 otelconf.OTLPHttpMetricExporterHeadersList(in.HeadersList),
			Timeout:                     otelconf.OTLPHttpMetricExporterTimeout(in.Timeout),
			DefaultHistogramAggregation: histAgg,
			TemporalityPreference:       temporality,
			Tls:                         httpTLSV03ToV1(in.Certificate, in.ClientCertificate, in.ClientKey),
		}
	}
	return &otelconf.OTLPGrpcMetricExporter{
		Compression:                 otelconf.OTLPGrpcMetricExporterCompression(in.Compression),
		Endpoint:                    otelconf.OTLPGrpcMetricExporterEndpoint(in.Endpoint),
		Headers:                     headersV03ToV1(in.Headers),
		HeadersList:                 otelconf.OTLPGrpcMetricExporterHeadersList(in.HeadersList),
		Timeout:                     otelconf.OTLPGrpcMetricExporterTimeout(in.Timeout),
		DefaultHistogramAggregation: histAgg,
		TemporalityPreference:       temporality,
		Tls:                         grpcTLSV03ToV1(in.Certificate, in.ClientCertificate, in.ClientKey, in.Insecure),
	}, nil
}

func prometheusV03ToV1(in *config.Prometheus) *otelconf.ExperimentalPrometheusMetricExporter {
	if in == nil {
		return nil
	}
	var withResourceLabels *otelconf.IncludeExclude
	if in.WithResourceConstantLabels != nil {
		withResourceLabels = &otelconf.IncludeExclude{
			Excluded: in.WithResourceConstantLabels.Excluded,
			Included: in.WithResourceConstantLabels.Included,
		}
	}
    strategy := otelconf.ExperimentalPrometheusTranslationStrategyNoUtf8EscapingWithSuffixes
            if in.WithoutTypeSuffix != nil && *in.WithoutTypeSuffix &&
                    in.WithoutUnits != nil && *in.WithoutUnits {
                    strategy =
    otelconf.ExperimentalPrometheusTranslationStrategyUnderscoreEscapingWithoutSuffixes
            }

	return &otelconf.ExperimentalPrometheusMetricExporter{
		Host:                       otelconf.ExperimentalPrometheusMetricExporterHost(in.Host),
		Port:                       otelconf.ExperimentalPrometheusMetricExporterPort(in.Port),
		WithoutScopeInfo:           otelconf.ExperimentalPrometheusMetricExporterWithoutScopeInfo(in.WithoutScopeInfo),
		WithResourceConstantLabels: withResourceLabels,
	}
}

func metricsConfigV03ToV1(v3 MetricsConfigV030, v1 *MetricsConfigV1) error {
	readers := make([]otelconf.MetricReader, len(v3.Readers))
	for idx, r := range v3.Readers {
		readers[idx] = otelconf.MetricReader{}
		if r.Periodic != nil {
			grpc, http := otlpMetricV03ToV1(r.Periodic.Exporter.OTLP)
			var console *otelconf.ConsoleMetricExporter
			if r.Periodic.Exporter.Console != nil {
				console = &otelconf.ConsoleMetricExporter{}
			}
			readers[idx].Periodic = &otelconf.PeriodicMetricReader{
				Interval: otelconf.PeriodicMetricReaderInterval(r.Periodic.Interval),
				Timeout:  otelconf.PeriodicMetricReaderTimeout(r.Periodic.Timeout),
				Exporter: otelconf.PushMetricExporter{
					Console:  console,
					OTLPGrpc: grpc,
					OTLPHttp: http,
				},
			}
		}
		if r.Pull != nil {
			readers[idx].Pull = &otelconf.PullMetricReader{
				Exporter: otelconf.PullMetricExporter{
					PrometheusDevelopment: prometheusV03ToV1(r.Pull.Exporter.Prometheus),
				},
			}
		}
	}
	v1.Level = v3.Level
	v1.Readers = readers
	return nil
}

func logExporterV03ToV1(in config.LogRecordExporter) otelconf.LogRecordExporter {
	grpc, http := otlpV03ToV1(in.OTLP)
	return otelconf.LogRecordExporter{
		Console:  otelconf.ConsoleExporter(in.Console),
		OTLPGrpc: grpc,
		OTLPHttp: http,
	}
}

func logsConfigV03ToV1(v3 logsConfigV030, v1 *LogsConfigV1) error {
	processors := make([]otelconf.LogRecordProcessor, len(v3.Processors))
	for idx, p := range v3.Processors {
		processors[idx] = otelconf.LogRecordProcessor{}
		if p.Batch != nil {
			processors[idx].Batch = &otelconf.BatchLogRecordProcessor{
				ExportTimeout:      otelconf.BatchLogRecordProcessorExportTimeout(p.Batch.ExportTimeout),
				MaxExportBatchSize: otelconf.BatchLogRecordProcessorMaxExportBatchSize(p.Batch.MaxExportBatchSize),
				MaxQueueSize:       otelconf.BatchLogRecordProcessorMaxQueueSize(p.Batch.MaxQueueSize),
				ScheduleDelay:      otelconf.BatchLogRecordProcessorScheduleDelay(p.Batch.ScheduleDelay),
				Exporter:           logExporterV03ToV1(p.Batch.Exporter),
			}
		}
		if p.Simple != nil {
			processors[idx].Simple = &otelconf.SimpleLogRecordProcessor{
				Exporter: logExporterV03ToV1(p.Simple.Exporter),
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v1.Level = v3.Level
	v1.Development = v3.Development
	v1.Encoding = v3.Encoding
	v1.DisableCaller = v3.DisableCaller
	v1.DisableStacktrace = v3.DisableStacktrace
	v1.Sampling = v3.Sampling
	v1.OutputPaths = v3.OutputPaths
	v1.ErrorOutputPaths = v3.ErrorOutputPaths
	v1.InitialFields = v3.InitialFields
	v1.Processors = processors
	v1.DisableZapResource = v3.DisableZapResource
	return nil
}
