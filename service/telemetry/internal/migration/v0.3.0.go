// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/internal/migration"

import (
	"time"

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

type LogsConfigV030 struct {
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

// LogsSamplingConfig sets a sampling strategy for the logger. Sampling caps the
// global CPU and I/O load that logging puts on your process while attempting
// to preserve a representative subset of your logs.
type LogsSamplingConfig struct {
	// Enabled enable sampling logging
	Enabled bool `mapstructure:"enabled"`
	// Tick represents the interval in seconds that the logger apply each sampling.
	Tick time.Duration `mapstructure:"tick"`
	// Initial represents the first M messages logged each Tick.
	Initial int `mapstructure:"initial"`
	// Thereafter represents the sampling rate, every Nth message will be sampled after Initial messages are logged during each Tick.
	// If Thereafter is zero, the logger will drop all the messages after the Initial each Tick.
	Thereafter int `mapstructure:"thereafter"`
}

func (c *LogsConfigV030) Unmarshal(conf *confmap.Conf) error {
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
