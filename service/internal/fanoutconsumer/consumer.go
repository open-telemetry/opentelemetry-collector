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

// Package fanoutconsumer contains implementations of Traces/Metrics/Logs consumers
// that fan out the data to multiple other consumers.
//
// Cloning connectors create clones of data before fanning out, which ensures each
// consumer gets their own copy of data and is free to modify it.
package fanoutconsumer

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/model/pdata"
)

// NewMetrics wraps multiple metrics consumers in a single one.
func NewMetrics(mcs []consumer.Metrics) consumer.Metrics {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
	return metricsConsumer(mcs)
}

type metricsConsumer []consumer.Metrics

var _ consumer.Metrics = (*metricsConsumer)(nil)

func (mfc metricsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics exports the pdata.Metrics to all consumers wrapped by the current one.
func (mfc metricsConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	var errs []error
	for _, mc := range mfc {
		if err := mc.ConsumeMetrics(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}
	return consumererror.Combine(errs)
}

// NewTraces wraps multiple trace consumers in a single one.
func NewTraces(tcs []consumer.Traces) consumer.Traces {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
	return traceConsumer(tcs)
}

type traceConsumer []consumer.Traces

var _ consumer.Traces = (*traceConsumer)(nil)

func (tfc traceConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeTraces exports the pdata.Traces to all consumers wrapped by the current one.
func (tfc traceConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error
	for _, tc := range tfc {
		if err := tc.ConsumeTraces(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return consumererror.Combine(errs)
}

// NewLogs wraps multiple log consumers in a single one.
func NewLogs(lcs []consumer.Logs) consumer.Logs {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	return logsConsumer(lcs)
}

type logsConsumer []consumer.Logs

var _ consumer.Logs = (*logsConsumer)(nil)

func (lfc logsConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeLogs exports the pdata.Logs to all consumers wrapped by the current one.
func (lfc logsConsumer) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	var errs []error
	for _, lc := range lfc {
		if err := lc.ConsumeLogs(ctx, ld); err != nil {
			errs = append(errs, err)
		}
	}
	return consumererror.Combine(errs)
}
