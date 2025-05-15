// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/internal/migration"

import (
	configv02 "go.opentelemetry.io/contrib/otelconf/v0.2.0"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
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
	Address string                   `mapstructure:"address"`
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

func otlpV02ToV03(in *configv02.OTLP) *config.OTLP {
	if in == nil {
		return nil
	}
	return &config.OTLP{
		Certificate:       in.Certificate,
		ClientCertificate: in.ClientCertificate,
		ClientKey:         in.ClientKey,
		Compression:       in.Compression,
		Endpoint:          normalizeEndpoint(in.Endpoint),
		Insecure:          in.Insecure,
		Protocol:          &in.Protocol,
		Timeout:           in.Timeout,
		Headers:           headersV02ToV03(in.Headers),
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
		Endpoint:          normalizeEndpoint(in.Endpoint),
		Insecure:          in.Insecure,
		Protocol:          &in.Protocol,
		Timeout:           in.Timeout,
		Headers:           headersV02ToV03(in.Headers),
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
	v3.Address = v2.Address
	v3.Readers = readers
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
