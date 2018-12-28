// Copyright 2018, OpenCensus Authors
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

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"github.com/census-instrumentation/opencensus-service/internal/collector/telemetry"
)

// SpanProcessor handles batches of spans converted to OpenCensus proto format.
type SpanProcessor interface {
	// ProcessSpans processes spans and return with the number of spans that failed and an error.
	ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error)
	// TODO: (@pjanotti) For shutdown improvement, the interface needs a method to attempt that.
}

// An initial processor that does not sends the data to any destination but helps debugging.
type debugSpanProcessor struct{ logger *zap.Logger }

var _ SpanProcessor = (*debugSpanProcessor)(nil)

func (sp *debugSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	if batch.Node == nil {
		sp.logger.Warn("Received batch with nil Node", zap.String("format", spanFormat))
	}

	statsTags := statsTagsForBatch("debug", serviceNameForBatch(batch), spanFormat)
	numSpans := len(batch.Spans)
	stats.RecordWithTags(context.Background(), statsTags, statReceivedBatchCount.M(1), statReceivedSpanCount.M(int64(numSpans)))

	sp.logger.Debug("debugSpanProcessor", zap.String("originalFormat", spanFormat), zap.Int("#spans", numSpans))
	return 0, nil
}

// NewNoopSpanProcessor creates an OC SpanProcessor that just drops the received data.
func NewNoopSpanProcessor(logger *zap.Logger) SpanProcessor {
	return &debugSpanProcessor{logger: logger}
}

// Keys and stats for telemetry.
var (
	tagSourceFormatKey, _ = tag.NewKey("format")
	tagServiceNameKey, _  = tag.NewKey("service")
	tagExporterNameKey, _ = tag.NewKey("exporter")

	statReceivedBatchCount = stats.Int64("batches_received", "counts the number of batches received", stats.UnitDimensionless)
	statDroppedBatchCount  = stats.Int64("batches_dropped", "counts the number of batches dropped", stats.UnitDimensionless)

	statReceivedSpanCount = stats.Int64("spans_received", "counts the number of spans received", stats.UnitDimensionless)
	statDroppedSpanCount  = stats.Int64("spans_dropped", "counts the number of spans dropped", stats.UnitDimensionless)
)

// metricTagKeys returns the metric tag keys according to the given telemetry level.
func metricTagKeys(level telemetry.Level) []tag.Key {
	var tagKeys []tag.Key
	switch level {
	case telemetry.Detailed:
		tagKeys = append(tagKeys, tagServiceNameKey)
		fallthrough
	case telemetry.Normal:
		tagKeys = append(tagKeys, tagSourceFormatKey)
		fallthrough
	case telemetry.Basic:
		tagKeys = append(tagKeys, tagExporterNameKey)
		break
	default:
		return nil
	}

	return tagKeys
}

// MetricViews return the metrics views according to given telemetry level.
func MetricViews(level telemetry.Level) []*view.View {
	tagKeys := metricTagKeys(level)
	if tagKeys == nil {
		return nil
	}

	// There are some metrics enabled, return the views.
	receivedBatchesView := &view.View{
		Name:        statReceivedBatchCount.Name(),
		Measure:     statReceivedBatchCount,
		Description: "The number of span batches received.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	droppedBatchesView := &view.View{
		Name:        statDroppedBatchCount.Name(),
		Measure:     statDroppedBatchCount,
		Description: "The number of span batches dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Count(),
	}
	receivedSpansView := &view.View{
		Name:        statReceivedSpanCount.Name(),
		Measure:     statReceivedSpanCount,
		Description: "The number of spans received.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}
	droppedSpansView := &view.View{
		Name:        statDroppedSpanCount.Name(),
		Measure:     statDroppedSpanCount,
		Description: "The number of spans dropped.",
		TagKeys:     tagKeys,
		Aggregation: view.Sum(),
	}

	return []*view.View{receivedBatchesView, droppedBatchesView, receivedSpansView, droppedSpansView}
}

func serviceNameForBatch(batch *agenttracepb.ExportTraceServiceRequest) string {
	var serviceName string
	if batch.Node == nil {
		serviceName = "<nil-batch-node>"
	} else if batch.Node.ServiceInfo == nil {
		serviceName = "<nil-service-info>"
	} else if batch.Node.ServiceInfo.Name == "" {
		serviceName = "<empty-service-info-name>"
	} else {
		serviceName = batch.Node.ServiceInfo.Name
	}
	return serviceName
}

func statsTagsForBatch(processorName, serviceName, spanFormat string) []tag.Mutator {
	statsTags := []tag.Mutator{
		tag.Upsert(tagSourceFormatKey, spanFormat),
		tag.Upsert(tagServiceNameKey, serviceName),
		tag.Upsert(tagExporterNameKey, processorName),
	}

	return statsTags
}
