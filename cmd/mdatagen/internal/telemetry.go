// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/cmd/mdatagen/internal"

type Telemetry struct {
	Metrics map[MetricName]Metric `mapstructure:"metrics"`

	// AllowNameVariation opts the generated TelemetryBuilder into a
	// variation-aware naming scheme. When enabled, NewTelemetryBuilder
	// takes a required positional variation string and every metric name
	// is emitted as "otelcol_<variation>_<key>". This is the mechanism
	// used by code reused across component kinds: the exporter call site
	// passes "exporter", the processor call site passes "processor", and
	// no other API surface (such as a functional option) is exposed.
	AllowNameVariation bool `mapstructure:"allow_name_variation"`
}
