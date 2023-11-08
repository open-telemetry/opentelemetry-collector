// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
)

// RequestSender is an abstraction of a sender for a request independent of the type of the data (traces, metrics, logs).
type requestSender interface {
	start(ctx context.Context, host component.Host) error
	send(req internal.Request) error
	shutdown()
	setNextSender(nextSender requestSender)
}

type baseRequestSender struct {
	nextSender requestSender
}

var _ requestSender = (*baseRequestSender)(nil)

func (b *baseRequestSender) start(context.Context, component.Host) error {
	return nil
}

func (b *baseRequestSender) shutdown() {}

func (b *baseRequestSender) send(req internal.Request) error {
	return b.nextSender.send(req)
}

func (b *baseRequestSender) setNextSender(nextSender requestSender) {
	b.nextSender = nextSender
}

type errorLoggingRequestSender struct {
	baseRequestSender
	logger *zap.Logger
}

func (l *errorLoggingRequestSender) send(req internal.Request) error {
	err := l.baseRequestSender.send(req)
	if err != nil {
		l.logger.Error(
			"Exporting failed",
			zap.Error(err),
		)
	}
	return err
}

type obsrepSenderFactory func(obsrep *ObsReport) requestSender

// baseRequest is a base implementation for the internal.Request.
type baseRequest struct {
	ctx                        context.Context
	processingFinishedCallback func()
}

func (req *baseRequest) Context() context.Context {
	return req.ctx
}

func (req *baseRequest) SetContext(ctx context.Context) {
	req.ctx = ctx
}

func (req *baseRequest) SetOnProcessingFinished(callback func()) {
	req.processingFinishedCallback = callback
}

func (req *baseRequest) OnProcessingFinished() {
	if req.processingFinishedCallback != nil {
		req.processingFinishedCallback()
	}
}

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

// WithRetry overrides the default RetrySettings for an exporter.
// The default RetrySettings is to disable retries.
func WithRetry(retrySettings RetrySettings) Option {
	return func(o *baseExporter) {
		o.retrySender = newRetrySender(o.set.ID, retrySettings, o.set.Logger, o.onTemporaryFailure)
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
		var queue internal.Queue
		if config.Enabled {
			if config.StorageID == nil {
				queue = internal.NewBoundedMemoryQueue(config.QueueSize)
			} else {
				queue = internal.NewPersistentQueue(o.set.Logger, config.QueueSize, o.set.ID, *config.StorageID, o.marshaler, o.unmarshaler, o.signal)
			}
		}
		qs := newQueueSender(o.set, config.NumConsumers, o.signal, queue)
		o.queueSender = qs
		o.setOnTemporaryFailure(qs.onTemporaryFailure)
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
	marshaler       internal.RequestMarshaler
	unmarshaler     internal.RequestUnmarshaler
	signal          component.DataType

	set    exporter.CreateSettings
	obsrep *ObsReport

	// Chain of senders that the exporter helper applies before passing the data to the actual exporter.
	// The data is handled by each sender in the respective order starting from the queueSender.
	// Most of the senders are optional, and initialized with a no-op path-through sender.
	queueSender   requestSender
	obsrepSender  requestSender
	retrySender   requestSender
	timeoutSender *timeoutSender // timeoutSender is always initialized.

	// onTemporaryFailure is a function that is called when the retrySender is unable to send data to the next consumer.
	onTemporaryFailure onRequestHandlingFinishedFunc

	consumerOptions []consumer.Option
}

// TODO: requestExporter, marshaler, and unmarshaler arguments can be removed when the old exporter helpers will be updated to call the new ones.
func newBaseExporter(set exporter.CreateSettings, signal component.DataType, requestExporter bool, marshaler internal.RequestMarshaler,
	unmarshaler internal.RequestUnmarshaler, osf obsrepSenderFactory, options ...Option) (*baseExporter, error) {

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
		retrySender:   &errorLoggingRequestSender{logger: set.Logger},
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
func (be *baseExporter) send(req internal.Request) error {
	return be.queueSender.send(req)
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
	if rs, ok := be.retrySender.(*retrySender); ok {
		rs.onTemporaryFailure = onTemporaryFailure
	}
}
