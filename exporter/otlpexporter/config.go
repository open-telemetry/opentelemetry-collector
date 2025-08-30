// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlpexporter // import "go.opentelemetry.io/collector/exporter/otlpexporter"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for OTLP exporter.
type Config struct {
	TimeoutConfig exporterhelper.TimeoutConfig    `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	QueueConfig   exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`
	RetryConfig   configretry.BackOffConfig       `mapstructure:"retry_on_failure"`
	ClientConfig  configgrpc.ClientConfig         `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// prevent unkeyed literal initialization
	_ struct{}
}

func (c *Config) Validate() error {
	return c.ClientConfig.Validate()
}

var _ component.Config = (*Config)(nil)
