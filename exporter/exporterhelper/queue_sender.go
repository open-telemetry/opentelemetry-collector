// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

// Deprecated: [v0.110.0] Use QueueConfig instead.
type QueueSettings = internal.QueueConfig

// QueueConfig defines configuration for queueing batches before sending to the consumerSender.
type QueueConfig = internal.QueueConfig

// Deprecated: [v0.110.0] Use NewDefaultQueueConfig instead.
func NewDefaultQueueSettings() QueueSettings {
	return internal.NewDefaultQueueConfig()
}

// NewDefaultQueueConfig returns the default config for QueueConfig.
func NewDefaultQueueConfig() QueueConfig {
	return internal.NewDefaultQueueConfig()
}
