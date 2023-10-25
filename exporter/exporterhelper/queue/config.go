// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue"

import "errors"

// Config defines configuration for queueing requests before exporting.
// It's supposed to be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Config struct {
	// Enabled indicates whether to not enqueue batches before exporting.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	// This field is left for backward compatibility with QueueSettings.
	// Later, it will be replaced with size fields specified explicitly in terms of items or batches.
	QueueSize int `mapstructure:"queue_size"`
}

// NewDefaultConfig returns the default QueueConfig.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewDefaultConfig() Config {
	return Config{
		Enabled:      true,
		NumConsumers: 10,
		QueueSize:    1000,
	}
}

// Validate checks if the QueueSettings configuration is valid
func (qCfg *Config) Validate() error {
	if !qCfg.Enabled {
		return nil
	}
	if qCfg.NumConsumers <= 0 {
		return errors.New("number of consumers must be positive")
	}
	if qCfg.QueueSize <= 0 {
		return errors.New("queue size must be positive")
	}
	return nil
}
