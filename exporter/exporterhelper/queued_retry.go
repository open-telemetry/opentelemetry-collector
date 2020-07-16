// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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

// QueuedSettings defines configuration for queueing batches before sending to the nextSender.
type QueuedSettings struct {
	// Disabled indicates whether to not enqueue batches before sending to the nextSender.
	Disabled bool `mapstructure:"disabled"`
	// NumWorkers is the number of consumers from the queue.
	NumWorkers int `mapstructure:"num_workers"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `mapstructure:"queue_size"`
}

func CreateDefaultQueuedSettings() QueuedSettings {
	return QueuedSettings{
		Disabled:   false,
		NumWorkers: 10,
		QueueSize:  5000,
	}
}

// RetrySettings defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type RetrySettings struct {
	// Disabled indicates whether to not retry sending batches in case of export failure.
	Disabled bool `mapstructure:"disabled"`
	// InitialBackoff the time to wait after the first failure before retrying
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`
	// MaxBackoff is the upper bound on backoff.
	MaxBackoff time.Duration `mapstructure:"max_backoff"`
	// MaxElapsedTime is the maximum amount of time spent trying to send a batch.
	MaxElapsedTime time.Duration `mapstructure:"max_elapsed_time"`
}

func CreateDefaultRetrySettings() RetrySettings {
	return RetrySettings{
		Disabled:       false,
		InitialBackoff: 5 * time.Second,
		MaxBackoff:     30 * time.Second,
		MaxElapsedTime: 5 * time.Minute,
	}
}

type queuedSender struct {
	cfg        *QueuedSettings
	nextSender requestSender
	queue      *queue.BoundedQueue
}

var errorRefused = errors.New("failed to add to the queue")

func newQueuedSender(cfg *QueuedSettings, nextSender requestSender) *queuedSender {
	return &queuedSender{
		cfg:        cfg,
		nextSender: nextSender,
		queue:      queue.NewBoundedQueue(cfg.QueueSize, func(item interface{}) {}),
	}
}

// start is invoked during service startup.
func (sp *queuedSender) start() {
	sp.queue.StartConsumers(sp.cfg.NumWorkers, func(item interface{}) {
		value := item.(request)
		_, _ = sp.nextSender.send(value)
	})
}

// ExportTraces implements the TExporter interface
func (sp *queuedSender) send(req request) (int, error) {
	if sp.cfg.Disabled {
		return sp.nextSender.send(req)
	}

	if !sp.queue.Produce(req) {
		return req.count(), errorRefused
	}

	return 0, nil
}

// shutdown is invoked during service shutdown.
func (sp *queuedSender) shutdown() {
	sp.queue.Stop()
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
	cfg        *RetrySettings
	nextSender requestSender
	stopCh     chan struct{}
}

func newRetrySender(cfg *RetrySettings, nextSender requestSender) *retrySender {
	return &retrySender{
		cfg:        cfg,
		nextSender: nextSender,
		stopCh:     make(chan struct{}),
	}
}

func (re *retrySender) send(req request) (int, error) {
	if re.cfg.Disabled {
		return re.nextSender.send(req)
	}

	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = re.cfg.InitialBackoff
	expBackoff.MaxInterval = re.cfg.MaxBackoff
	expBackoff.MaxElapsedTime = re.cfg.MaxElapsedTime
	for {
		droppedItems, err := re.nextSender.send(req)

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
		case <-re.stopCh:
			return req.count(), fmt.Errorf("interrupted due to shutdown %w", err)
		case <-time.After(backoffDelay):
		}
	}
}

// shutdown is invoked during service shutdown.
func (re *retrySender) shutdown() {
	close(re.stopCh)
}

// max returns the larger of x or y.
func max(x, y time.Duration) time.Duration {
	if x < y {
		return y
	}
	return x
}
