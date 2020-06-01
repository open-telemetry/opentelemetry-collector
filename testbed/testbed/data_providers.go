// Copyright 2020, OpenTelemetry Authors
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

package testbed

import (
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/translator/internaldata"
)

type DataProvider interface {
	GenerateTraces() pdata.Traces
	GenerateTracesOld() []*tracepb.Span
	GenerateMetrics() data.MetricData
	GenerateMetricsOld() []*metricspb.Metric
}

type PerfTestDataProvider struct {
	options       LoadOptions
	batchesSent   *uint64
	dataItemsSent *uint64
}

func NewPerfTestDataProvider(options LoadOptions, batchesSent *uint64,
	dataItemsSent *uint64) *PerfTestDataProvider {
	return &PerfTestDataProvider{
		options:       options,
		batchesSent:   batchesSent,
		dataItemsSent: dataItemsSent,
	}
}

func (dp *PerfTestDataProvider) GenerateTracesOld() []*tracepb.Span {

	var spans []*tracepb.Span
	traceID := atomic.AddUint64(dp.batchesSent, 1)
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()

		spanID := atomic.AddUint64(dp.dataItemsSent, 1)

		// Create a span.
		span := &tracepb.Span{
			TraceId: GenerateSequentialTraceID(traceID),
			SpanId:  GenerateSequentialSpanID(spanID),
			Name:    &tracepb.TruncatableString{Value: "load-generator-span"},
			Kind:    tracepb.Span_CLIENT,
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{
					"load_generator.span_seq_num": {
						Value: &tracepb.AttributeValue_IntValue{IntValue: int64(spanID)},
					},
					"load_generator.trace_seq_num": {
						Value: &tracepb.AttributeValue_IntValue{IntValue: int64(traceID)},
					},
				},
			},
			StartTime: timeToTimestamp(startTime),
			EndTime:   timeToTimestamp(startTime.Add(time.Duration(time.Millisecond))),
		}

		// Append attributes.
		for k, v := range dp.options.Attributes {
			span.Attributes.AttributeMap[k] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: v}},
			}
		}

		spans = append(spans, span)
	}
	return spans
}

func (dp *PerfTestDataProvider) GenerateTraces() pdata.Traces {

	traceData := pdata.NewTraces()
	traceData.ResourceSpans().Resize(1)
	ilss := traceData.ResourceSpans().At(0).InstrumentationLibrarySpans()
	ilss.Resize(1)
	spans := ilss.At(0).Spans()
	spans.Resize(dp.options.ItemsPerBatch)

	traceID := atomic.AddUint64(dp.batchesSent, 1)
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Duration(time.Millisecond))

		spanID := atomic.AddUint64(dp.dataItemsSent, 1)

		span := spans.At(i)

		// Create a span.
		span.SetTraceID(GenerateSequentialTraceID(traceID))
		span.SetSpanID(GenerateSequentialSpanID(spanID))
		span.SetName("load-generator-span")
		span.SetKind(pdata.SpanKindCLIENT)
		attrs := span.Attributes()
		attrs.UpsertInt("load_generator.span_seq_num", int64(spanID))
		attrs.UpsertInt("load_generator.trace_seq_num", int64(traceID))
		// Additional attributes.
		for k, v := range dp.options.Attributes {
			attrs.UpsertString(k, v)
		}
		span.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
		span.SetEndTime(pdata.TimestampUnixNano(uint64(endTime.UnixNano())))
	}
	return traceData
}

func GenerateSequentialTraceID(id uint64) []byte {
	var traceID [16]byte
	binary.PutUvarint(traceID[:], id)
	return traceID[:]
}

func GenerateSequentialSpanID(id uint64) []byte {
	var spanID [8]byte
	binary.PutUvarint(spanID[:], id)
	return spanID[:]
}

func (dp *PerfTestDataProvider) GenerateMetricsOld() []*metricspb.Metric {

	resource := &resourcepb.Resource{
		Labels: dp.options.Attributes,
	}

	// Generate 7 data points per metric.
	const dataPointsPerMetric = 7

	var metrics []*metricspb.Metric
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		metric := &metricspb.Metric{
			MetricDescriptor: &metricspb.MetricDescriptor{
				Name:        "load_generator_" + strconv.Itoa(i),
				Description: "Load Generator Counter #" + strconv.Itoa(i),
				Unit:        "",
				Type:        metricspb.MetricDescriptor_GAUGE_INT64,
				LabelKeys: []*metricspb.LabelKey{
					{Key: "item_index"},
					{Key: "batch_index"},
				},
			},
			Resource: resource,
		}

		batchIndex := atomic.AddUint64(dp.batchesSent, 1)

		// Generate data points for the metric. We generate timeseries each containing
		// a single data points. This is the most typical payload composition since
		// monitoring libraries typically generated one data point at a time.
		for j := 0; j < dataPointsPerMetric; j++ {
			timeseries := &metricspb.TimeSeries{}

			startTime := time.Now()
			value := atomic.AddUint64(dp.dataItemsSent, 1)

			// Create a data point.
			point := &metricspb.Point{
				Timestamp: timeToTimestamp(startTime),
				Value:     &metricspb.Point_Int64Value{Int64Value: int64(value)},
			}
			timeseries.Points = append(timeseries.Points, point)
			timeseries.LabelValues = []*metricspb.LabelValue{
				{Value: "item_" + strconv.Itoa(j)},
				{Value: "batch_" + strconv.Itoa(int(batchIndex))},
			}

			metric.Timeseries = append(metric.Timeseries, timeseries)
		}

		metrics = append(metrics, metric)
	}
	return metrics
}

func (dp *PerfTestDataProvider) GenerateMetrics() data.MetricData {

	// Generate 7 data points per metric.
	const dataPointsPerMetric = 7

	metricData := data.NewMetricData()
	metricData.ResourceMetrics().Resize(1)
	metricData.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	if dp.options.Attributes != nil {
		attrs := metricData.ResourceMetrics().At(0).Resource().Attributes()
		attrs.InitEmptyWithCapacity(len(dp.options.Attributes))
		for k, v := range dp.options.Attributes {
			attrs.UpsertString(k, v)
		}
	}
	metrics := metricData.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	metrics.Resize(dp.options.ItemsPerBatch)

	for i := 0; i < dp.options.ItemsPerBatch; i++ {
		metric := metrics.At(i)
		metricDescriptor := metric.MetricDescriptor()
		metricDescriptor.InitEmpty()
		metricDescriptor.SetName("load_generator_" + strconv.Itoa(i))
		metricDescriptor.SetDescription("Load Generator Counter #" + strconv.Itoa(i))
		metricDescriptor.SetType(pdata.MetricTypeGaugeInt64)

		batchIndex := atomic.AddUint64(dp.batchesSent, 1)

		// Generate data points for the metric.
		metric.Int64DataPoints().Resize(dataPointsPerMetric)
		for j := 0; j < dataPointsPerMetric; j++ {
			dataPoint := metric.Int64DataPoints().At(j)
			dataPoint.SetStartTime(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
			value := atomic.AddUint64(dp.dataItemsSent, 1)
			dataPoint.SetValue(int64(value))
			dataPoint.LabelsMap().InitFromMap(map[string]string{
				"item_index":  "item_" + strconv.Itoa(j),
				"batch_index": "batch_" + strconv.Itoa(int(batchIndex)),
			})
		}
	}
	return metricData
}

// timeToTimestamp converts a time.Time to a timestamp.Timestamp pointer.
func timeToTimestamp(t time.Time) *timestamp.Timestamp {
	if t.IsZero() {
		return nil
	}
	nanoTime := t.UnixNano()
	return &timestamp.Timestamp{
		Seconds: nanoTime / 1e9,
		Nanos:   int32(nanoTime % 1e9),
	}
}

type GoldenDataProvider struct {
	tracePairsFile string
	spanPairsFile  string
	random         io.Reader
}

func NewGoldenDataProvider(tracePairsFile string, spanPairsFile string, randomSeed int64) *GoldenDataProvider {
	return &GoldenDataProvider{
		tracePairsFile: tracePairsFile,
		spanPairsFile:  spanPairsFile,
		random:         io.Reader(rand.New(rand.NewSource(randomSeed))),
	}
}

func (dp *GoldenDataProvider) GenerateTraces() pdata.Traces {
	resourceSpans, err := goldendataset.GenerateResourceSpans(dp.tracePairsFile, dp.spanPairsFile, dp.random)
	if err != nil {
		log.Printf("cannot generate traces: %s", err)
		resourceSpans = make([]*otlptrace.ResourceSpans, 0)
	}
	return pdata.TracesFromOtlp(resourceSpans)
}

func (dp *GoldenDataProvider) GenerateTracesOld() []*tracepb.Span {
	traces := dp.GenerateTraces()
	spans := make([]*tracepb.Span, 0, traces.SpanCount())
	traceDatas := internaldata.TraceDataToOC(traces)
	for _, traceData := range traceDatas {
		spans = append(spans, traceData.Spans...)
	}
	return spans
}

func (dp *GoldenDataProvider) GenerateMetrics() data.MetricData {
	return data.MetricData{}
}

func (dp *GoldenDataProvider) GenerateMetricsOld() []*metricspb.Metric {
	return make([]*metricspb.Metric, 0)
}
