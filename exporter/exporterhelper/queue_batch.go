// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
)

// Deprecated: [v0.123.0] use QueueBatchConfig.
type QueueConfig = QueueBatchConfig

// Deprecated: [v0.123.0] use WithQueueBatch.
func WithRequestQueue(cfg QueueBatchConfig, encoding QueueBatchEncoding[Request]) Option {
	return WithQueueBatch(cfg, QueueBatchSettings{Encoding: encoding})
}

// WithQueue overrides the default QueueBatchConfig for an exporter.
// The default QueueBatchConfig is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config QueueBatchConfig) Option {
	return internal.WithQueue(config)
}

// WithBatcher enables batching for an exporter based on custom request types.
// For now, it can be used only with the New[Traces|Metrics|Logs]RequestExporter exporter helpers and
// WithRequestBatchFuncs provided.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithBatcher(cfg BatcherConfig) Option {
	return internal.WithBatcher(cfg)
}

// QueueBatchConfig defines configuration for queueing and batching for the exporter.
type QueueBatchConfig = queuebatch.Config

// QueueBatchEncoding defines the encoding to be used if persistent queue is configured.
// Duplicate definition with queuebatch.Encoding since aliasing generics is not supported by default.
type QueueBatchEncoding[T any] interface {
	// Marshal is a function that can marshal a request into bytes.
	Marshal(T) ([]byte, error)

	// Unmarshal is a function that can unmarshal bytes into a request.
	Unmarshal([]byte) (T, error)
}

var ErrQueueIsFull = queuebatch.ErrQueueIsFull

// QueueBatchSettings are settings for the QueueBatch component.
// They include things line Encoding to be used with persistent queue, or the available Sizers, etc.
type QueueBatchSettings = internal.QueueBatchSettings[Request]

// WithQueueBatch enables queueing and batching for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithQueueBatch(cfg QueueBatchConfig, set QueueBatchSettings) Option {
	return internal.WithQueueBatch(cfg, set)
}

// NewDefaultQueueConfig returns the default config for QueueBatchConfig.
// By default, the queue stores 1000 items of telemetry and is non-blocking when full.
var NewDefaultQueueConfig = internal.NewDefaultQueueConfig

// BatcherConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatcherConfig = internal.BatcherConfig

// SizeConfig sets the size limits for a batch.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type SizeConfig = internal.SizeConfig

var NewDefaultBatcherConfig = internal.NewDefaultBatcherConfig
