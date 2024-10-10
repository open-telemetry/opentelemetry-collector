// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

// QueueConfig defines configuration for queueing batches before sending to the consumerSender.
type QueueConfig = internal.QueueConfig

// NewDefaultQueueConfig returns the default config for QueueConfig.
func NewDefaultQueueConfig() QueueConfig {
	return internal.NewDefaultQueueConfig()
}
