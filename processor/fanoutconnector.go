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

package processor

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/oterr"
)

// This file contains implementations of Trace/Metrics connectors
// that fan out the data to multiple other consumers.

// NewMetricsFanOutConnector wraps multiple metrics consumers in a single one.
func NewMetricsFanOutConnector(mcs []consumer.MetricsConsumer) consumer.MetricsConsumer {
	return metricsFanOutConnector(mcs)
}

type metricsFanOutConnector []consumer.MetricsConsumer

var _ consumer.MetricsConsumer = (*metricsFanOutConnector)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsFanOutConnector) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	var errs []error
	for _, mc := range mfc {
		if err := mc.ConsumeMetricsData(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}
	return oterr.CombineErrors(errs)
}

// NewTraceFanOutConnector wraps multiple trace consumers in a single one.
func NewTraceFanOutConnector(tcs []consumer.TraceConsumer) consumer.TraceConsumer {
	return traceFanOutConnector(tcs)
}

type traceFanOutConnector []consumer.TraceConsumer

var _ consumer.TraceConsumer = (*traceFanOutConnector)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc traceFanOutConnector) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error
	for _, tc := range tfc {
		if err := tc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return oterr.CombineErrors(errs)
}
