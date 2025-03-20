// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
)

// QueueConfig defines configuration for queueing batches before sending to the consumerSender.
type QueueConfig = exporterqueue.Config

// Deprecated: [v0.123.0] use WithQueueBatch.
func WithRequestQueue(cfg exporterqueue.Config, encoding exporterqueue.Encoding[Request]) Option {
	return WithQueueBatch(cfg, QueueBatchSettings{Encoding: encoding})
}

// QueueBatchConfig defines configuration for queueing and batching for the exporter.
type QueueBatchConfig = exporterqueue.Config

// QueueBatchSettings are settings for the QueueBatch component.
// They include things line Encoding to be used with persistent queue, or the available Sizers, etc.
type QueueBatchSettings = queuebatch.Settings[Request]

// WithQueueBatch enables queueing and batching for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithQueueBatch(cfg QueueBatchConfig, set QueueBatchSettings) Option {
	return internal.WithQueueBatch(cfg, set)
}

// NewDefaultQueueConfig returns the default config for QueueConfig.
func NewDefaultQueueConfig() QueueConfig {
	return exporterqueue.Config{
		Enabled:      true,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize: 1_000,
		Blocking:  false,
	}
}
