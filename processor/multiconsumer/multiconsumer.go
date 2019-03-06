// Copyright 2019, OpenCensus Authors
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

package multiconsumer

import (
	"context"

	"github.com/census-instrumentation/opencensus-service/consumer"
	"github.com/census-instrumentation/opencensus-service/data"
	"github.com/census-instrumentation/opencensus-service/internal"
	"github.com/census-instrumentation/opencensus-service/processor"
)

// NewMetricsProcessor wraps multiple metrics consumers in a single one.
func NewMetricsProcessor(mcs []consumer.MetricsConsumer) processor.MetricsProcessor {
	return metricsConsumers(mcs)
}

type metricsConsumers []consumer.MetricsConsumer

var _ processor.MetricsProcessor = (*metricsConsumers)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mcs metricsConsumers) ConsumeMetricsData(ctx context.Context, md data.MetricsData) error {
	var errs []error
	for _, mdp := range mcs {
		if err := mdp.ConsumeMetricsData(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}
	return internal.CombineErrors(errs)
}

// NewTraceProcessor wraps multiple trace consumers in a single one.
func NewTraceProcessor(tcs []consumer.TraceConsumer) processor.TraceProcessor {
	return traceConsumers(tcs)
}

type traceConsumers []consumer.TraceConsumer

var _ processor.TraceProcessor = (*traceConsumers)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tcs traceConsumers) ConsumeTraceData(ctx context.Context, td data.TraceData) error {
	var errs []error
	for _, tdp := range tcs {
		if err := tdp.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return internal.CombineErrors(errs)
}
