// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterqueue" //nolint // Deprecated package only used in deprecated functions
	internalExporterQueue "go.opentelemetry.io/collector/exporter/internal/exporterqueue"
)

// Option apply changes to BaseExporter.
type Option = internal.Option

// WithStart overrides the default Start function for an exporter.
// The default start function does nothing and always returns nil.
func WithStart(start component.StartFunc) Option {
	return internal.WithStart(start)
}

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return internal.WithShutdown(shutdown)
}

// WithTimeout overrides the default TimeoutConfig for an exporter.
// The default TimeoutConfig is 5 seconds.
func WithTimeout(timeoutConfig TimeoutConfig) Option {
	return internal.WithTimeout(timeoutConfig)
}

// WithRetry overrides the default configretry.BackOffConfig for an exporter.
// The default configretry.BackOffConfig is to disable retries.
func WithRetry(config configretry.BackOffConfig) Option {
	return internal.WithRetry(config)
}

// WithQueue overrides the default QueueConfig for an exporter.
// The default QueueConfig is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config internal.QueueConfig) Option {
	return internal.WithQueue(config)
}

// WithRequestQueue enables queueing for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
//
// Deprecated: [v0.112.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so that we don't remove it.
func WithRequestQueue(cfg exporterqueue.Config, queueFactory exporterqueue.Factory[Request]) Option {
	// Ugly code to convert between external and internal defined types
	internalFactory := func(ctx context.Context, s internalExporterQueue.Settings, c internalExporterQueue.Config) internalExporterQueue.Queue[Request] {
		return queueFactory(ctx, s, c)
	}
	return internal.WithRequestQueue(cfg, internalFactory)
}

// WithCapabilities overrides the default Capabilities() function for a Consumer.
// The default is non-mutable data.
// TODO: Verify if we can change the default to be mutable as we do for processors.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return internal.WithCapabilities(capabilities)
}

// BatcherOption apply changes to batcher sender.
type BatcherOption = internal.BatcherOption

// WithRequestBatchFuncs sets the functions for merging and splitting batches for an exporter built for custom request types.
//
// Deprecated: [v0.112.0] If you use this API, please comment on
// https://github.com/open-telemetry/opentelemetry-collector/issues/11142 so that we don't remove it.
func WithRequestBatchFuncs(mf exporterbatcher.BatchMergeFunc[Request], msf exporterbatcher.BatchMergeSplitFunc[Request]) BatcherOption {
	return internal.WithRequestBatchFuncs(mf, msf)
}

// WithBatcher enables batching for an exporter based on custom request types.
// For now, it can be used only with the New[Traces|Metrics|Logs]RequestExporter exporter helpers and
// WithRequestBatchFuncs provided.
//
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithBatcher(cfg exporterbatcher.Config, opts ...BatcherOption) Option {
	return internal.WithBatcher(cfg, opts...)
}
