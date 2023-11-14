// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

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

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/internal/obsreportconfig/obsmetrics"
)

// RetrySettings defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type RetrySettings struct {
	// Enabled indicates whether to not retry sending batches in case of export failure.
	Enabled bool `mapstructure:"enabled"`
	// InitialInterval the time to wait after the first failure before retrying.
	InitialInterval time.Duration `mapstructure:"initial_interval"`
	// RandomizationFactor is a random factor used to calculate next backoffs
	// Randomized interval = RetryInterval * (1 ± RandomizationFactor)
	RandomizationFactor float64 `mapstructure:"randomization_factor"`
	// Multiplier is the value multiplied by the backoff interval bounds
	Multiplier float64 `mapstructure:"multiplier"`
	// MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
	// consecutive retries will always be `MaxInterval`.
	MaxInterval time.Duration `mapstructure:"max_interval"`
	// MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
	// Once this value is reached, the data is discarded.
	MaxElapsedTime time.Duration `mapstructure:"max_elapsed_time"`
}

// NewDefaultRetrySettings returns the default settings for RetrySettings.
func NewDefaultRetrySettings() RetrySettings {
	return RetrySettings{
		Enabled:             true,
		InitialInterval:     5 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      5 * time.Minute,
	}
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

type onRequestHandlingFinishedFunc func(context.Context, Request, error, *zap.Logger) error

type retrySender struct {
	baseRequestSender
	traceAttribute     attribute.KeyValue
	cfg                RetrySettings
	stopCh             chan struct{}
	logger             *zap.Logger
	onTemporaryFailure onRequestHandlingFinishedFunc
}

func newRetrySender(config RetrySettings, set exporter.CreateSettings, onTemporaryFailure onRequestHandlingFinishedFunc) *retrySender {
	if onTemporaryFailure == nil {
		onTemporaryFailure = func(_ context.Context, _ Request, err error, _ *zap.Logger) error {
			return err
		}
	}
	return &retrySender{
		traceAttribute:     attribute.String(obsmetrics.ExporterKey, set.ID.String()),
		cfg:                config,
		stopCh:             make(chan struct{}),
		logger:             set.Logger,
		onTemporaryFailure: onTemporaryFailure,
	}
}

func (rs *retrySender) Shutdown(context.Context) error {
	close(rs.stopCh)
	return nil
}

// send implements the requestSender interface
func (rs *retrySender) send(ctx context.Context, req Request) error {
	// Do not use NewExponentialBackOff since it calls Reset and the code here must
	// call Reset after changing the InitialInterval (this saves an unnecessary call to Now).
	expBackoff := backoff.ExponentialBackOff{
		InitialInterval:     rs.cfg.InitialInterval,
		RandomizationFactor: rs.cfg.RandomizationFactor,
		Multiplier:          rs.cfg.Multiplier,
		MaxInterval:         rs.cfg.MaxInterval,
		MaxElapsedTime:      rs.cfg.MaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	expBackoff.Reset()
	span := trace.SpanFromContext(ctx)
	retryNum := int64(0)
	for {
		span.AddEvent(
			"Sending request.",
			trace.WithAttributes(rs.traceAttribute, attribute.Int64("retry_num", retryNum)))

		err := rs.nextSender.send(ctx, req)
		if err == nil {
			return nil
		}

		// Immediately drop data on permanent errors.
		if consumererror.IsPermanent(err) {
			rs.logger.Error(
				"Exporting failed. The error is not retryable. Dropping data.",
				zap.Error(err),
				zap.Int("dropped_items", req.ItemsCount()),
			)
			return err
		}

		// Give the request a chance to extract signal data to retry if only some data
		// failed to process.
		if errReq, ok := req.(RequestErrorHandler); ok {
			req = errReq.OnError(err)
		}

		backoffDelay := expBackoff.NextBackOff()
		if backoffDelay == backoff.Stop {
			// throw away the batch
			err = fmt.Errorf("max elapsed time expired %w", err)
			return rs.onTemporaryFailure(ctx, req, err, rs.logger)
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
		case <-ctx.Done():
			return fmt.Errorf("Request is cancelled or timed out %w", err)
		case <-rs.stopCh:
			return rs.onTemporaryFailure(ctx, req, fmt.Errorf("interrupted due to shutdown %w", err), rs.logger)
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
