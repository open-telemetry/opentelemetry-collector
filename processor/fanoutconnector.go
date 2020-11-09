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

package processor

import (
	"context"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// This file contains implementations of Trace/Metrics connectors
// that fan out the data to multiple other consumers.

// NewMetricsFanOutConnector wraps multiple metrics consumers in a single one.
func NewMetricsFanOutConnector(mcs []consumer.MetricsConsumer) consumer.MetricsConsumer {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
	return metricsFanOutConnector(mcs)
}

type metricsFanOutConnector []consumer.MetricsConsumer

var _ consumer.MetricsConsumer = (*metricsFanOutConnector)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsFanOutConnector) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	var errs []error
	for _, mc := range mfc {
		if err := mc.ConsumeMetrics(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

// NewTracesFanOutConnector wraps multiple trace consumers in a single one.
func NewTracesFanOutConnector(tcs []consumer.TracesConsumer) consumer.TracesConsumer {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
	return traceFanOutConnector(tcs)
}

type traceFanOutConnector []consumer.TracesConsumer

var _ consumer.TracesConsumer = (*traceFanOutConnector)(nil)

// ConsumeTraces exports the span data to all trace consumers wrapped by the current one.
func (tfc traceFanOutConnector) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error
	for _, tc := range tfc {
		if err := tc.ConsumeTraces(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

// NewLogsFanOutConnector wraps multiple log consumers in a single one.
func NewLogsFanOutConnector(lcs []consumer.LogsConsumer) consumer.LogsConsumer {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	return logsFanOutConnector(lcs)
}

type logsFanOutConnector []consumer.LogsConsumer

var _ consumer.LogsConsumer = (*logsFanOutConnector)(nil)

// Consume exports the log data to all consumers wrapped by the current one.
func (fc logsFanOutConnector) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	var errs []error
	for _, tc := range fc {
		if err := tc.ConsumeLogs(ctx, ld); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}
