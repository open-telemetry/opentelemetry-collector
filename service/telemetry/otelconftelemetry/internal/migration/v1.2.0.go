// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package migration // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry/internal/migration"

import (
	"errors"
	"fmt"

	config "go.opentelemetry.io/contrib/otelconf/x"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
)

type TracesConfigV120 struct {
	// Level configures whether spans are emitted or not, the possible values are:
	//  - "none" indicates that no tracing data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	Level configtelemetry.Level `mapstructure:"level"`
	// Propagators is a list of TextMapPropagators from the supported propagators list. Currently,
	// tracecontext and  b3 are supported. By default, the value is set to empty list and
	// context propagation is disabled.
	Propagators []string `mapstructure:"propagators"`

	config.TracerProvider `mapstructure:",squash"`

	// MigratedFromV02 is set when the config was transparently migrated from the v0.2.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV02 bool `mapstructure:"-"`
	// MigratedFromV03 is set when the config was transparently migrated from the v0.3.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV03 bool `mapstructure:"-"`
}

func (c *TracesConfigV120) Unmarshal(conf *confmap.Conf) error {
	unmarshaled := *c
	if err := conf.Unmarshal(c); err != nil {
		v2TracesConfig := tracesConfigV020{
			Level:       unmarshaled.Level,
			Propagators: unmarshaled.Propagators,
		}
		// try unmarshaling using v0.2.0 struct
		if e := conf.Unmarshal(&v2TracesConfig); e == nil {
			c.MigratedFromV02 = true
			return tracesConfigV02ToV120(v2TracesConfig, c)
		}
		v03 := TracesConfigV030{
			Level:       unmarshaled.Level,
			Propagators: unmarshaled.Propagators,
		}
		// try unmarshaling using v0.3.0 struct
		if e := conf.Unmarshal(&v03); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		c.MigratedFromV03 = true
		return tracesConfigV03ToV120(v03, c)
	}
	// ensure endpoint normalization occurs
	for _, r := range c.Processors {
		if r.Batch != nil {
			if r.Batch.Exporter.OTLPGrpc != nil {
				r.Batch.Exporter.OTLPGrpc.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLPGrpc.Endpoint, r.Batch.Exporter.OTLPGrpc.Tls.Insecure)
			}
			if r.Batch.Exporter.OTLPHttp != nil {
				r.Batch.Exporter.OTLPHttp.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLPHttp.Endpoint, new(false))
			}
		}
		if r.Simple != nil {
			if r.Simple.Exporter.OTLPGrpc != nil {
				r.Simple.Exporter.OTLPGrpc.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLPGrpc.Endpoint, r.Simple.Exporter.OTLPGrpc.Tls.Insecure)
			}
			if r.Simple.Exporter.OTLPHttp != nil {
				r.Simple.Exporter.OTLPHttp.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLPHttp.Endpoint, new(false))
			}
		}
	}
	return nil
}

var _ confmap.Marshaler = TracesConfigV120{}

func (c TracesConfigV120) Marshal(conf *confmap.Conf) error {
	if err := conf.Marshal(c); err != nil {
		return fmt.Errorf("otelconftelemetry: failed to marshal traces configuration: %w", err)
	}

	// Redact header values the way configopaque would
	sm := conf.ToStringMap()
	redactHeaders(sm, "processors.*.simple|batch.exporter.otlp_http|otlp_grpc.headers.*.value")
	return conf.Marshal(sm)
}

type MetricsConfigV120 struct {
	// Level is the level of telemetry metrics, the possible values are:
	//  - "none" indicates that no telemetry data should be collected;
	//  - "basic" is the recommended and covers the basics of the service telemetry.
	//  - "normal" adds some other indicators on top of basic.
	//  - "detailed" adds dimensions and views to the previous levels.
	Level configtelemetry.Level `mapstructure:"level"`

	config.MeterProvider `mapstructure:",squash"`

	// MigratedFromV02 is set when the config was transparently migrated from the v0.2.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV02 bool `mapstructure:"-"`

	// MigratedFromV03 is set when the config was transparently migrated from the v0.3.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV03 bool `mapstructure:"-"`

	// PrometheusStableKeyReaders holds the indices of the readers whose pull
	// exporter was configured with the stable `prometheus` key and rewritten to
	// `prometheus/development` on unmarshal. Marshal uses it to emit the key
	// the user wrote. It is not part of the config schema.
	PrometheusStableKeyReaders map[int]bool `mapstructure:"-"`
}

const (
	prometheusKey            = "prometheus"
	prometheusDevelopmentKey = "prometheus/development"
)

// metricsConfigV120Plain has the fields of MetricsConfigV120 but not its
// Unmarshal method. Unmarshaling a rewritten copy of the config into it
// avoids confmap re-invoking MetricsConfigV120.Unmarshal indefinitely, which
// would happen because the rewritten Conf is not the one confmap passed in.
type metricsConfigV120Plain MetricsConfigV120

// renamePullExporterKey renames the key `from` to `to` in the
// readers[*].pull.exporter map of each reader of the given metrics
// configuration. If only is non-nil, only readers whose index is in it are
// considered. It returns the indices of the readers that were renamed, and an
// error if both keys are set on the same exporter.
func renamePullExporterKey(metrics map[string]any, from, to string, only map[int]bool) (map[int]bool, error) {
	renamed := map[int]bool{}
	readers, _ := metrics["readers"].([]any)
	for i, r := range readers {
		if only != nil && !only[i] {
			continue
		}
		reader, _ := r.(map[string]any)
		pull, _ := reader["pull"].(map[string]any)
		exporter, _ := pull["exporter"].(map[string]any)
		v, ok := exporter[from]
		if !ok {
			continue
		}
		if _, exists := exporter[to]; exists {
			return nil, fmt.Errorf("pull exporter cannot set both %q and %q", from, to)
		}
		exporter[to] = v
		delete(exporter, from)
		renamed[i] = true
	}
	return renamed, nil
}

// rewritePrometheusDevelopment rewrites the stable `prometheus` pull exporter
// key to `prometheus/development`, which is what the underlying otelconf
// types currently understand. This lets users write the stable key today,
// without impacting them once otelconf supports it natively. It returns the
// indices of the readers that were rewritten, so that Marshal can restore the
// stable key for exactly those readers.
func rewritePrometheusDevelopment(metrics map[string]any) (map[int]bool, error) {
	return renamePullExporterKey(metrics, prometheusKey, prometheusDevelopmentKey, nil)
}

// rewritePrometheusStable is the inverse of rewritePrometheusDevelopment: it
// rewrites `prometheus/development` back to `prometheus` for the readers at
// the given indices, used when marshaling.
func rewritePrometheusStable(metrics map[string]any, readers map[int]bool) error {
	if len(readers) == 0 {
		return nil
	}
	_, err := renamePullExporterKey(metrics, prometheusDevelopmentKey, prometheusKey, readers)
	return err
}

func (c *MetricsConfigV120) Unmarshal(conf *confmap.Conf) error {
	unmarshaled := *c

	// Accept the stable `prometheus` key by rewriting it to `prometheus/development`.
	sm := conf.ToStringMap()
	stableReaders, err := rewritePrometheusDevelopment(sm)
	if err != nil {
		return err
	}

	if err = confmap.NewFromStringMap(sm).Unmarshal((*metricsConfigV120Plain)(c)); err != nil {
		v02 := metricsConfigV020{
			Level: unmarshaled.Level,
		}
		// try unmarshaling using v0.2.0 struct
		if e := conf.Unmarshal(&v02); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		c.MigratedFromV02 = true
		return metricsConfigV02ToV120(v02, c)
	}
	c.PrometheusStableKeyReaders = stableReaders
	// if err := conf.Unmarshal(c); err != nil {
	// 	v03 := MetricsConfigV030{
	// 		Level: unmarshaled.Level,
	// 	}
	// 	// try unmarshaling using v0.2.0 struct
	// 	if e := conf.Unmarshal(&v03); e != nil {
	// 		// error could not be resolved through backwards
	// 		// compatibility attempts
	// 		return err
	// 	}
	// 	c.MigratedFromV03 = true
	// 	return metricsConfigV03ToV120(v03, c)
	// }
	for _, r := range c.Readers {
		// ensure endpoint normalization occurs
		if r.Periodic != nil {
			if r.Periodic.Exporter.OTLPGrpc != nil {
				r.Periodic.Exporter.OTLPGrpc.Endpoint = normalizeEndpoint(*r.Periodic.Exporter.OTLPGrpc.Endpoint, r.Periodic.Exporter.OTLPGrpc.Tls.Insecure)
			}
			if r.Periodic.Exporter.OTLPHttp != nil {
				r.Periodic.Exporter.OTLPHttp.Endpoint = normalizeEndpoint(*r.Periodic.Exporter.OTLPHttp.Endpoint, new(false))
			}
		}
		// Apply Prometheus exporter defaults for fields not explicitly set.
		// When users explicitly configure the telemetry section (e.g. to
		// change the host), unset *bool fields default to nil which the
		// Prometheus exporter treats as false. This ensures the defaults
		// match the implicit/default configuration where these are true.
		//
		// This is necessary because specifying any metric readers in the
		// SDK config completely overwrites the list of readers, so the
		// Prometheus exporter is treated as a new config without any of
		// the defaults set in the createDefaultConfig function in
		// otelconftelemetry. We must re-apply those defaults here.
		if r.Pull != nil && r.Pull.Exporter.PrometheusDevelopment != nil {
			applyExperimentalPrometheusMetricExporterDefaults(r.Pull.Exporter.PrometheusDevelopment)
		}
	}
	return nil
}

var _ confmap.Marshaler = MetricsConfigV120{}

func (c MetricsConfigV120) Marshal(conf *confmap.Conf) error {
	if err := conf.Marshal(c); err != nil {
		return fmt.Errorf("otelconftelemetry: failed to marshal metrics configuration: %w", err)
	}

	// Redact header values the way configopaque would
	sm := conf.ToStringMap()
	redactHeaders(sm, "readers.*.periodic.exporter.otlp_http|otlp_grpc.headers.*.value")
	// Restore the stable `prometheus` key for the readers that used it, the
	// inverse of the rewrite done on unmarshal.
	if err := rewritePrometheusStable(sm, c.PrometheusStableKeyReaders); err != nil {
		return err
	}
	return conf.Marshal(sm)
}

// applyPrometheusDefaults sets default values for Prometheus exporter
// fields that were not explicitly provided in the configuration.
func applyExperimentalPrometheusMetricExporterDefaults(p *config.ExperimentalPrometheusMetricExporter) {
	if p.WithoutScopeInfo == nil {
		p.WithoutScopeInfo = new(true)
	}
	// TODO: fill in other defaults
}

type LogsConfigV120 struct {
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
	// Note that this setting only affects zap's own internal errors (e.g., failures
	// to write to the configured output). It does NOT route application error-level
	// log messages. To control where application logs go, use output_paths instead.
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

	// MigratedFromV02 is set when the config was transparently migrated from the v0.2.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV02 bool `mapstructure:"-"`
	// MigratedFromV03 is set when the config was transparently migrated from the v0.3.0
	// format. It is not part of the config schema and is used to emit a deprecation warning.
	MigratedFromV03 bool `mapstructure:"-"`
}

// ResourceConfigV120 represents the v0.3.0 resource configuration, with
// backward-compatible support for the legacy map format.
type ResourceConfigV120 struct {
	config.Resource `mapstructure:",squash"`

	DetectionDevelopment *config.ExperimentalResourceDetection `mapstructure:"detection/development,omitempty"`
	LegacyAttributes     map[string]any                        `mapstructure:",remain"`
}

var _ confmap.Validator = (*ResourceConfigV120)(nil)

func (cfg *ResourceConfigV120) Validate() error {
	// resource::attributes_list isn't currently supported by otelconf, so we have to put the default values under resource::attributes.
	// However, resource::attributes_list theoretically has lower priority than resource::attributes,
	// so if otelconf started supporting it, its values would be overridden by the defaults.
	// To avoid this surprising behavior, we explicitly disallow the use of resource::attributes_list for now.
	if cfg.AttributesList != nil {
		return errors.New("resource::attributes_list is not currently supported, please use resource::attributes")
	}

	// mapstructure only supports map[string]any for ",remain" fields, but we need it to be equivalent to map[string]*string
	for key, val := range cfg.LegacyAttributes {
		switch val.(type) {
		case nil, string:
		default:
			return fmt.Errorf("legacy resource attribute %q must be string or null", key)
		}
	}

	if len(cfg.Attributes) > 0 && len(cfg.LegacyAttributes) > 0 {
		return errors.New("resource::attributes cannot be used together with legacy inline resource attributes")
	}

	return nil
}

func (c *LogsConfigV120) Unmarshal(conf *confmap.Conf) error {
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
		if e := conf.Unmarshal(&v02); e == nil {
			c.MigratedFromV02 = true
			return logsConfigV02ToV120(v02, c)
		}
		v03 := LogsConfigV030{
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
		// try unmarshaling using v0.3.0 struct
		if e := conf.Unmarshal(&v03); e != nil {
			// error could not be resolved through backwards
			// compatibility attempts
			return err
		}
		c.MigratedFromV03 = true
		return logsConfigV03ToV120(v03, c)
	}
	// ensure endpoint normalization occurs
	for _, r := range c.Processors {
		if r.Batch != nil {
			if r.Batch.Exporter.OTLPGrpc != nil {
				r.Batch.Exporter.OTLPGrpc.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLPGrpc.Endpoint, r.Batch.Exporter.OTLPGrpc.Tls.Insecure)
			}
			if r.Batch.Exporter.OTLPHttp != nil {
				r.Batch.Exporter.OTLPHttp.Endpoint = normalizeEndpoint(*r.Batch.Exporter.OTLPHttp.Endpoint, new(false))
			}
		}
		if r.Simple != nil {
			if r.Simple.Exporter.OTLPGrpc != nil {
				r.Simple.Exporter.OTLPGrpc.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLPGrpc.Endpoint, r.Simple.Exporter.OTLPGrpc.Tls.Insecure)
			}
			if r.Simple.Exporter.OTLPHttp != nil {
				r.Simple.Exporter.OTLPHttp.Endpoint = normalizeEndpoint(*r.Simple.Exporter.OTLPHttp.Endpoint, new(false))
			}
		}
	}
	return nil
}

var _ confmap.Marshaler = LogsConfigV120{}

func (c LogsConfigV120) Marshal(conf *confmap.Conf) error {
	if err := conf.Marshal(c); err != nil {
		return fmt.Errorf("otelconftelemetry: failed to marshal logs configuration: %w", err)
	}

	// Redact header values the way configopaque would
	sm := conf.ToStringMap()
	redactHeaders(sm, "processors.*.simple|batch.exporter.otlp_http|otlp_grpc.headers.*.value")
	return conf.Marshal(sm)
}
