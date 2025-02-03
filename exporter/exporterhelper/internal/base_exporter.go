// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterqueue" // BaseExporter contains common fields between different exporter types.
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pipeline"
)

var usePullingBasedExporterQueueBatcher = featuregate.GlobalRegistry().MustRegister(
	"exporter.UsePullingBasedExporterQueueBatcher",
	featuregate.StageAlpha,
	featuregate.WithRegisterFromVersion("v0.115.0"),
	featuregate.WithRegisterDescription("if set to true, turns on the pulling-based exporter queue bathcer"),
)

type ObsrepSenderFactory = func(obsrep *ObsReport, next Sender[internal.Request]) Sender[internal.Request]

// Option apply changes to BaseExporter.
type Option func(*BaseExporter) error

type BaseExporter struct {
	component.StartFunc
	component.ShutdownFunc

	Marshaler   exporterqueue.Marshaler[internal.Request]
	Unmarshaler exporterqueue.Unmarshaler[internal.Request]

	Set exporter.Settings

	// Message for the user to be added with an export failure message.
	ExportFailureMessage string

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	BatchSender  Sender[internal.Request]
	QueueSender  Sender[internal.Request]
	ObsrepSender Sender[internal.Request]
	RetrySender  Sender[internal.Request]

	firstSender Sender[internal.Request]

	ConsumerOptions []consumer.Option

	timeoutCfg   TimeoutConfig
	retryCfg     configretry.BackOffConfig
	queueFactory exporterqueue.Factory[internal.Request]
	queueCfg     exporterqueue.Config
	batcherCfg   exporterbatcher.Config
}

func NewBaseExporter(set exporter.Settings, signal pipeline.Signal, osf ObsrepSenderFactory, options ...Option) (*BaseExporter, error) {
	obsReport, err := NewObsReport(ObsReportSettings{ExporterSettings: set, Signal: signal})
	if err != nil {
		return nil, err
	}

	be := &BaseExporter{
		timeoutCfg: NewDefaultTimeoutConfig(),
		Set:        set,
	}

	for _, op := range options {
		if err = op(be); err != nil {
			return nil, err
		}
	}

	// TimeoutSender is always initialized.
	be.firstSender = &TimeoutSender{cfg: be.timeoutCfg}
	if be.retryCfg.Enabled {
		be.RetrySender = newRetrySender(be.retryCfg, set, be.firstSender)
		be.firstSender = be.RetrySender
	}

	be.ObsrepSender = osf(obsReport, be.firstSender)
	be.firstSender = be.ObsrepSender

	if be.batcherCfg.Enabled {
		// Batcher mutates the data.
		be.ConsumerOptions = append(be.ConsumerOptions, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	}

	if be.batcherCfg.Enabled && !(usePullingBasedExporterQueueBatcher.IsEnabled() && be.queueCfg.Enabled) {
		concurrencyLimit := int64(0)
		if be.queueCfg.Enabled {
			concurrencyLimit = int64(be.queueCfg.NumConsumers)
		}
		be.BatchSender = NewBatchSender(be.batcherCfg, set, concurrencyLimit, be.firstSender)
		be.firstSender = be.BatchSender
	}

	if be.queueCfg.Enabled {
		qSet := exporterqueue.Settings{
			Signal:           signal,
			ExporterSettings: set,
		}
		be.QueueSender, err = NewQueueSender(be.queueFactory, qSet, be.queueCfg, be.batcherCfg, be.ExportFailureMessage, be.firstSender)
		if err != nil {
			return nil, err
		}
		be.firstSender = be.QueueSender
	}

	return be, nil
}

// Send sends the request using the first sender in the chain.
func (be *BaseExporter) Send(ctx context.Context, req internal.Request) error {
	err := be.firstSender.Send(ctx, req)
	if err != nil {
		be.Set.Logger.Error("Exporting failed. Rejecting data."+be.ExportFailureMessage,
			zap.Error(err), zap.Int("rejected_items", req.ItemsCount()))
	}
	return err
}

func (be *BaseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the wrapped exporter.
	if err := be.StartFunc.Start(ctx, host); err != nil {
		return err
	}

	if be.BatchSender != nil {
		// If no error then start the BatchSender.
		if err := be.BatchSender.Start(ctx, host); err != nil {
			return err
		}
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

	// Then shutdown the batch sender
	if be.BatchSender != nil {
		err = multierr.Append(err, be.BatchSender.Shutdown(ctx))
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
func WithQueue(config QueueConfig) Option {
	return func(o *BaseExporter) error {
		if o.Marshaler == nil || o.Unmarshaler == nil {
			return errors.New("WithQueue option is not available for the new request exporters, use WithRequestQueue instead")
		}
		if !config.Enabled {
			o.ExportFailureMessage += " Try enabling sending_queue to survive temporary failures."
			return nil
		}
		o.queueCfg = exporterqueue.Config{
			Enabled:      config.Enabled,
			NumConsumers: config.NumConsumers,
			QueueSize:    config.QueueSize,
			Blocking:     config.Blocking,
		}
		o.queueFactory = exporterqueue.NewPersistentQueueFactory[internal.Request](config.StorageID, exporterqueue.PersistentQueueSettings[internal.Request]{
			Marshaler:   o.Marshaler,
			Unmarshaler: o.Unmarshaler,
		})
		return nil
	}
}

// WithRequestQueue enables queueing for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithRequestQueue(cfg exporterqueue.Config, queueFactory exporterqueue.Factory[internal.Request]) Option {
	return func(o *BaseExporter) error {
		if o.Marshaler != nil || o.Unmarshaler != nil {
			return errors.New("WithRequestQueue option must be used with the new request exporters only, use WithQueue instead")
		}
		if !cfg.Enabled {
			o.ExportFailureMessage += " Try enabling sending_queue to survive temporary failures."
			return nil
		}
		o.queueCfg = cfg
		o.queueFactory = queueFactory
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

// WithMarshaler is used to set the request marshaler for the new exporter helper.
// It must be provided as the first option when creating a new exporter helper.
func WithMarshaler(marshaler exporterqueue.Marshaler[internal.Request]) Option {
	return func(o *BaseExporter) error {
		o.Marshaler = marshaler
		return nil
	}
}

// WithUnmarshaler is used to set the request unmarshaler for the new exporter helper.
// It must be provided as the first option when creating a new exporter helper.
func WithUnmarshaler(unmarshaler exporterqueue.Unmarshaler[internal.Request]) Option {
	return func(o *BaseExporter) error {
		o.Unmarshaler = unmarshaler
		return nil
	}
}

func CheckStatus(t *testing.T, sd sdktrace.ReadOnlySpan, err error) {
	if err != nil {
		require.Equal(t, codes.Error, sd.Status().Code, "SpanData %v", sd)
		require.EqualError(t, err, sd.Status().Description, "SpanData %v", sd)
	} else {
		require.Equal(t, codes.Unset, sd.Status().Code, "SpanData %v", sd)
	}
}
