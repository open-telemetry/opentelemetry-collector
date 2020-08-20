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

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/converter"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
)

// This file contains implementations of cloning Trace/Metrics connectors
// that fan out the data to multiple other consumers. Cloning connectors create
// clones of data before fanning out, which ensures each consumer gets their
// own copy of data and is free to modify it.

// CreateMetricsCloningFanOutConnector is a placeholder function for now.
// It supposed to create an old type connector or a new type connector based on type of provided metrics consumer.
func CreateMetricsCloningFanOutConnector(mcs []consumer.MetricsConsumerBase) consumer.MetricsConsumerBase {
	if len(mcs) == 1 {
		// Don't wrap if no need to do it.
		return mcs[0]
	}
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
		return newMetricsCloningFanOutConnectorOld(metricsConsumersOld)
	}
	return newMetricsCloningFanOutConnector(metricsConsumers)
}

func newMetricsCloningFanOutConnectorOld(mcs []consumer.MetricsConsumerOld) consumer.MetricsConsumerOld {
	return metricsCloningFanOutConnectorOld(mcs)
}

type metricsCloningFanOutConnectorOld []consumer.MetricsConsumerOld

var _ consumer.MetricsConsumerOld = (*metricsCloningFanOutConnectorOld)(nil)

// ConsumeMetricsData exports the MetricsData to all consumers wrapped by the current one.
func (mfc metricsCloningFanOutConnectorOld) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(mfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := mfc[i].ConsumeMetricsData(ctx, pdatautil.CloneMetricsDataOld(md)); err != nil {
			errs = append(errs, err)
		}
	}

	if len(mfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := mfc[len(mfc)-1]
		if err := lastTc.ConsumeMetricsData(ctx, md); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

func newMetricsCloningFanOutConnector(mcs []consumer.MetricsConsumer) consumer.MetricsConsumer {
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
		clone := pdatautil.CloneMetrics(md)
		if err := mfc[i].ConsumeMetrics(ctx, clone); err != nil {
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

// CreateTraceCloningFanOutConnector is a placeholder function for now.
// It supposed to create an old type connector or a new type connector based on type of provided trace consumer.
func CreateTraceCloningFanOutConnector(tcs []consumer.TraceConsumerBase) consumer.TraceConsumerBase {
	if len(tcs) == 1 {
		// Don't wrap if no need to do it.
		return tcs[0]
	}
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
		return newTraceCloningFanOutConnectorOld(traceConsumersOld)
	}
	return newTraceCloningFanOutConnector(traceConsumers)
}

func newTraceCloningFanOutConnectorOld(tcs []consumer.TraceConsumerOld) consumer.TraceConsumerOld {
	return traceCloningFanOutConnectorOld(tcs)
}

type traceCloningFanOutConnectorOld []consumer.TraceConsumerOld

var _ consumer.TraceConsumerOld = (*traceCloningFanOutConnectorOld)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc traceCloningFanOutConnectorOld) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		if err := tfc[i].ConsumeTraceData(ctx, cloneTraceDataOld(td)); err != nil {
			errs = append(errs, err)
		}
	}

	if len(tfc) > 0 {
		// Give the original data to the last consumer.
		lastTc := tfc[len(tfc)-1]
		if err := lastTc.ConsumeTraceData(ctx, td); err != nil {
			errs = append(errs, err)
		}
	}

	return componenterror.CombineErrors(errs)
}

func newTraceCloningFanOutConnector(tcs []consumer.TraceConsumer) consumer.TraceConsumer {
	return traceCloningFanOutConnector(tcs)
}

type traceCloningFanOutConnector []consumer.TraceConsumer

var _ consumer.TraceConsumer = (*traceCloningFanOutConnector)(nil)

// ConsumeTraceData exports the span data to all trace consumers wrapped by the current one.
func (tfc traceCloningFanOutConnector) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(tfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone := td.Clone()
		if err := tfc[i].ConsumeTraces(ctx, clone); err != nil {
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

// NewLogCloningFanOutConnector wraps multiple trace consumers in a single one.
func NewLogCloningFanOutConnector(lcs []consumer.LogsConsumer) consumer.LogsConsumer {
	return LogCloningFanOutConnector(lcs)
}

type LogCloningFanOutConnector []consumer.LogsConsumer

var _ consumer.LogsConsumer = (*LogCloningFanOutConnector)(nil)

// ConsumeLogs exports the span data to all  consumers wrapped by the current one.
func (lfc LogCloningFanOutConnector) ConsumeLogs(ctx context.Context, ld pdata.Logs) error {
	var errs []error

	// Fan out to first len-1 consumers.
	for i := 0; i < len(lfc)-1; i++ {
		// Create a clone of data. We need to clone because consumers may modify the data.
		clone := ld.Clone()
		if err := lfc[i].ConsumeLogs(ctx, clone); err != nil {
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

func cloneTraceDataOld(td consumerdata.TraceData) consumerdata.TraceData {
	clone := consumerdata.TraceData{
		SourceFormat: td.SourceFormat,
		Node:         proto.Clone(td.Node).(*commonpb.Node),
		Resource:     proto.Clone(td.Resource).(*resourcepb.Resource),
	}

	if td.Spans != nil {
		clone.Spans = make([]*tracepb.Span, 0, len(td.Spans))

		for _, span := range td.Spans {
			spanClone := proto.Clone(span).(*tracepb.Span)
			clone.Spans = append(clone.Spans, spanClone)
		}
	}

	return clone
}
