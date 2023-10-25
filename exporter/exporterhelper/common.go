// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	intrequest "go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/queue/persistentqueue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/request"
)

// requestSender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).
type requestSender interface {
	send(req *intrequest.Request) error
}

type starter interface {
	start(context.Context, component.Host) error
}

type shutdowner interface {
	shutdown()
}

type senderWrapper struct {
	sender     requestSender
	nextSender *senderWrapper
}

func (b *senderWrapper) start(ctx context.Context, host component.Host) error {
	if s, ok := b.sender.(starter); ok {
		return s.start(ctx, host)
	}
	return nil
}

func (b *senderWrapper) shutdown() {
	if s, ok := b.sender.(shutdowner); ok {
		s.shutdown()
	}
}

func (b *senderWrapper) send(req *intrequest.Request) error {
	if b.sender == nil {
		return b.nextSender.send(req)
	}
	return b.sender.send(req)
}

type errorLoggingRequestSender struct {
	logger     *zap.Logger
	nextSender *senderWrapper
}

func (l *errorLoggingRequestSender) send(req *intrequest.Request) error {
	err := l.nextSender.send(req)
	if err != nil {
		l.logger.Error(
			"Exporting failed",
			zap.Error(err),
		)
	}
	return err
}

type obsrepSenderFactory func(obsrep *ObsReport, nextSender *senderWrapper) requestSender

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
		o.timeoutSender.sender = &timeoutSender{cfg: timeoutSettings}
	}
}

// WithRetry overrides the default RetrySettings for an exporter.
// The default RetrySettings is to disable retries.
func WithRetry(retrySettings RetrySettings) Option {
	return func(o *baseExporter) {
		o.retrySender.sender = newRetrySender(o.set.ID, retrySettings, o.set.Logger, o.onTemporaryFailure, o.retrySender.nextSender)
	}
}

// WithQueue overrides the default QueueSettings for an exporter.
// The default QueueSettings is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config QueueSettings) Option {
	return func(o *baseExporter) {
		if o.requestExporter {
			panic("this option is not available for the new request exporters, " +
				"use WithMemoryQueue or WithPersistentQueue instead")
		}
		factory := persistentqueue.NewFactory(persistentqueue.Config{
			Config: queue.Config{
				Enabled:      config.Enabled,
				NumConsumers: config.NumConsumers,
				QueueSize:    config.QueueSize,
			},
			StorageID: config.StorageID,
		}, o.marshaler, o.unmarshaler)
		qs := newQueueSender(o.set, o.signal, factory, o.queueSender.nextSender)
		o.setOnTemporaryFailure(qs.onTemporaryFailure)
		o.queueSender.sender = qs
	}
}

// WithRequestQueue enables queueing for an exporter.
// This option should be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
// This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func WithRequestQueue(queueFactory queue.Factory) Option {
	return func(o *baseExporter) {
		qs := newQueueSender(o.set, o.signal, queueFactory, o.queueSender.nextSender)
		o.setOnTemporaryFailure(qs.onTemporaryFailure)
		o.queueSender.sender = qs
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
	marshaler       request.Marshaler
	unmarshaler     request.Unmarshaler
	signal          component.DataType

	set    exporter.CreateSettings
	obsrep *ObsReport

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	queueSender   *senderWrapper
	obsrepSender  *senderWrapper
	retrySender   *senderWrapper
	timeoutSender *senderWrapper

	// onTemporaryFailure is a function that is called when the retrySender is unable to send data to the next consumer.
	onTemporaryFailure onRequestHandlingFinishedFunc

	consumerOptions []consumer.Option
}

// TODO: requestExporter, marshaler, and unmarshaler arguments can be removed when the old exporter helpers will be updated to call the new ones.
func newBaseExporter(set exporter.CreateSettings, signal component.DataType, requestExporter bool, marshaler request.Marshaler,
	unmarshaler request.Unmarshaler, osf obsrepSenderFactory, options ...Option) (*baseExporter, error) {

	obsReport, err := NewObsReport(ObsReportSettings{ExporterID: set.ID, ExporterCreateSettings: set})
	if err != nil {
		return nil, err
	}

	be := &baseExporter{
		requestExporter: requestExporter,
		marshaler:       marshaler,
		unmarshaler:     unmarshaler,
		signal:          signal,

		set:    set,
		obsrep: obsReport,
	}

	// Initialize the chain of senders in the reverse order.
	be.timeoutSender = &senderWrapper{sender: &timeoutSender{cfg: NewDefaultTimeoutSettings()}}
	be.retrySender = &senderWrapper{
		sender:     &errorLoggingRequestSender{logger: set.Logger, nextSender: be.timeoutSender},
		nextSender: be.timeoutSender,
	}
	be.obsrepSender = &senderWrapper{sender: osf(obsReport, be.retrySender)}
	be.queueSender = &senderWrapper{nextSender: be.obsrepSender}

	for _, op := range options {
		op(be)
	}

	return be, nil
}

// send sends the request using the first sender in the chain.
func (be *baseExporter) send(req *intrequest.Request) error {
	return be.queueSender.send(req)
}

func (be *baseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the wrapped exporter.
	if err := be.StartFunc.Start(ctx, host); err != nil {
		return err
	}

	// If no error then start the queueSender.
	return be.queueSender.start(ctx, host)
}

func (be *baseExporter) Shutdown(ctx context.Context) error {
	// First shutdown the retry sender, so it can push any pending requests to back the queue.
	be.retrySender.shutdown()

	// Then shutdown the queue sender.
	be.queueSender.shutdown()

	// Last shutdown the wrapped exporter itself.
	return be.ShutdownFunc.Shutdown(ctx)
}

func (be *baseExporter) setOnTemporaryFailure(onTemporaryFailure onRequestHandlingFinishedFunc) {
	be.onTemporaryFailure = onTemporaryFailure
	if rs, ok := be.retrySender.sender.(*retrySender); ok {
		rs.onTemporaryFailure = onTemporaryFailure
	}
}
