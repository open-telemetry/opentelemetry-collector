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

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

var (
	errSendingQueueIsFull = errors.New("sending_queue is full")
)

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

type onRequestHandlingFinishedFunc func(*zap.Logger, request, error) error

type retrySender struct {
	traceAttribute     attribute.KeyValue
	cfg                RetrySettings
	nextSender         requestSender
	stopCh             chan struct{}
	logger             *zap.Logger
	onTemporaryFailure onRequestHandlingFinishedFunc
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
			return rs.onTemporaryFailure(rs.logger, req, err)
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
