// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterqueue" // BaseExporter contains common fields between different exporter types.
	"go.opentelemetry.io/collector/pipeline"
)

// Option apply changes to BaseExporter.
type Option func(*BaseExporter) error

type BaseExporter struct {
	component.StartFunc
	component.ShutdownFunc

	Set exporter.Settings

	// Message for the user to be added with an export failure message.
	ExportFailureMessage string

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	QueueSender Sender[request.Request]
	RetrySender Sender[request.Request]

	firstSender Sender[request.Request]

	ConsumerOptions []consumer.Option

	timeoutCfg TimeoutConfig
	retryCfg   configretry.BackOffConfig

	queueBatchSettings queuebatch.Settings[request.Request]
	queueCfg           exporterqueue.Config
	batcherCfg         exporterbatcher.Config
}

func NewBaseExporter(set exporter.Settings, signal pipeline.Signal, pusher func(context.Context, request.Request) error, options ...Option) (*BaseExporter, error) {
	be := &BaseExporter{
		Set:        set,
		timeoutCfg: NewDefaultTimeoutConfig(),
	}

	for _, op := range options {
		if err := op(be); err != nil {
			return nil, err
		}
	}

	//nolint:staticcheck
	if be.batcherCfg.MinSizeItems != nil || be.batcherCfg.MaxSizeItems != nil {
		set.Logger.Warn("Using of deprecated fields `min_size_items` and `max_size_items`")
	}

	// Consumer Sender is always initialized.
	be.firstSender = newSender(pusher)

	// Next setup the timeout Sender since we want the timeout to control only the export functionality.
	// Only initialize if not explicitly disabled.
	if be.timeoutCfg.Timeout != 0 {
		be.firstSender = newTimeoutSender(be.timeoutCfg, be.firstSender)
	}

	if be.retryCfg.Enabled {
		be.RetrySender = newRetrySender(be.retryCfg, set, be.firstSender)
		be.firstSender = be.RetrySender
	}

	var err error
	be.firstSender, err = newObsReportSender(set, signal, be.firstSender)
	if err != nil {
		return nil, err
	}

	if be.batcherCfg.Enabled {
		// Batcher mutates the data.
		be.ConsumerOptions = append(be.ConsumerOptions, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	}

	if be.queueCfg.Enabled || be.batcherCfg.Enabled {
		qSet := exporterqueue.Settings[request.Request]{
			Signal:           signal,
			ExporterSettings: set,
			Encoding:         be.queueBatchSettings.Encoding,
		}
		be.QueueSender, err = NewQueueSender(qSet, be.queueCfg, be.batcherCfg, be.ExportFailureMessage, be.firstSender)
		if err != nil {
			return nil, err
		}
		be.firstSender = be.QueueSender
	}

	return be, nil
}

// Send sends the request using the first sender in the chain.
func (be *BaseExporter) Send(ctx context.Context, req request.Request) error {
	// Have to read the number of items before sending the request since the request can
	// be modified by the downstream components like the batcher.
	itemsCount := req.ItemsCount()
	err := be.firstSender.Send(ctx, req)
	if err != nil {
		be.Set.Logger.Error("Exporting failed. Rejecting data."+be.ExportFailureMessage,
			zap.Error(err), zap.Int("rejected_items", itemsCount))
	}
	return err
}

func (be *BaseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the wrapped exporter.
	if err := be.StartFunc.Start(ctx, host); err != nil {
		return err
	}

	// Last start the queueSender.
	if be.QueueSender != nil {
		return be.QueueSender.Start(ctx, host)
	}

	return nil
}

func (be *BaseExporter) Shutdown(ctx context.Context) error {
	var err error

	// First shutdown the retry sender, so the queue sender can flush the queue without retries.
	if be.RetrySender != nil {
		err = multierr.Append(err, be.RetrySender.Shutdown(ctx))
	}

	// Then shutdown the queue sender.
	if be.QueueSender != nil {
		err = multierr.Append(err, be.QueueSender.Shutdown(ctx))
	}

	// Last shutdown the wrapped exporter itself.
	return multierr.Append(err, be.ShutdownFunc.Shutdown(ctx))
}

// WithStart overrides the default Start function for an exporter.
// The default start function does nothing and always returns nil.
func WithStart(start component.StartFunc) Option {
	return func(o *BaseExporter) error {
		o.StartFunc = start
		return nil
	}
}

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return func(o *BaseExporter) error {
		o.ShutdownFunc = shutdown
		return nil
	}
}

// WithTimeout overrides the default TimeoutConfig for an exporter.
// The default TimeoutConfig is 5 seconds.
func WithTimeout(timeoutConfig TimeoutConfig) Option {
	return func(o *BaseExporter) error {
		o.timeoutCfg = timeoutConfig
		return nil
	}
}

// WithRetry overrides the default configretry.BackOffConfig for an exporter.
// The default configretry.BackOffConfig is to disable retries.
func WithRetry(config configretry.BackOffConfig) Option {
	return func(o *BaseExporter) error {
		if !config.Enabled {
			o.ExportFailureMessage += " Try enabling retry_on_failure config option to retry on retryable errors."
			return nil
		}
		o.retryCfg = config
		return nil
	}
}

// WithQueue overrides the default QueueConfig for an exporter.
// The default QueueConfig is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(cfg exporterqueue.Config) Option {
	return func(o *BaseExporter) error {
		if o.queueBatchSettings.Encoding == nil {
			return errors.New("WithQueue option is not available for the new request exporters, use WithQueueBatch instead")
		}
		return WithQueueBatch(cfg, o.queueBatchSettings)(o)
	}
}

// WithQueueBatch enables queueing for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithQueueBatch(cfg exporterqueue.Config, set queuebatch.Settings[request.Request]) Option {
	return func(o *BaseExporter) error {
		if !cfg.Enabled {
			o.ExportFailureMessage += " Try enabling sending_queue to survive temporary failures."
			return nil
		}
		if cfg.StorageID != nil && set.Encoding == nil {
			return errors.New("`QueueBatchSettings.Encoding` must not be nil when persistent queue is enabled")
		}
		o.queueBatchSettings = set
		o.queueCfg = cfg
		return nil
	}
}

// WithCapabilities overrides the default Capabilities() function for a Consumer.
// The default is non-mutable data.
// TODO: Verify if we can change the default to be mutable as we do for processors.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *BaseExporter) error {
		o.ConsumerOptions = append(o.ConsumerOptions, consumer.WithCapabilities(capabilities))
		return nil
	}
}

// WithBatcher enables batching for an exporter based on custom request types.
// For now, it can be used only with the New[Traces|Metrics|Logs]RequestExporter exporter helpers and
// WithRequestBatchFuncs provided.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithBatcher(cfg exporterbatcher.Config) Option {
	return func(o *BaseExporter) error {
		o.batcherCfg = cfg
		return nil
	}
}

// WithQueueBatchSettings is used to set the queuebatch.Settings for the new request based exporter helper.
// It must be provided as the first option when creating a new exporter helper.
func WithQueueBatchSettings(set queuebatch.Settings[request.Request]) Option {
	return func(o *BaseExporter) error {
		o.queueBatchSettings = set
		return nil
	}
}
