// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package configretry // import "go.opentelemetry.io/collector/config/configretry"

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// NewDefaultBackOffConfig returns the default settings for RetryConfig.
func NewDefaultBackOffConfig() BackOffConfig {
	return BackOffConfig{
		Enabled:             true,
		InitialInterval:     5 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         30 * time.Second,
		MaxElapsedTime:      5 * time.Minute,
	}
}

// BackOffConfig defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type BackOffConfig struct {
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
	// Once this value is reached, the data is discarded. If set to 0, the retries are never stopped.
	MaxElapsedTime time.Duration `mapstructure:"max_elapsed_time"`
}

func (bs *BackOffConfig) Validate() error {
	if !bs.Enabled {
		return nil
	}
	if bs.InitialInterval < 0 {
		return errors.New("'initial_interval' must be non-negative")
	}
	if bs.RandomizationFactor < 0 || bs.RandomizationFactor > 1 {
		return errors.New("'randomization_factor' must be within [0, 1]")
	}
	if bs.Multiplier < 0 {
		return errors.New("'multiplier' must be non-negative")
	}
	if bs.MaxInterval < 0 {
		return errors.New("'max_interval' must be non-negative")
	}
	if bs.MaxElapsedTime < 0 {
		return errors.New("'max_elapsed_time' must be non-negative")
	}
	if bs.MaxElapsedTime > 0 {
		if bs.MaxElapsedTime < bs.InitialInterval {
			return errors.New("'max_elapsed_time' must not be less than 'initial_interval'")
		}
		if bs.MaxElapsedTime < bs.MaxInterval {
			return errors.New("'max_elapsed_time' must not be less than 'max_interval'")
		}

	}
	return nil
}

func NewBackOff[T any](cfg BackOffConfig, set component.TelemetrySettings, next consumer.ConsumeFunc[T]) consumer.ConsumeFunc[T] {
	return func(ctx context.Context, req T) error {
		// Do not use NewExponentialBackOff since it calls Reset and the code here must
		// call Reset after changing the InitialInterval (this saves an unnecessary call to Now).
		expBackoff := backoff.ExponentialBackOff{
			InitialInterval:     cfg.InitialInterval,
			RandomizationFactor: cfg.RandomizationFactor,
			Multiplier:          cfg.Multiplier,
			MaxInterval:         cfg.MaxInterval,
			MaxElapsedTime:      cfg.MaxElapsedTime,
			Stop:                backoff.Stop,
			Clock:               backoff.SystemClock,
		}
		expBackoff.Reset()
		span := trace.SpanFromContext(ctx)
		retryNum := int64(0)
		for {
			span.AddEvent(
				"Sending request.",
				trace.WithAttributes(attribute.Int64("retry_num", retryNum)))

			err := next(ctx, req)
			if err == nil {
				return nil
			}

			// Immediately drop data on permanent errors.
			if consumererror.IsPermanent(err) {
				return fmt.Errorf("not retryable error: %w", err)
			}

			backoffDelay := expBackoff.NextBackOff()
			if backoffDelay == backoff.Stop {
				return fmt.Errorf("no more retries left: %w", err)
			}

			throttleErr := throttleRetry{}
			if errors.As(err, &throttleErr) {
				backoffDelay = max(backoffDelay, throttleErr.delay)
			}

			if deadline, has := ctx.Deadline(); has && time.Until(deadline) < backoffDelay {
				// The delay is longer than the deadline.  There is no point in
				// waiting for cancelation.
				return fmt.Errorf("request will be cancelled before next retry: %w", err)
			}

			backoffDelayStr := backoffDelay.String()
			if span.IsRecording() {
				span.AddEvent(
					"Exporting failed. Will retry the request after interval.",
					trace.WithAttributes(
						attribute.String("interval", backoffDelayStr),
						attribute.String("error", err.Error())))
			}
			set.Logger.Info(
				"Exporting failed. Will retry the request after interval.",
				zap.Error(err),
				zap.String("interval", backoffDelayStr),
			)
			retryNum++

			// back-off, but get interrupted when shutting down or request is cancelled or timed out.
			select {
			case <-ctx.Done():
				return fmt.Errorf("request is cancelled or timed out %w", err)
			case <-time.After(backoffDelay):
			}
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
