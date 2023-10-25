// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue/persistentqueue"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
)

// Config defines configuration for queueing requests before exporting using a persistent storage.
// It's supposed to be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter and will replace
// QueueSettings in the future.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Config struct {
	queue.Config `mapstructure:",squash"`
	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID *component.ID `mapstructure:"storage"`
}

// NewDefaultConfig returns the default Config.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewDefaultConfig() Config {
	return Config{
		Config: queue.NewDefaultConfig(),
	}
}
