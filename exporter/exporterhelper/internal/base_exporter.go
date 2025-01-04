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

type ObsrepSenderFactory = func(obsrep *ObsReport) Sender[internal.Request]

// Option apply changes to BaseExporter.
type Option func(*BaseExporter) error

type BaseExporter struct {
	component.StartFunc
	component.ShutdownFunc

	Marshaler   exporterqueue.Marshaler[internal.Request]
	Unmarshaler exporterqueue.Unmarshaler[internal.Request]

	Set    exporter.Settings
	Obsrep *ObsReport

	// Message for the user to be added with an export failure message.
	ExportFailureMessage string

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	BatchSender   Sender[internal.Request]
	QueueSender   Sender[internal.Request]
	ObsrepSender  Sender[internal.Request]
	RetrySender   Sender[internal.Request]
	TimeoutSender *TimeoutSender // TimeoutSender is always initialized.

	ConsumerOptions []consumer.Option

	queueCfg     exporterqueue.Config
	queueFactory exporterqueue.Factory[internal.Request]
	BatcherCfg   exporterbatcher.Config
}

func NewBaseExporter(set exporter.Settings, signal pipeline.Signal, osf ObsrepSenderFactory, options ...Option) (*BaseExporter, error) {
	obsReport, err := NewExporter(ObsReportSettings{ExporterID: set.ID, ExporterCreateSettings: set, Signal: signal})
	if err != nil {
		return nil, err
	}

	be := &BaseExporter{
		BatchSender:   &BaseSender[internal.Request]{},
		QueueSender:   &BaseSender[internal.Request]{},
		ObsrepSender:  osf(obsReport),
		RetrySender:   &BaseSender[internal.Request]{},
		TimeoutSender: &TimeoutSender{cfg: NewDefaultTimeoutConfig()},

		Set:    set,
		Obsrep: obsReport,
	}

	for _, op := range options {
		err = multierr.Append(err, op(be))
	}
	if err != nil {
		return nil, err
	}

	if be.queueCfg.Enabled {
		q := be.queueFactory(
			context.Background(),
			exporterqueue.Settings{
				Signal:           signal,
				ExporterSettings: be.Set,
			},
			be.queueCfg)
		be.QueueSender = NewQueueSender(q, be.Set, be.queueCfg.NumConsumers, be.ExportFailureMessage, be.Obsrep, be.BatcherCfg)
	}

	if !usePullingBasedExporterQueueBatcher.IsEnabled() && be.BatcherCfg.Enabled ||
		usePullingBasedExporterQueueBatcher.IsEnabled() && be.BatcherCfg.Enabled && !be.queueCfg.Enabled {
		bs := NewBatchSender(be.BatcherCfg, be.Set)
		be.BatchSender = bs
	}

	be.connectSenders()

	if bs, ok := be.BatchSender.(*BatchSender); ok {
		// If queue sender is enabled assign to the batch sender the same number of workers.
		if qs, ok := be.QueueSender.(*QueueSender); ok {
			bs.concurrencyLimit = int64(qs.numConsumers)
		}
		// Batcher sender mutates the data.
		be.ConsumerOptions = append(be.ConsumerOptions, consumer.WithCapabilities(consumer.Capabilities{MutatesData: true}))
	}

	return be, nil
}

// send sends the request using the first sender in the chain.
func (be *BaseExporter) Send(ctx context.Context, req internal.Request) error {
	err := be.QueueSender.Send(ctx, req)
	if err != nil {
		be.Set.Logger.Error("Exporting failed. Rejecting data."+be.ExportFailureMessage,
			zap.Error(err), zap.Int("rejected_items", req.ItemsCount()))
	}
	return err
}

// connectSenders connects the senders in the predefined order.
func (be *BaseExporter) connectSenders() {
	be.QueueSender.SetNextSender(be.BatchSender)
	be.BatchSender.SetNextSender(be.ObsrepSender)
	be.ObsrepSender.SetNextSender(be.RetrySender)
	be.RetrySender.SetNextSender(be.TimeoutSender)
}

func (be *BaseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the wrapped exporter.
	if err := be.StartFunc.Start(ctx, host); err != nil {
		return err
	}

	// If no error then start the BatchSender.
	if err := be.BatchSender.Start(ctx, host); err != nil {
		return err
	}

	// Last start the queueSender.
	return be.QueueSender.Start(ctx, host)
}

func (be *BaseExporter) Shutdown(ctx context.Context) error {
	return multierr.Combine(
		// First shutdown the retry sender, so the queue sender can flush the queue without retries.
		be.RetrySender.Shutdown(ctx),
		// Then shutdown the batch sender
		be.BatchSender.Shutdown(ctx),
		// Then shutdown the queue sender.
		be.QueueSender.Shutdown(ctx),
		// Last shutdown the wrapped exporter itself.
		be.ShutdownFunc.Shutdown(ctx))
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
		o.TimeoutSender.cfg = timeoutConfig
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
		o.RetrySender = newRetrySender(config, o.Set)
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
		o.BatcherCfg = cfg
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

// withUnmarshaler is used to set the request unmarshaler for the new exporter helper.
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
