// Copyright The OpenTelemetry Authors
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

package telemetryprocessor

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/processor"
)

type telemetryProcessor struct {
	name       string
	traceNext  consumer.TraceConsumer
	metricNext consumer.MetricsConsumer
}

var _ consumer.TraceConsumer = (*telemetryProcessor)(nil)

func newTelemetryTracesProcessor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.TraceConsumer,
	cfg *Config,
) *telemetryProcessor {
	return &telemetryProcessor{
		name:      cfg.Name(),
		traceNext: nextConsumer,
	}
}

func newTelemetryMetricsProcessor(
	params component.ProcessorCreateParams,
	nextConsumer consumer.MetricsConsumer,
	cfg *Config,
) *telemetryProcessor {
	return &telemetryProcessor{
		name:       cfg.Name(),
		traceNext:  nil,
		metricNext: nextConsumer,
	}
}

// Start is invoked during service startup.
func (sp *telemetryProcessor) Start(ctx context.Context, _ component.Host) error {
	// emit 0's so that the metric is present and reported, rather than absent
	statsTags := []tag.Mutator{tag.Insert(processor.TagProcessorNameKey, sp.name)}
	_ = stats.RecordWithTags(
		ctx,
		statsTags,
		processor.StatTraceBatchesDroppedCount.M(int64(0)),
		processor.StatDroppedSpanCount.M(int64(0)))

	return nil
}

// ConsumeTraces implements the TracesProcessor interface
func (sp *telemetryProcessor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	ctx = obsreport.ProcessorContext(ctx, sp.name)

	stats := processor.NewSpanCountStats(td)
	processor.RecordsSpanCountMetrics(ctx, stats, processor.StatReceivedSpanCount)

	if err := sp.traceNext.ConsumeTraces(ctx, td); err != nil {
		obsreport.ProcessorTraceDataRefused(ctx, stats.GetAllSpansCount())
		return err
	}

	obsreport.ProcessorTraceDataAccepted(ctx, stats.GetAllSpansCount())

	return nil
}

// ConsumeMetrics implements the MetricsProcessor interface
func (sp *telemetryProcessor) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	ctx = obsreport.ProcessorContext(ctx, sp.name)

	_, numPoints := md.MetricAndDataPointCount()

	if err := sp.metricNext.ConsumeMetrics(ctx, md); err != nil {
		obsreport.ProcessorMetricsDataRefused(ctx, numPoints)
		return err
	}
	obsreport.ProcessorMetricsDataAccepted(ctx, numPoints)
	return nil
}

func (sp *telemetryProcessor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: false}
}

// Shutdown is invoked during service shutdown.
func (sp *telemetryProcessor) Shutdown(context.Context) error {
	return nil
}
