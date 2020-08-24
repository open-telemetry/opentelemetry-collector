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
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/jaegertracing/jaeger/pkg/queue"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `mapstructure:"num_consumers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
}

// CreateDefaultQueueSettings returns the default settings for QueueSettings.
func CreateDefaultQueueSettings() QueueSettings {
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

// CreateDefaultRetrySettings returns the default settings for RetrySettings.
func CreateDefaultRetrySettings() RetrySettings {
	return RetrySettings{
		Enabled:         true,
		InitialInterval: 5 * time.Second,
		MaxInterval:     30 * time.Second,
		MaxElapsedTime:  5 * time.Minute,
	}
}

type queuedRetrySender struct {
	cfg            QueueSettings
	consumerSender requestSender
	queue          *queue.BoundedQueue
	retryStopCh    chan struct{}
}

var errorRefused = errors.New("failed to add to the queue")

func newQueuedRetrySender(qCfg QueueSettings, rCfg RetrySettings, nextSender requestSender) *queuedRetrySender {
	retryStopCh := make(chan struct{})
	return &queuedRetrySender{
		cfg: qCfg,
		consumerSender: &retrySender{
			cfg:        rCfg,
			nextSender: nextSender,
			stopCh:     retryStopCh,
		},
		queue:       queue.NewBoundedQueue(qCfg.QueueSize, func(item interface{}) {}),
		retryStopCh: retryStopCh,
	}
}

// start is invoked during service startup.
func (qrs *queuedRetrySender) start() {
	qrs.queue.StartConsumers(qrs.cfg.NumConsumers, func(item interface{}) {
		value := item.(request)
		_, _ = qrs.consumerSender.send(value)
	})
}

// send implements the requestSender interface
func (qrs *queuedRetrySender) send(req request) (int, error) {
	if !qrs.cfg.Enabled {
		return qrs.consumerSender.send(req)
	}

	if !qrs.queue.Produce(req) {
		return req.count(), errorRefused
	}

	return 0, nil
}

// shutdown is invoked during service shutdown.
func (qrs *queuedRetrySender) shutdown() {
	// First stop the retry goroutines, so that unblocks the queue workers.
	close(qrs.retryStopCh)

	// Stop the queued sender, this will drain the queue and will call the retry (which is stopped) that will only
	// try once every request.
	qrs.queue.Stop()
}

// TODO: Clean this by forcing all exporters to return an internal error type that always include the information about retries.
type throttleRetry struct {
	error
	delay time.Duration
}

func NewThrottleRetry(err error, delay time.Duration) error {
	return &throttleRetry{
		error: err,
		delay: delay,
	}
}

type retrySender struct {
	cfg        RetrySettings
	nextSender requestSender
	stopCh     chan struct{}
}

// send implements the requestSender interface
func (rs *retrySender) send(req request) (int, error) {
	if !rs.cfg.Enabled {
		return rs.nextSender.send(req)
	}

	// Do not use NewExponentialBackOff since it calls Reset and the code here must
	// call Reset after changing the InitialInterval (this saves an unnecessary call to Now).
	expBackoff := backoff.ExponentialBackOff{
		InitialInterval:     rs.cfg.InitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         rs.cfg.MaxInterval,
		MaxElapsedTime:      rs.cfg.MaxElapsedTime,
		Clock:               backoff.SystemClock,
	}
	expBackoff.Reset()
	for {
		droppedItems, err := rs.nextSender.send(req)

		if err == nil {
			return droppedItems, nil
		}

		// Immediately drop data on permanent errors.
		if consumererror.IsPermanent(err) {
			return droppedItems, err
		}

		// If partial error, update data and stats with non exported data.
		if partialErr, isPartial := err.(consumererror.PartialError); isPartial {
			req = req.onPartialError(partialErr)
		}

		backoffDelay := expBackoff.NextBackOff()

		if backoffDelay == backoff.Stop {
			// throw away the batch
			return req.count(), fmt.Errorf("max elapsed time expired %w", err)
		}

		if throttleErr, isThrottle := err.(*throttleRetry); isThrottle {
			backoffDelay = max(backoffDelay, throttleErr.delay)
		}

		// back-off, but get interrupted when shutting down or request is cancelled or timed out.
		select {
		case <-req.context().Done():
			return req.count(), fmt.Errorf("request is cancelled or timed out %w", err)
		case <-rs.stopCh:
			return req.count(), fmt.Errorf("interrupted due to shutdown %w", err)
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
