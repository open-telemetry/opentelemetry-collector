// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "go.opentelemetry.io/collector/service/telemetry"

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

var _ confmap.Unmarshaler = (*Config)(nil)

var disableAddressFieldForInternalTelemetryFeatureGate = featuregate.GlobalRegistry().MustRegister(
	"telemetry.disableAddressFieldForInternalTelemetry",
	featuregate.StageBeta,
	featuregate.WithRegisterFromVersion("v0.111.0"),
	featuregate.WithRegisterToVersion("v0.123.0"),
	featuregate.WithRegisterDescription("controls whether the deprecated address field for internal telemetry is still supported"))

// Config defines the configurable settings for service telemetry.
type Config struct {
	Logs    LogsConfig    `mapstructure:"logs"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Traces  TracesConfig  `mapstructure:"traces,omitempty"`

	// Resource specifies user-defined attributes to include with all emitted telemetry.
	// Note that some attributes are added automatically (e.g. service.version) even
	// if they are not specified here. In order to suppress such attributes the
	// attribute must be specified in this map with null YAML value (nil string pointer).
	Resource map[string]*string `mapstructure:"resource,omitempty"`
}

// LogsConfig defines the configurable settings for service telemetry logs.
// This MUST be compatible with zap.Config. Cannot use directly zap.Config because
// the collector uses mapstructure and not yaml tags.
type LogsConfig = migration.LogsConfigV030

// LogsSamplingConfig sets a sampling strategy for the logger. Sampling caps the
// global CPU and I/O load that logging puts on your process while attempting
// to preserve a representative subset of your logs.
type LogsSamplingConfig = migration.LogsSamplingConfig

// MetricsConfig exposes the common Telemetry configuration for one component.
// Experimental: *NOTE* this structure is subject to change or removal in the future.
type MetricsConfig = migration.MetricsConfigV030

// TracesConfig exposes the common Telemetry configuration for collector's internal spans.
// Experimental: *NOTE* this structure is subject to change or removal in the future.
type TracesConfig = migration.TracesConfigV030

func (c *Config) Unmarshal(conf *confmap.Conf) error {
	if err := conf.Unmarshal(c); err != nil {
		return err
	}

	// If the support for "metrics::address" is disabled, nothing to do.
	// TODO: when this gate is marked stable remove the whole Unmarshal definition.
	if disableAddressFieldForInternalTelemetryFeatureGate.IsEnabled() {
		return nil
	}

	if len(c.Metrics.Address) != 0 { //nolint:staticcheck // SA1019
		host, port, err := net.SplitHostPort(c.Metrics.Address) //nolint:staticcheck // SA1019
		if err != nil {
			return fmt.Errorf("failing to parse metrics address %q: %w", c.Metrics.Address, err) //nolint:staticcheck // SA1019
		}
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("failing to extract the port from the metrics address %q: %w", c.Metrics.Address, err) //nolint:staticcheck // SA1019
		}

		// User did not overwrite readers, so we will remove the default configured reader.
		if !conf.IsSet("metrics::readers") {
			c.Metrics.Readers = nil
		}

		c.Metrics.Readers = append(c.Metrics.Readers, config.MetricReader{
			Pull: &config.PullMetricReader{
				Exporter: config.PullMetricExporter{
					Prometheus: &config.Prometheus{
						Host:              &host,
						Port:              &portInt,
						WithoutScopeInfo:  newPtr(true),
						WithoutUnits:      newPtr(true),
						WithoutTypeSuffix: newPtr(true),
						WithResourceConstantLabels: &config.IncludeExclude{
							Included: []string{},
						},
					},
				},
			},
		})
	}

	return nil
}

// Validate checks whether the current configuration is valid
func (c *Config) Validate() error {
	// Check when service telemetry metric level is not none, the metrics readers should not be empty
	if c.Metrics.Level != configtelemetry.LevelNone && len(c.Metrics.Readers) == 0 {
		return errors.New("collector telemetry metrics reader should exist when metric level is not none")
	}

	if c.Metrics.Views != nil && c.Metrics.Level != configtelemetry.LevelDetailed {
		return errors.New("service::telemetry::metrics::views can only be set when service::telemetry::metrics::level is detailed")
	}

	return nil
}
