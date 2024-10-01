// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"go.opentelemetry.io/collector/exporter/internal/exporterqueue"
)

// Config defines configuration for queueing requests before exporting.
// It's supposed to be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type Config = exporterqueue.Config

// NewDefaultConfig returns the default Config.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewDefaultConfig() Config {
	return exporterqueue.NewDefaultConfig()
}

// PersistentQueueConfig defines configuration for queueing requests in a persistent storage.
// The struct is provided to be added in the exporter configuration as one struct under the "sending_queue" key.
// The exporter helper Go interface requires the fields to be provided separately to WithRequestQueue and
// NewPersistentQueueFactory.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type PersistentQueueConfig = exporterqueue.PersistentQueueConfig
