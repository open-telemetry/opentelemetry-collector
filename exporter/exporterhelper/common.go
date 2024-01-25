// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
)

// requestSender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).
type requestSender interface {
	component.Component
	send(context.Context, Request) error
	setNextSender(nextSender requestSender)
}

type baseRequestSender struct {
	component.StartFunc
	component.ShutdownFunc
	nextSender requestSender
}

var _ requestSender = (*baseRequestSender)(nil)

func (b *baseRequestSender) send(ctx context.Context, req Request) error {
	return b.nextSender.send(ctx, req)
}

func (b *baseRequestSender) setNextSender(nextSender requestSender) {
	b.nextSender = nextSender
}

type obsrepSenderFactory func(obsrep *ObsReport) requestSender

// Option apply changes to baseExporter.
type Option func(*baseExporter)

// WithStart overrides the default Start function for an exporter.
// The default start function does nothing and always returns nil.
func WithStart(start component.StartFunc) Option {
	return func(o *baseExporter) {
		o.StartFunc = start
	}
}

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return func(o *baseExporter) {
		o.ShutdownFunc = shutdown
	}
}

// WithTimeout overrides the default TimeoutSettings for an exporter.
// The default TimeoutSettings is 5 seconds.
func WithTimeout(timeoutSettings TimeoutSettings) Option {
	return func(o *baseExporter) {
		o.timeoutSender.cfg = timeoutSettings
	}
}

// WithRetry overrides the default configretry.BackOffConfig for an exporter.
// The default configretry.BackOffConfig is to disable retries.
func WithRetry(config configretry.BackOffConfig) Option {
	return func(o *baseExporter) {
		if !config.Enabled {
			o.exportFailureMessage += " Try enabling retry_on_failure config option to retry on retryable errors."
			return
		}
		o.retrySender = newRetrySender(config, o.set)
	}
}

// WithQueue overrides the default QueueSettings for an exporter.
// The default QueueSettings is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config QueueSettings) Option {
	return func(o *baseExporter) {
		if o.requestExporter {
			panic("queueing is not available for the new request exporters yet")
		}
		if !config.Enabled {
			o.exportFailureMessage += " Try enabling sending_queue to survive temporary failures."
			return
		}
		consumeErrHandler := func(err error, req Request) {
			o.set.Logger.Error("Exporting failed. Dropping data."+o.exportFailureMessage,
				zap.Error(err), zap.Int("dropped_items", req.ItemsCount()))
		}
		o.queueSender = newQueueSender(config, o.set, o.signal, o.marshaler, o.unmarshaler, consumeErrHandler)
	}
}

// WithCapabilities overrides the default Capabilities() function for a Consumer.
// The default is non-mutable data.
// TODO: Verify if we can change the default to be mutable as we do for processors.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *baseExporter) {
		o.consumerOptions = append(o.consumerOptions, consumer.WithCapabilities(capabilities))
	}
}

// baseExporter contains common fields between different exporter types.
type baseExporter struct {
	component.StartFunc
	component.ShutdownFunc

	requestExporter bool
	marshaler       RequestMarshaler
	unmarshaler     RequestUnmarshaler
	signal          component.DataType

	set    exporter.CreateSettings
	obsrep *ObsReport

	// Message for the user to be added with an export failure message.
	exportFailureMessage string

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	queueSender   requestSender
	obsrepSender  requestSender
	retrySender   requestSender
	timeoutSender *timeoutSender // timeoutSender is always initialized.

	consumerOptions []consumer.Option
}

// TODO: requestExporter, marshaler, and unmarshaler arguments can be removed when the old exporter helpers will be updated to call the new ones.
func newBaseExporter(set exporter.CreateSettings, signal component.DataType, requestExporter bool, marshaler RequestMarshaler,
	unmarshaler RequestUnmarshaler, osf obsrepSenderFactory, options ...Option) (*baseExporter, error) {

	obsReport, err := NewObsReport(ObsReportSettings{ExporterID: set.ID, ExporterCreateSettings: set})
	if err != nil {
		return nil, err
	}

	be := &baseExporter{
		requestExporter: requestExporter,
		marshaler:       marshaler,
		unmarshaler:     unmarshaler,
		signal:          signal,

		queueSender:   &baseRequestSender{},
		obsrepSender:  osf(obsReport),
		retrySender:   &baseRequestSender{},
		timeoutSender: &timeoutSender{cfg: NewDefaultTimeoutSettings()},

		set:    set,
		obsrep: obsReport,
	}

	for _, op := range options {
		op(be)
	}
	be.connectSenders()

	return be, nil
}

// send sends the request using the first sender in the chain.
func (be *baseExporter) send(ctx context.Context, req Request) error {
	err := be.queueSender.send(ctx, req)
	if err != nil {
		be.set.Logger.Error("Exporting failed. Rejecting data."+be.exportFailureMessage,
			zap.Error(err), zap.Int("rejected_items", req.ItemsCount()))
	}
	return err
}

// connectSenders connects the senders in the predefined order.
func (be *baseExporter) connectSenders() {
	be.queueSender.setNextSender(be.obsrepSender)
	be.obsrepSender.setNextSender(be.retrySender)
	be.retrySender.setNextSender(be.timeoutSender)
}

func (be *baseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the wrapped exporter.
	if err := be.StartFunc.Start(ctx, host); err != nil {
		return err
	}

	// If no error then start the queueSender.
	return be.queueSender.Start(ctx, host)
}

func (be *baseExporter) Shutdown(ctx context.Context) error {
	return multierr.Combine(
		// First shutdown the retry sender, so the queue sender can flush the queue without retries.
		be.retrySender.Shutdown(ctx),
		// Then shutdown the queue sender.
		be.queueSender.Shutdown(ctx),
		// Last shutdown the wrapped exporter itself.
		be.ShutdownFunc.Shutdown(ctx))
}
