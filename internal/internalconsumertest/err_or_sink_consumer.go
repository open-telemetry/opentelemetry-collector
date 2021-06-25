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

package internalconsumertest

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/model/pdata"
)

type ErrOrSinkConsumer struct {
	*consumertest.TracesSink
	*consumertest.MetricsSink
	mu           sync.Mutex
	consumeError error // to be returned by ConsumeTraces, if set
}

// SetConsumeError sets an error that will be returned by the Consume function.
func (esc *ErrOrSinkConsumer) SetConsumeError(err error) {
	esc.mu.Lock()
	defer esc.mu.Unlock()
	esc.consumeError = err
}

func (esc *ErrOrSinkConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces stores traces to this sink.
func (esc *ErrOrSinkConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.TracesSink.ConsumeTraces(ctx, td)
}

// ConsumeMetrics stores metrics to this sink.
func (esc *ErrOrSinkConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	if esc.consumeError != nil {
		return esc.consumeError
	}

	return esc.MetricsSink.ConsumeMetrics(ctx, md)
}

// Reset deletes any stored in the sinks, resets error to nil.
func (esc *ErrOrSinkConsumer) Reset() {
	esc.mu.Lock()
	defer esc.mu.Unlock()

	esc.consumeError = nil
	if esc.TracesSink != nil {
		esc.TracesSink.Reset()
	}
	if esc.MetricsSink != nil {
		esc.MetricsSink.Reset()
	}
}
