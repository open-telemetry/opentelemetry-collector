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

// This file contains implementations of cloning Trace/Metrics connectors
// that fan out the data to multiple other consumers. Cloning connectors create
// clones of data before fanning out, which ensures each consumer gets their
// own copy of data and is free to modify it.

// NewMetricsCloningFanOutConnector wraps multiple metrics consumers in a single one and clones the data
// before fanning out.
func NewMetricsCloningFanOutConnector(mcs []consumer.MetricsConsumer) consumer.MetricsConsumer {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
	return metricsCloningFanOutConnector(mcs)
}

type metricsCloningFanOutConnector []consumer.MetricsConsumer

var _ consumer.MetricsConsumer = (*metricsCloningFanOutConnector)(nil)

// ConsumeMetrics exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsCloningFanOutConnector) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(mfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := mfc[i].ConsumeMetrics(ctx, md.Clone()); err != nil {
			errs = append(errs, err)
		}
	}

	if len(mfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := mfc[len(mfc)-1]
		if err := lastTc.ConsumeMetrics(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

// NewTracesCloningFanOutConnector wraps multiple traces consumers in a single one and clones the data
// before fanning out.
func NewTracesCloningFanOutConnector(tcs []consumer.TracesConsumer) consumer.TracesConsumer {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
	return tracesCloningFanOutConnector(tcs)
}

type tracesCloningFanOutConnector []consumer.TracesConsumer

var _ consumer.TracesConsumer = (*tracesCloningFanOutConnector)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc tracesCloningFanOutConnector) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := tfc[i].ConsumeTraces(ctx, td.Clone()); err != nil {
			errs = append(errs, err)
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeTraces(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

// NewLogsCloningFanOutConnector wraps multiple trace consumers in a single one.
func NewLogsCloningFanOutConnector(lcs []consumer.LogsConsumer) consumer.LogsConsumer {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	return logsCloningFanOutConnector(lcs)
}

type logsCloningFanOutConnector []consumer.LogsConsumer

var _ consumer.LogsConsumer = (*logsCloningFanOutConnector)(nil)

// ConsumeLogs exports the log data to all consumers wrapped by the current one.
func (lfc logsCloningFanOutConnector) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(lfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := lfc[i].ConsumeLogs(ctx, ld.Clone()); err != nil {
			errs = append(errs, err)
		}
	}

	if len(lfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := lfc[len(lfc)-1]
		if err := lastTc.ConsumeLogs(ctx, ld); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}
