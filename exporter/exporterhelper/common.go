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
}

type baseRequestSender struct {
	component.StartFunc
	component.ShutdownFunc
	nextSender requestSender
}

type errorLoggingRequestSender struct {
	baseRequestSender
	logger  *zap.Logger
	message string
}

func (l *errorLoggingRequestSender) send(ctx context.Context, req Request) error {
	err := l.nextSender.send(ctx, req)
	if err != nil {
		l.logger.Error(l.message, zap.Int("dropped_items", req.ItemsCount()), zap.Error(err))
	}
	return err
}

type obsrepSenderFactory func(obsrep *ObsReport, nextSender requestSender) requestSender

type settings struct {
	requestExporter bool

	startFunc    component.StartFunc
	shutdownFunc component.ShutdownFunc

	timeoutConfig TimeoutSettings
	queueConfig   QueueSettings
	retryConfig   *configretry.BackOffConfig

	consumerOptions []consumer.Option
}

// Option apply changes to baseExporter.
type Option func(cfg *settings)

// WithStart overrides the default Start function for an exporter.
// The default start function does nothing and always returns nil.
func WithStart(start component.StartFunc) Option {
	return func(o *settings) {
		o.startFunc = start
	}
}

// WithShutdown overrides the default Shutdown function for an exporter.
// The default shutdown function does nothing and always returns nil.
func WithShutdown(shutdown component.ShutdownFunc) Option {
	return func(o *settings) {
		o.shutdownFunc = shutdown
	}
}

// WithTimeout overrides the default TimeoutSettings for an exporter.
// The default TimeoutSettings is 5 seconds.
func WithTimeout(config TimeoutSettings) Option {
	return func(o *settings) {
		o.timeoutConfig = config
	}
}

// WithRetry overrides the default RetrySettings for an exporter.
// The default RetrySettings is to disable retries.
func WithRetry(config configretry.BackOffConfig) Option {
	return func(o *settings) {
		o.retryConfig = &config
	}
}

// WithQueue overrides the default QueueSettings for an exporter.
// The default QueueSettings is to disable queueing.
// This option cannot be used with the new exporter helpers New[Traces|Metrics|Logs]RequestExporter.
func WithQueue(config QueueSettings) Option {
	return func(o *settings) {
		if o.requestExporter {
			panic("queueing is not available for the new request exporters yet")
		}
		o.queueConfig = config
	}
}

// WithCapabilities overrides the default Capabilities() function for a Consumer.
// The default is non-mutable data.
// TODO: Verify if we can change the default to be mutable as we do for processors.
func WithCapabilities(capabilities consumer.Capabilities) Option {
	return func(o *settings) {
		o.consumerOptions = append(o.consumerOptions, consumer.WithCapabilities(capabilities))
	}
}

// baseExporter contains common fields between different exporter types.
type baseExporter struct {
	allSenders []requestSender
	lastSender requestSender

	// TODO: Determine how to remove the need for keeping this references for testing and unconventional shutdown.
	exporterSender *exporterSender
	queueSender    *queueSender
	retrySender    *retrySender

	obsrep          *ObsReport
	consumerOptions []consumer.Option
}

// TODO: requestExporter, marshaler, and unmarshaler arguments can be removed when the old exporter helpers will be updated to call the new ones.
func newBaseExporter(set exporter.CreateSettings, signal component.DataType, requestExporter bool, marshaler RequestMarshaler,
	unmarshaler RequestUnmarshaler, osf obsrepSenderFactory, opts ...Option) (*baseExporter, error) {

	obsReport, err := NewObsReport(ObsReportSettings{ExporterID: set.ID, ExporterCreateSettings: set})
	if err != nil {
		return nil, err
	}

	sets := &settings{
		requestExporter: requestExporter,
		timeoutConfig:   NewDefaultTimeoutSettings(),
	}
	for _, op := range opts {
		op(sets)
	}

	be := &baseExporter{
		obsrep:          obsReport,
		consumerOptions: sets.consumerOptions,
	}

	// Create senders in reverse order:
	// Start with the basic exporter wrapper.
	be.exporterSender = &exporterSender{StartFunc: sets.startFunc, ShutdownFunc: sets.shutdownFunc}
	be.addSender(be.exporterSender)
	// Before that add the timeout sender.
	be.addSender(newTimeoutSender(sets.timeoutConfig, be.lastSender))
	// Before that add the retry sender if configured.
	if sets.retryConfig != nil {
		if !sets.retryConfig.Enabled {
			// TODO: Simplify logic for error logging request. Consider to add it always if both retry_on_failure is not enabled, not only when configured but disabled.
			be.addSender(&errorLoggingRequestSender{
				baseRequestSender: baseRequestSender{nextSender: be.lastSender},
				logger:            set.Logger,
				message:           "Exporting failed. Try enabling retry_on_failure config option to retry on retryable errors",
			})
		} else {
			be.retrySender = newRetrySender(*sets.retryConfig, set, be.lastSender)
			be.addSender(be.retrySender)
		}
	}
	// Before that add the observability sender.
	be.addSender(osf(obsReport, be.lastSender))
	// Before that add the queue sender if configured.
	if sets.queueConfig.Enabled {
		be.queueSender = newQueueSender(sets.queueConfig, set, signal, marshaler, unmarshaler, be.lastSender)
		be.addSender(be.queueSender)
	}

	return be, nil
}

func (be *baseExporter) addSender(sender requestSender) {
	be.allSenders = append(be.allSenders, sender)
	be.lastSender = sender
}

// Start all senders in the reverse order starting from the main exporter.
func (be *baseExporter) Start(ctx context.Context, host component.Host) error {
	// First start the next sender.
	for _, sender := range be.allSenders {
		if err := sender.Start(ctx, host); err != nil {
			return err
		}
	}

	return nil
}

// send the request to the exporter.
func (be *baseExporter) send(ctx context.Context, req Request) error {
	return be.lastSender.send(ctx, req)
}

func (be *baseExporter) Shutdown(ctx context.Context) error {
	var err error
	// First shutdown the retry sender, so it can push any pending requests to back the queue.
	if be.retrySender != nil {
		err = multierr.Append(err, be.retrySender.Shutdown(ctx))
	}
	// Then shutdown the queue sender.
	if be.queueSender != nil {
		err = multierr.Append(err, be.queueSender.Shutdown(ctx))
	}
	// Last shutdown the wrapped exporter itself.
	return multierr.Append(err, be.exporterSender.Shutdown(ctx))
}

type exporterSender struct {
	component.StartFunc
	component.ShutdownFunc
}

// send the request to the exporter.
func (be *exporterSender) send(ctx context.Context, req Request) error {
	return req.Export(ctx)
}
