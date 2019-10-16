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

package batchprocessor

import (
	"time"
)

// Option is an option to batchprocessor.
type Option func(b *batcher)

// WithTimeout sets the time after which a batch will be sent
// regardless of its size.
func WithTimeout(timeout time.Duration) Option {
	return func(b *batcher) {
		b.timeout = timeout
	}
}

// WithNumTickers sets the number of tickers to use to
// divide the work of looping over all nodebuckets.
func WithNumTickers(numTickers int) Option {
	return func(b *batcher) {
		b.numTickers = numTickers
	}
}

// WithTickTime sets the time interval at which the tickers
// will tick.
func WithTickTime(tickTime time.Duration) Option {
	return func(b *batcher) {
		b.tickTime = tickTime
	}
}

// WithSendBatchSize sets the size after which a batch will
// be sent.
func WithSendBatchSize(sendBatchSize int) Option {
	return func(b *batcher) {
		b.sendBatchSize = uint32(sendBatchSize)
	}
}

// WithRemoveAfterTicks sets the number of ticks that must pass
// without new spans arriving for a node before that node is deleted
// from the batcher.
func WithRemoveAfterTicks(cycles int) Option {
	return func(b *batcher) {
		b.removeAfterCycles = uint32(cycles)
	}
}
