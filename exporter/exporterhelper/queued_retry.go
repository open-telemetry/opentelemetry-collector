// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterhelper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opencensus.io/metric"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

var (
	r = metric.NewRegistry()

	queueSizeGauge, _ = r.AddInt64DerivedGauge(
		obsmetrics.ExporterKey+"/queue_size",
		metric.WithDescription("Current size of the retry queue (in batches)"),
		metric.WithLabelKeys(obsmetrics.ExporterKey),
		metric.WithUnit(metricdata.UnitDimensionless))

	errSendingQueueIsFull = errors.New("sending_queue is full")
)

func init() {
	metricproducer.GlobalManager().AddProducer(r)
}

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
}

// DefaultQueueSettings returns the default settings for QueueSettings.
func DefaultQueueSettings() QueueSettings {
	return QueueSettings{
		Enabled:      true,
		NumConsumers: 10,
		// For 5000 queue elements at 100 requests/sec gives about 50 sec of survival of destination outage.
		// This is a pretty decent value for production.
		// User should calculate this from the perspective of how many seconds to buffer in case of a backend outage,
		// multiply that by the number of requests per seconds.
		QueueSize: 5000,
	}
}

// RetrySettings defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type RetrySettings struct {
	// Enabled indicates whether to not retry sending batches in case of export failure.
	Enabled bool `mapstructure:"enabled"`
	// InitialInterval the time to wait after the first failure before retrying.
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	// MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
	// consecutive retries will always be `MaxInterval`.
	MaxInterval time.Duration `mapstructure:"max_interval"`
	// MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
	// Once this value is reached, the data is discarded.
	MaxElapsedTime time.Duration `mapstructure:"max_elapsed_time"`
}

// DefaultRetrySettings returns the default settings for RetrySettings.
func DefaultRetrySettings() RetrySettings {
	return RetrySettings{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  5 * time.Minute,
	}
}

type queuedRetrySender struct {
	fullName        string
	cfg             QueueSettings
	consumerSender  requestSender
	queue           *internal.BoundedQueue
	retryStopCh     chan struct{}
	traceAttributes []attribute.KeyValue
	logger          *zap.Logger
}

func createSampledLogger(logger *zap.Logger) *zap.Logger {
	if logger.Core().Enabled(zapcore.DebugLevel) {
		// Debugging is enabled. Don't do any sampling.
		return logger
	}

	// Create a logger that samples all messages to 1 per 10 seconds initially,
	// and 1/100 of messages after that.
	opts := zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewSamplerWithOptions(
			core,
			10*time.Second,
			1,
			100,
		)
	})
	return logger.WithOptions(opts)
}

func newQueuedRetrySender(fullName string, qCfg QueueSettings, rCfg RetrySettings, nextSender requestSender, logger *zap.Logger) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	sampledLogger := createSampledLogger(logger)
	traceAttr := attribute.String(obsmetrics.ExporterKey, fullName)
	return &queuedRetrySender{
		fullName: fullName,
		cfg:      qCfg,
		consumerSender: &retrySender{
			traceAttribute: traceAttr,
			cfg:            rCfg,
			nextSender:     nextSender,
			stopCh:         retryStopCh,
			logger:         sampledLogger,
		},
		queue:           internal.NewBoundedQueue(qCfg.QueueSize, func(item interface{}) {}),
		retryStopCh:     retryStopCh,
		traceAttributes: []attribute.KeyValue{traceAttr},
		logger:          sampledLogger,
	}
}

// start is invoked during service startup.
func (qrs *queuedRetrySender) start() error {
	qrs.queue.StartConsumers(qrs.cfg.NumConsumers, func(item interface{}) {
		req := item.(request)
		_ = qrs.consumerSender.send(req)
	})

	// Start reporting queue length metric
	if qrs.cfg.Enabled {
		err := queueSizeGauge.UpsertEntry(func() int64 {
			return int64(qrs.queue.Size())
		}, metricdata.NewLabelValue(qrs.fullName))
		if err != nil {
			return fmt.Errorf("failed to create retry queue size metric: %v", err)
		}
	}

	return nil
}

// send implements the requestSender interface
func (qrs *queuedRetrySender) send(req request) error {
	if !qrs.cfg.Enabled {
		err := qrs.consumerSender.send(req)
		if err != nil {
			qrs.logger.Error(
				"Exporting failed. Dropping data. Try enabling sending_queue to survive temporary failures.",
				zap.Int("dropped_items", req.count()),
			)
		}
		return err
	}

	// Prevent cancellation and deadline to propagate to the context stored in the queue.
	// The grpc/http based receivers will cancel the request context after this function returns.
	req.setContext(noCancellationContext{Context: req.context()})

	span := trace.SpanFromContext(req.context())
	if !qrs.queue.Produce(req) {
		qrs.logger.Error(
			"Dropping data because sending_queue is full. Try increasing queue_size.",
			zap.Int("dropped_items", req.count()),
		)
		span.AddEvent("Dropped item, sending_queue is full.", trace.WithAttributes(qrs.traceAttributes...))
		return errSendingQueueIsFull
	}

	span.AddEvent("Enqueued item.", trace.WithAttributes(qrs.traceAttributes...))
	return nil
}

// shutdown is invoked during service shutdown.
func (qrs *queuedRetrySender) shutdown() {
	// Cleanup queue metrics reporting
	if qrs.cfg.Enabled {
		_ = queueSizeGauge.UpsertEntry(func() int64 {
			return int64(0)
		}, metricdata.NewLabelValue(qrs.fullName))
	}

	// First stop the retry goroutines, so that unblocks the queue workers.
	close(qrs.retryStopCh)

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	qrs.queue.Stop()
}

// TODO: Clean this by forcing all exporters to return an internal error type that always include the information about retries.
type throttleRetry struct {
	err   error
	delay time.Duration
}

func (t throttleRetry) Error() string {
	return "Throttle (" + t.delay.String() + "), error: " + t.err.Error()
}

func (t throttleRetry) Unwrap() error {
	return t.err
}

// NewThrottleRetry creates a new throttle retry error.
func NewThrottleRetry(err error, delay time.Duration) error {
	return throttleRetry{
		err:   err,
		delay: delay,
	}
}

type retrySender struct {
	traceAttribute attribute.KeyValue
	cfg            RetrySettings
	nextSender     requestSender
	stopCh         chan struct{}
	logger         *zap.Logger
}

// send implements the requestSender interface
func (rs *retrySender) send(req request) error {
	if !rs.cfg.Enabled {
		err := rs.nextSender.send(req)
		if err != nil {
			rs.logger.Error(
				"Exporting failed. Try enabling retry_on_failure config option.",
				zap.Error(err),
			)
		}
		return err
	}

	// Do not use NewExponentialBackOff since it calls Reset and the code here must
	// call Reset after changing the InitialInterval (this saves an unnecessary call to Now).
	expBackoff := backoff.ExponentialBackOff{
		InitialInterval:     rs.cfg.InitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         rs.cfg.MaxInterval,
		MaxElapsedTime:      rs.cfg.MaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	expBackoff.Reset()
	span := trace.SpanFromContext(req.context())
	retryNum := int64(0)
	for {
		span.AddEvent(
			"Sending request.",
			trace.WithAttributes(rs.traceAttribute, attribute.Int64("retry_num", retryNum)))

		err := rs.nextSender.send(req)
		if err == nil {
			return nil
		}

		// Immediately drop data on permanent errors.
		if consumererror.IsPermanent(err) {
			rs.logger.Error(
				"Exporting failed. The error is not retryable. Dropping data.",
				zap.Error(err),
				zap.Int("dropped_items", req.count()),
			)
			return err
		}

		// Give the request a chance to extract signal data to retry if only some data
		// failed to process.
		req = req.onError(err)

		backoffDelay := expBackoff.NextBackOff()
		if backoffDelay == backoff.Stop {
			// throw away the batch
			err = fmt.Errorf("max elapsed time expired %w", err)
			rs.logger.Error(
				"Exporting failed. No more retries left. Dropping data.",
				zap.Error(err),
				zap.Int("dropped_items", req.count()),
			)
			return err
		}

		throttleErr := throttleRetry{}
		isThrottle := errors.As(err, &throttleErr)
		if isThrottle {
			backoffDelay = max(backoffDelay, throttleErr.delay)
		}

		backoffDelayStr := backoffDelay.String()
		span.AddEvent(
			"Exporting failed. Will retry the request after interval.",
			trace.WithAttributes(
				rs.traceAttribute,
				attribute.String("interval", backoffDelayStr),
				attribute.String("error", err.Error())))
		rs.logger.Info(
			"Exporting failed. Will retry the request after interval.",
			zap.Error(err),
			zap.String("interval", backoffDelayStr),
		)
		retryNum++

		// back-off, but get interrupted when shutting down or request is cancelled or timed out.
		select {
		case <-req.context().Done():
			return fmt.Errorf("request is cancelled or timed out %w", err)
		case <-rs.stopCh:
			return fmt.Errorf("interrupted due to shutdown %w", err)
		case <-time.After(backoffDelay):
		}
	}
}

// max returns the larger of x or y.
func max(x, y time.Duration) time.Duration {
	if x < y {
		return y
	}
	return x
}

type noCancellationContext struct {
	context.Context
}

func (noCancellationContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (noCancellationContext) Done() <-chan struct{} {
	return nil
}

func (noCancellationContext) Err() error {
	return nil
}
