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

package fanoutconsumer

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/model/pdata"
)

// NewMetricsCloning wraps multiple metrics consumers in a single one and clones the data
// before fanning out.
func NewMetricsCloning(mcs []consumer.Metrics) consumer.Metrics {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
	return metricsCloningConsumer(mcs)
}

type metricsCloningConsumer []consumer.Metrics

var _ consumer.Metrics = (*metricsCloningConsumer)(nil)

func (mfc metricsCloningConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeMetrics exports the pdata.Metrics to all consumers wrapped by the current one.
func (mfc metricsCloningConsumer) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
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

	return consumererror.Combine(errs)
}

// NewTracesCloning wraps multiple traces consumers in a single one and clones the data
// before fanning out.
func NewTracesCloning(tcs []consumer.Traces) consumer.Traces {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
	return tracesCloningConsumer(tcs)
}

type tracesCloningConsumer []consumer.Traces

var _ consumer.Traces = (*tracesCloningConsumer)(nil)

func (tfc tracesCloningConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeTraces exports the pdata.Traces to all consumers wrapped by the current one.
func (tfc tracesCloningConsumer) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
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

	return consumererror.Combine(errs)
}

// NewLogsCloning wraps multiple trace consumers in a single one and clones the data
// before fanning out.
func NewLogsCloning(lcs []consumer.Logs) consumer.Logs {
	if len(lcs) == 1 {
		// Don't wrap if no need to do it.
		return lcs[0]
	}
	return logsCloningConsumer(lcs)
}

type logsCloningConsumer []consumer.Logs

var _ consumer.Logs = (*logsCloningConsumer)(nil)

func (lfc logsCloningConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// ConsumeLogs exports the pdata.Logs to all consumers wrapped by the current one.
func (lfc logsCloningConsumer) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
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

	return consumererror.Combine(errs)
}
