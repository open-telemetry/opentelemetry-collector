// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterqueue // import "go.opentelemetry.io/collector/exporter/exporterqueue"

import (
	"go.opentelemetry.io/collector/exporter/internal/exporterqueue"
)

// Config defines configuration for queueing requests before exporting.
// It's supposed to be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type Config = exporterqueue.Config

// NewDefaultConfig returns the default Config.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
func NewDefaultConfig() Config {
	return exporterqueue.NewDefaultConfig()
}

// PersistentQueueConfig defines configuration for queueing requests in a persistent storage.
// The struct is provided to be added in the exporter configuration as one struct under the "sending_queue" key.
// The exporter helper Go interface requires the fields to be provided separately to WithRequestQueue and
// NewPersistentQueueFactory.
//
// Deprecated: [v0.111.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so we don't remove it.
type PersistentQueueConfig = exporterqueue.PersistentQueueConfig
