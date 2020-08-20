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
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/converter"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// This file contains implementations of Trace/Metrics connectors
// that fan out the data to multiple other consumers.

// CreateMetricsFanOutConnector creates a connector based on provided type of trace consumer.
// If any of the wrapped metrics consumers are of the new type, use metricsFanOutConnector,
// otherwise use the old type connector.
func CreateMetricsFanOutConnector(mcs []consumer.MetricsConsumerBase) consumer.MetricsConsumerBase {
	metricsConsumersOld := make([]consumer.MetricsConsumerOld, 0, len(mcs))
	metricsConsumers := make([]consumer.MetricsConsumer, 0, len(mcs))
	allMetricsConsumersOld := true
	for _, mc := range mcs {
		if metricsConsumer, ok := mc.(consumer.MetricsConsumer); ok {
			allMetricsConsumersOld = false
			metricsConsumers = append(metricsConsumers, metricsConsumer)
		} else {
			metricsConsumerOld := mc.(consumer.MetricsConsumerOld)
			metricsConsumersOld = append(metricsConsumersOld, metricsConsumerOld)
			metricsConsumers = append(metricsConsumers, converter.NewInternalToOCMetricsConverter(metricsConsumerOld))
		}
	}

	if allMetricsConsumersOld {
		return NewMetricsFanOutConnectorOld(metricsConsumersOld)
	}
	return NewMetricsFanOutConnector(metricsConsumers)
}

// NewMetricsFanOutConnectorOld wraps multiple metrics consumers in a single one.
func NewMetricsFanOutConnectorOld(mcs []consumer.MetricsConsumerOld) consumer.MetricsConsumerOld {
	return metricsFanOutConnectorOld(mcs)
}

type metricsFanOutConnectorOld []consumer.MetricsConsumerOld

var _ consumer.MetricsConsumerOld = (*metricsFanOutConnectorOld)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsFanOutConnectorOld) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	var errs []error
	for _, mc := range mfc {
		if err := mc.ConsumeMetricsData(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

// NewMetricsFanOutConnector wraps multiple new type metrics consumers in a single one.
func NewMetricsFanOutConnector(mcs []consumer.MetricsConsumer) consumer.MetricsConsumer {
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

// CreateTraceFanOutConnector wraps multiple trace consumers in a single one.
// If any of the wrapped trace consumers are of the new type, use traceFanOutConnector,
// otherwise use the old type connector
func CreateTraceFanOutConnector(tcs []consumer.TraceConsumerBase) consumer.TraceConsumerBase {
	traceConsumersOld := make([]consumer.TraceConsumerOld, 0, len(tcs))
	traceConsumers := make([]consumer.TraceConsumer, 0, len(tcs))
	allTraceConsumersOld := true
	for _, tc := range tcs {
		if traceConsumer, ok := tc.(consumer.TraceConsumer); ok {
			allTraceConsumersOld = false
			traceConsumers = append(traceConsumers, traceConsumer)
		} else {
			traceConsumerOld := tc.(consumer.TraceConsumerOld)
			traceConsumersOld = append(traceConsumersOld, traceConsumerOld)
			traceConsumers = append(traceConsumers, converter.NewInternalToOCTraceConverter(traceConsumerOld))
		}
	}

	if allTraceConsumersOld {
		return NewTraceFanOutConnectorOld(traceConsumersOld)
	}
	return NewTraceFanOutConnector(traceConsumers)
}

// NewTraceFanOutConnectorOld wraps multiple trace consumers in a single one.
func NewTraceFanOutConnectorOld(tcs []consumer.TraceConsumerOld) consumer.TraceConsumerOld {
	return traceFanOutConnectorOld(tcs)
}

type traceFanOutConnectorOld []consumer.TraceConsumerOld

var _ consumer.TraceConsumerOld = (*traceFanOutConnectorOld)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc traceFanOutConnectorOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error
	for _, tc := range tfc {
		if err := tc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}

// NewTraceFanOutConnector wraps multiple new type trace consumers in a single one.
func NewTraceFanOutConnector(tcs []consumer.TraceConsumer) consumer.TraceConsumer {
	return traceFanOutConnector(tcs)
}

type traceFanOutConnector []consumer.TraceConsumer

var _ consumer.TraceConsumer = (*traceFanOutConnector)(nil)

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

// NewLogFanOutConnector wraps multiple new type  consumers in a single one.
func NewLogFanOutConnector(lcs []consumer.LogsConsumer) consumer.LogsConsumer {
	return LogFanOutConnector(lcs)
}

type LogFanOutConnector []consumer.LogsConsumer

var _ consumer.LogsConsumer = (*LogFanOutConnector)(nil)

// Consume exports the span data to all  consumers wrapped by the current one.
func (fc LogFanOutConnector) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	var errs []error
	for _, tc := range fc {
		if err := tc.ConsumeLogs(ctx, ld); err != nil {
			errs = append(errs, err)
		}
	}
	return componenterror.CombineErrors(errs)
}
