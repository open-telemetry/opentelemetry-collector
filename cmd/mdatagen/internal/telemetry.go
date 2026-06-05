// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

type Telemetry struct {
	Metrics map[MetricName]Metric `mapstructure:"metrics"`

	// AllowNameSubstitution opts the generated TelemetryBuilder into a
	// WithMetricNamePrefixReplacement option. When enabled, callers may
	// request that every metric name produced by this builder be transformed
	// by replacing a configured prefix with a different one at construction
	// time. Intended for code that is reused across component kinds (e.g.
	// exporterhelper reused as a processor), so the same TelemetryBuilder
	// can emit either "otelcol_exporter_*" or "otelcol_processor_*" names.
	AllowNameSubstitution bool `mapstructure:"allow_name_substitution"`
}
