// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelconftelemetry // import "go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"

import (
	"errors"

	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/service/internal/resource"
	"go.opentelemetry.io/collector/service/telemetry/internal/migration"
)

// Config defines the configurable settings for service telemetry.
type Config struct {
	Logs    LogsConfig    `mapstructure:"logs"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Traces  TracesConfig  `mapstructure:"traces,omitempty"`

	// Resource follows the declarative configuration schema. Static attributes, OTEL_RESOURCE_ATTRIBUTES-style lists,
	// and detection settings can all be configured under this block. Besides the schema-defined detectors (container,
	// host, process, service) the collector also exposes additional detectors such as env, aws/*, gcp, and azure.
	Resource resource.Config `mapstructure:"resource,omitempty"`
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
