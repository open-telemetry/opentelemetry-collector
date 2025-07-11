// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelineprocessor // import "go.opentelemetry.io/collector/processor/pipelineprocessor"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for the pipeline processor.
type Config struct {
	// TimeoutConfig contains the timeout configuration for the processor.
	exporterhelper.TimeoutConfig `mapstructure:",squash"`

	// QueueConfig contains the queue configuration for the processor.
	QueueConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`

	// RetryConfig contains the retry configuration for the processor.
	RetryConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// prevent unkeyed literal initialization
	_ struct{}
}

var _ component.Config = (*Config)(nil)

// Validate checks if the processor configuration is valid
func (cfg *Config) Validate() error {
	if err := cfg.TimeoutConfig.Validate(); err != nil {
		return err
	}
	if err := cfg.QueueConfig.Validate(); err != nil {
		return err
	}
	if err := cfg.RetryConfig.Validate(); err != nil {
		return err
	}
	return nil
}
