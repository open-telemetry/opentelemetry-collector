// Copyright 2019, OpenTelemetry Authors
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

package queuedprocessor

import (
	"time"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/processor/batchprocessor"
)

const (
	// DefaultNumWorkers is the default number of workers consuming from the processor queue
	DefaultNumWorkers = 10
	// DefaultQueueSize is the default maximum number of span batches allowed in the processor's queue
	DefaultQueueSize = 1000
)

type options struct {
	logger                   *zap.Logger
	name                     string
	numWorkers               int
	queueSize                int
	backoffDelay             time.Duration
	extraFormatTypes         []string
	retryOnProcessingFailure bool
	batchingEnabled          bool
	batchingOptions          []batchprocessor.Option
}

// Option is a function that sets some option on the component.
type Option func(c *options)

// Options is a factory for all available Option's
var Options options

// WithLogger creates a Option that initializes the logger
func (options) WithLogger(logger *zap.Logger) Option {
	return func(b *options) {
		b.logger = logger
	}
}

// WithName creates an Option that initializes the name of the processor
func (options) WithName(name string) Option {
	return func(b *options) {
		b.name = name
	}
}

// WithNumWorkers creates an Option that initializes the number of queue consumers AKA workers
func (options) WithNumWorkers(numWorkers int) Option {
	return func(b *options) {
		b.numWorkers = numWorkers
	}
}

// WithQueueSize creates an Option that initializes the queue size
func (options) WithQueueSize(queueSize int) Option {
	return func(b *options) {
		b.queueSize = queueSize
	}
}

// WithBackoffDelay creates an Option that initializes the backoff delay
func (options) WithBackoffDelay(backoffDelay time.Duration) Option {
	return func(b *options) {
		b.backoffDelay = backoffDelay
	}
}

// WithExtraFormatTypes creates an Option that initializes the extra list of format types
func (options) WithExtraFormatTypes(extraFormatTypes []string) Option {
	return func(b *options) {
		b.extraFormatTypes = extraFormatTypes
	}
}

// WithRetryOnProcessingFailures creates an Option that initializes the retryOnProcessingFailure boolean
func (options) WithRetryOnProcessingFailures(retryOnProcessingFailure bool) Option {
	return func(b *options) {
		b.retryOnProcessingFailure = retryOnProcessingFailure
	}
}

// WithBatching creates an Option that enabled batching
func (options) WithBatching(batchingEnabled bool) Option {
	return func(b *options) {
		b.batchingEnabled = batchingEnabled
	}
}

// WithBatchingOptions creates an Option that will apply batcher options to
// the batcher if batching is enabled.
func (options) WithBatchingOptions(opts ...batchprocessor.Option) Option {
	return func(b *options) {
		b.batchingOptions = opts
	}
}

func (o options) apply(opts ...Option) options {
	ret := options{}
	for _, opt := range opts {
		opt(&ret)
	}
	if ret.logger == nil {
		ret.logger = zap.NewNop()
	}
	if ret.numWorkers == 0 {
		ret.numWorkers = DefaultNumWorkers
	}
	if ret.queueSize == 0 {
		ret.queueSize = DefaultQueueSize
	}
	return ret
}
