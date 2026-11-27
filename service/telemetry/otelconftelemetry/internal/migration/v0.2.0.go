// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"

import (
	"errors"

	configv02 "go.opentelemetry.io/contrib/otelconf/v0.2.0"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	configv120 "go.opentelemetry.io/contrib/otelconf/x"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
)

type tracesConfigV020 struct {
	Level       configtelemetry.Level     `mapstructure:"level"`
	Propagators []string                  `mapstructure:"propagators"`
	Processors  []configv02.SpanProcessor `mapstructure:"processors"`
}

type metricsConfigV020 struct {
	Level   configtelemetry.Level    `mapstructure:"level"`
	Readers []configv02.MetricReader `mapstructure:"readers"`
}

type logsConfigV020 struct {
	Level             zapcore.Level                  `mapstructure:"level"`
	Development       bool                           `mapstructure:"development"`
	Encoding          string                         `mapstructure:"encoding"`
	DisableCaller     bool                           `mapstructure:"disable_caller"`
	DisableStacktrace bool                           `mapstructure:"disable_stacktrace"`
	Sampling          *LogsSamplingConfig            `mapstructure:"sampling"`
	OutputPaths       []string                       `mapstructure:"output_paths"`
	ErrorOutputPaths  []string                       `mapstructure:"error_output_paths"`
	InitialFields     map[string]any                 `mapstructure:"initial_fields"`
	Processors        []configv02.LogRecordProcessor `mapstructure:"processors"`
}

func headersV02ToV03(in configv02.Headers) []config.NameStringValuePair {
	headers := make([]config.NameStringValuePair, 0, len(in))
	for k, v := range in {
		headers = append(headers, config.NameStringValuePair{
			Name:  k,
			Value: &v,
		})
	}
	return headers
}

func headersV02ToV120(in configv02.Headers) []configv120.NameStringValuePair {
	headers := make([]configv120.NameStringValuePair, 0, len(in))
	for k, v := range in {
		headers = append(headers, configv120.NameStringValuePair{
			Name:  k,
			Value: &v,
		})
	}
	return headers
}

func otlpV02ToV03(in *configv02.OTLP) *config.OTLP {
	if in == nil {
		return nil
	}
	return &config.OTLP{
		Certificate:       in.Certificate,
		ClientCertificate: in.ClientCertificate,
		ClientKey:         in.ClientKey,
		Compression:       in.Compression,
		Endpoint:          normalizeEndpoint(in.Endpoint, in.Insecure),
		Insecure:          in.Insecure,
		Protocol:          &in.Protocol,
		Timeout:           in.Timeout,
		Headers:           headersV02ToV03(in.Headers),
	}
}

func otlpGRPCV02ToV120(in *configv02.OTLP) *configv120.OTLPGrpcExporter {
	if in == nil {
		return nil
	}
	return &configv120.OTLPGrpcExporter{
		Compression: in.Compression,
		Endpoint:    &in.Endpoint,
		Headers:     headersV02ToV120(in.Headers),
		Timeout:     in.Timeout,
		Tls: &configv120.GrpcTls{
			CaFile:   in.Certificate,
			CertFile: in.ClientCertificate,
			KeyFile:  in.ClientKey,
			Insecure: in.Insecure,
		},
	}
}

func otlpHTTPV02ToV120(in *configv02.OTLP) *configv120.OTLPHttpExporter {
	if in == nil {
		return nil
	}
	return &configv120.OTLPHttpExporter{
		Compression: in.Compression,
		Endpoint:    &in.Endpoint,
		Headers:     headersV02ToV120(in.Headers),
		Timeout:     in.Timeout,
		Tls: &configv120.HttpTls{
			CaFile:   in.Certificate,
			CertFile: in.ClientCertificate,
			KeyFile:  in.ClientKey,
		},
	}
}

func otlpMetricV02ToV03(in *configv02.OTLPMetric) *config.OTLPMetric {
	if in == nil {
		return nil
	}
	return &config.OTLPMetric{
		Certificate:       in.Certificate,
		ClientCertificate: in.ClientCertificate,
		ClientKey:         in.ClientKey,
		Compression:       in.Compression,
		Endpoint:          normalizeEndpoint(in.Endpoint, in.Insecure),
		Insecure:          in.Insecure,
		Protocol:          &in.Protocol,
		Timeout:           in.Timeout,
		Headers:           headersV02ToV03(in.Headers),
	}
}

func otlpMetricGRPCV02ToV120(in *configv02.OTLPMetric) *configv120.OTLPGrpcMetricExporter {
	if in == nil {
		return nil
	}
	return &configv120.OTLPGrpcMetricExporter{
		Compression: in.Compression,
		Endpoint:    &in.Endpoint,
		Headers:     headersV02ToV120(in.Headers),
		Timeout:     in.Timeout,
		Tls: &configv120.GrpcTls{
			CaFile:   in.Certificate,
			CertFile: in.ClientCertificate,
			KeyFile:  in.ClientKey,
			Insecure: in.Insecure,
		},
	}
}

func otlpMetricHTTPV02ToV120(in *configv02.OTLPMetric) *configv120.OTLPHttpMetricExporter {
	if in == nil {
		return nil
	}
	return &configv120.OTLPHttpMetricExporter{
		Compression: in.Compression,
		Endpoint:    &in.Endpoint,
		Headers:     headersV02ToV120(in.Headers),
		Timeout:     in.Timeout,
		Tls: &configv120.HttpTls{
			CaFile:   in.Certificate,
			CertFile: in.ClientCertificate,
			KeyFile:  in.ClientKey,
		},
	}
}

func zipkinV02ToV03(in *configv02.Zipkin) *config.Zipkin {
	if in == nil {
		return nil
	}
	return &config.Zipkin{
		Endpoint: &in.Endpoint,
		Timeout:  in.Timeout,
	}
}

func tracesConfigV02ToV03(v2 tracesConfigV020, v3 *TracesConfigV030) error {
	processors := make([]config.SpanProcessor, len(v2.Processors))
	for idx, p := range v2.Processors {
		processors[idx] = config.SpanProcessor{}
		if p.Batch != nil {
			processors[idx].Batch = &config.BatchSpanProcessor{
				ExportTimeout:      p.Batch.ExportTimeout,
				MaxExportBatchSize: p.Batch.MaxExportBatchSize,
				MaxQueueSize:       p.Batch.MaxQueueSize,
				ScheduleDelay:      p.Batch.ScheduleDelay,
				Exporter: config.SpanExporter{
					Console:              config.Console(p.Batch.Exporter.Console),
					OTLP:                 otlpV02ToV03(p.Batch.Exporter.OTLP),
					Zipkin:               zipkinV02ToV03(p.Batch.Exporter.Zipkin),
					AdditionalProperties: p.Batch.Exporter.AdditionalProperties,
				},
			}
		}
		if p.Simple != nil {
			processors[idx].Simple = &config.SimpleSpanProcessor{
				Exporter: config.SpanExporter{
					Console:              config.Console(p.Simple.Exporter.Console),
					OTLP:                 otlpV02ToV03(p.Simple.Exporter.OTLP),
					Zipkin:               zipkinV02ToV03(p.Simple.Exporter.Zipkin),
					AdditionalProperties: p.Simple.Exporter.AdditionalProperties,
				},
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v3.Level = v2.Level
	v3.Propagators = v2.Propagators
	v3.Processors = processors
	return nil
}

func tracesConfigV02ToV120(v2 tracesConfigV020, v120 *TracesConfigV120) error {
	processors := make([]configv120.SpanProcessor, len(v2.Processors))
	for idx, p := range v2.Processors {
		processors[idx] = configv120.SpanProcessor{}
		if p.Batch != nil {
			var protocol string
			if p.Batch.Exporter.OTLP != nil {
				protocol = p.Batch.Exporter.OTLP.Protocol
			}
			if p.Batch.Exporter.Zipkin != nil {
				return errors.New("zipkin exporter is no longer supported, please update your configuration")
			}
			processors[idx].Batch = &configv120.BatchSpanProcessor{
				ExportTimeout:      p.Batch.ExportTimeout,
				MaxExportBatchSize: p.Batch.MaxExportBatchSize,
				MaxQueueSize:       p.Batch.MaxQueueSize,
				ScheduleDelay:      p.Batch.ScheduleDelay,
				Exporter: configv120.SpanExporter{
					AdditionalProperties: p.Batch.Exporter.AdditionalProperties,
				},
			}
			// TODO: set console if configured
			// Console:              configv120.Console(p.Batch.Exporter.Console),
			switch protocol {
			case "grpc":
				processors[idx].Batch.Exporter.OTLPGrpc = otlpGRPCV02ToV120(p.Batch.Exporter.OTLP)
			case "http/protobuf":
				processors[idx].Batch.Exporter.OTLPHttp = otlpHTTPV02ToV120(p.Batch.Exporter.OTLP)
			}
		}
		if p.Simple != nil {
			if p.Simple.Exporter.Zipkin != nil {
				return errors.New("zipkin exporter is no longer supported, please update your configuration")
			}
			var protocol string
			if p.Simple.Exporter.OTLP != nil {
				protocol = p.Simple.Exporter.OTLP.Protocol
			}
			processors[idx].Simple = &configv120.SimpleSpanProcessor{
				Exporter: configv120.SpanExporter{
					AdditionalProperties: p.Simple.Exporter.AdditionalProperties,
				},
			}
			// TODO: set console if configured
			// Console:              configv120.Console(p.Simple.Exporter.Console),
			switch protocol {
			case "grpc":
				processors[idx].Simple.Exporter.OTLPGrpc = otlpGRPCV02ToV120(p.Simple.Exporter.OTLP)
			case "http/protobuf":
				processors[idx].Simple.Exporter.OTLPHttp = otlpHTTPV02ToV120(p.Simple.Exporter.OTLP)
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v120.Level = v2.Level
	v120.Propagators = v2.Propagators
	v120.Processors = processors
	return nil
}

func prometheusV02ToV03(in *configv02.Prometheus) *config.Prometheus {
	if in == nil {
		return nil
	}
	return &config.Prometheus{
		Host:                       in.Host,
		Port:                       in.Port,
		WithoutScopeInfo:           in.WithoutScopeInfo,
		WithoutUnits:               in.WithoutUnits,
		WithoutTypeSuffix:          in.WithoutTypeSuffix,
		WithResourceConstantLabels: (*config.IncludeExclude)(in.WithResourceConstantLabels),
	}
}

// type ExperimentalPrometheusMetricExporter struct {
// 	// Configure how metric names are translated to Prometheus metric names.
// 	// Values include:
// 	// * no_translation: Special character escaping is disabled. Type and unit
// 	// suffixes are disabled. Metric names are unaltered.
// 	// * no_utf8_escaping_with_suffixes: Special character escaping is disabled. Type
// 	// and unit suffixes are enabled.
// 	// * underscore_escaping_with_suffixes: Special character escaping is enabled.
// 	// Type and unit suffixes are enabled.
// 	// * underscore_escaping_without_suffixes: Special character escaping is enabled.
// 	// Type and unit suffixes are disabled. This represents classic Prometheus metric
// 	// name compatibility.
// 	// If omitted, underscore_escaping_with_suffixes is used.
// 	//
// 	TranslationStrategy *ExperimentalPrometheusTranslationStrategy `json:"translation_strategy,omitempty,omitzero" yaml:"translation_strategy,omitempty" mapstructure:"translation_strategy,omitempty"`

// 	// Configure Prometheus Exporter to produce metrics without a target info metric
// 	// for the resource.
// 	// If omitted or null, false is used.
// 	//
// 	WithoutTargetInfo ExperimentalPrometheusMetricExporterWithoutTargetInfo `json:"without_target_info,omitempty,omitzero" yaml:"without_target_info,omitempty" mapstructure:"without_target_info,omitempty"`
// }

func prometheusV02ToV120(in *configv02.Prometheus) *configv120.ExperimentalPrometheusMetricExporter {
	if in == nil {
		return nil
	}
	return &configv120.ExperimentalPrometheusMetricExporter{
		Host:             in.Host,
		Port:             in.Port,
		WithoutScopeInfo: in.WithoutScopeInfo,
		// TODO: fill in the following
		// TranslationStrategy:
		// WithoutTargetInfo:
		WithResourceConstantLabels: (*configv120.IncludeExclude)(in.WithResourceConstantLabels),
	}
}

func metricsConfigV02ToV03(v2 metricsConfigV020, v3 *MetricsConfigV030) error {
	readers := make([]config.MetricReader, len(v2.Readers))
	for idx, p := range v2.Readers {
		readers[idx] = config.MetricReader{}
		if p.Periodic != nil {
			readers[idx].Periodic = &config.PeriodicMetricReader{
				Interval: p.Periodic.Interval,
				Timeout:  p.Periodic.Timeout,
				Exporter: config.PushMetricExporter{
					Console:              config.Console(p.Periodic.Exporter.Console),
					OTLP:                 otlpMetricV02ToV03(p.Periodic.Exporter.OTLP),
					AdditionalProperties: p.Periodic.Exporter.AdditionalProperties,
				},
			}
		}
		if p.Pull != nil {
			readers[idx].Pull = &config.PullMetricReader{
				Exporter: config.PullMetricExporter{
					Prometheus:           prometheusV02ToV03(p.Pull.Exporter.Prometheus),
					AdditionalProperties: p.Pull.Exporter.AdditionalProperties,
				},
			}
		}
	}
	v3.Level = v2.Level
	v3.Readers = readers
	return nil
}

func metricsConfigV02ToV120(v2 metricsConfigV020, v120 *MetricsConfigV120) error {
	readers := make([]configv120.MetricReader, len(v2.Readers))
	for idx, p := range v2.Readers {
		readers[idx] = configv120.MetricReader{}
		if p.Periodic != nil {
			var protocol string
			if p.Periodic.Exporter.OTLP != nil {
				protocol = p.Periodic.Exporter.OTLP.Protocol
			}
			readers[idx].Periodic = &configv120.PeriodicMetricReader{
				Interval: p.Periodic.Interval,
				Timeout:  p.Periodic.Timeout,
				Exporter: configv120.PushMetricExporter{
					AdditionalProperties: p.Periodic.Exporter.AdditionalProperties,
				},
			}
			// Console:              configv120.ConsoleMetricExporter(p.Periodic.Exporter.Console),
			// TODO: set console if configured
			switch protocol {
			case "grpc":
				readers[idx].Periodic.Exporter.OTLPGrpc = otlpMetricGRPCV02ToV120(p.Periodic.Exporter.OTLP)
			case "http/protobuf":
				readers[idx].Periodic.Exporter.OTLPHttp = otlpMetricHTTPV02ToV120(p.Periodic.Exporter.OTLP)
			}

		}
		if p.Pull != nil {
			readers[idx].Pull = &configv120.PullMetricReader{
				Exporter: configv120.PullMetricExporter{
					PrometheusDevelopment: prometheusV02ToV120(p.Pull.Exporter.Prometheus),
					AdditionalProperties:  p.Pull.Exporter.AdditionalProperties,
				},
			}
		}
	}
	v120.Level = v2.Level
	v120.Readers = readers
	return nil
}

func logsConfigV02ToV03(v2 logsConfigV020, v3 *LogsConfigV030) error {
	processors := make([]config.LogRecordProcessor, len(v2.Processors))
	for idx, p := range v2.Processors {
		processors[idx] = config.LogRecordProcessor{}
		if p.Batch != nil {
			processors[idx].Batch = &config.BatchLogRecordProcessor{
				ExportTimeout:      p.Batch.ExportTimeout,
				MaxExportBatchSize: p.Batch.MaxExportBatchSize,
				MaxQueueSize:       p.Batch.MaxQueueSize,
				ScheduleDelay:      p.Batch.ScheduleDelay,
				Exporter: config.LogRecordExporter{
					Console:              config.Console(p.Batch.Exporter.Console),
					OTLP:                 otlpV02ToV03(p.Batch.Exporter.OTLP),
					AdditionalProperties: p.Batch.Exporter.AdditionalProperties,
				},
			}
		}
		if p.Simple != nil {
			processors[idx].Simple = &config.SimpleLogRecordProcessor{
				Exporter: config.LogRecordExporter{
					Console:              config.Console(p.Simple.Exporter.Console),
					OTLP:                 otlpV02ToV03(p.Simple.Exporter.OTLP),
					AdditionalProperties: p.Simple.Exporter.AdditionalProperties,
				},
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v3.Level = v2.Level
	v3.Development = v2.Development
	v3.Encoding = v2.Encoding
	v3.DisableCaller = v2.DisableCaller
	v3.DisableStacktrace = v2.DisableStacktrace
	v3.Sampling = v2.Sampling
	v3.OutputPaths = v2.OutputPaths
	v3.ErrorOutputPaths = v2.ErrorOutputPaths
	v3.InitialFields = v2.InitialFields
	v3.Processors = processors
	return nil
}

func logsConfigV02ToV120(v2 logsConfigV020, v120 *LogsConfigV120) error {
	processors := make([]configv120.LogRecordProcessor, len(v2.Processors))
	for idx, p := range v2.Processors {
		processors[idx] = configv120.LogRecordProcessor{}
		if p.Batch != nil {
			var protocol string
			if p.Batch.Exporter.OTLP != nil {
				protocol = p.Batch.Exporter.OTLP.Protocol
			}
			processors[idx].Batch = &configv120.BatchLogRecordProcessor{
				ExportTimeout:      p.Batch.ExportTimeout,
				MaxExportBatchSize: p.Batch.MaxExportBatchSize,
				MaxQueueSize:       p.Batch.MaxQueueSize,
				ScheduleDelay:      p.Batch.ScheduleDelay,
				Exporter: configv120.LogRecordExporter{
					AdditionalProperties: p.Batch.Exporter.AdditionalProperties,
				},
			}
			// TODO: set console if configured
			// Console:              configv120.Console(p.Batch.Exporter.Console),
			switch protocol {
			case "grpc":
				processors[idx].Batch.Exporter.OTLPGrpc = otlpGRPCV02ToV120(p.Batch.Exporter.OTLP)
			case "http/protobuf":
				processors[idx].Batch.Exporter.OTLPHttp = otlpHTTPV02ToV120(p.Batch.Exporter.OTLP)
			}
		}
		if p.Simple != nil {
			var protocol string
			if p.Simple.Exporter.OTLP != nil {
				protocol = p.Simple.Exporter.OTLP.Protocol
			}
			processors[idx].Simple = &configv120.SimpleLogRecordProcessor{
				Exporter: configv120.LogRecordExporter{
					AdditionalProperties: p.Simple.Exporter.AdditionalProperties,
				},
			}
			// TODO: set console if configured
			// Console:              configv120.Console(p.Simple.Exporter.Console),
			switch protocol {
			case "grpc":
				processors[idx].Simple.Exporter.OTLPGrpc = otlpGRPCV02ToV120(p.Simple.Exporter.OTLP)
			case "http/protobuf":
				processors[idx].Simple.Exporter.OTLPHttp = otlpHTTPV02ToV120(p.Simple.Exporter.OTLP)
			}
		}
		processors[idx].AdditionalProperties = p.AdditionalProperties
	}
	v120.Level = v2.Level
	v120.Development = v2.Development
	v120.Encoding = v2.Encoding
	v120.DisableCaller = v2.DisableCaller
	v120.DisableStacktrace = v2.DisableStacktrace
	v120.Sampling = v2.Sampling
	v120.OutputPaths = v2.OutputPaths
	v120.ErrorOutputPaths = v2.ErrorOutputPaths
	v120.InitialFields = v2.InitialFields
	v120.Processors = processors
	return nil
}
