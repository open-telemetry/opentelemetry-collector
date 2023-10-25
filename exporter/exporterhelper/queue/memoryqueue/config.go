// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package memoryqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/queue/memoryqueue"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
)

// Config defines configuration for queueing requests before exporting using a memory storage.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Config struct {
	queue.Config `mapstructure:",squash"`
}

// NewDefaultConfig returns the default Config.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewDefaultConfig() Config {
	return Config{
		Config: queue.NewDefaultConfig(),
	}
}
