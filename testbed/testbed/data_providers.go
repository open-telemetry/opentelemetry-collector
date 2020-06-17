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
	"encoding/hex"
	"fmt"
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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/translator/internaldata"
)

//DataProvider defines the interface for generators of test data used to drive various end-to-end tests.
type DataProvider interface {
	//SetLoadGeneratorCounters supplies pointers to LoadGenerator counters.
	//The data provider implementation should increment these as it generates data.
	SetLoadGeneratorCounters(batchesGenerated *uint64, dataItemsGenerated *uint64)
	//GenerateTraces returns an internal Traces instance with an OTLP ResourceSpans slice populated with test data.
	GenerateTraces() (pdata.Traces, bool)
	//GenerateTracesOld returns a slice of OpenCensus Span instances populated with test data.
	GenerateTracesOld() ([]*tracepb.Span, bool)
	//GenerateMetrics returns an internal MetricData instance with an OTLP ResourceMetrics slice of test data.
	GenerateMetrics() (data.MetricData, bool)
	//GenerateMetricsOld returns a slice of OpenCensus Metric instances populated with test data.
	GenerateMetricsOld() ([]*metricspb.Metric, bool)
	//GetGeneratedSpan returns the generated Span matching the provided traceId and spanId or else nil if no match found.
	GetGeneratedSpan(traceID []byte, spanID []byte) *otlptrace.Span
}

//PerfTestDataProvider in an implementation of the DataProvider for use in performance tests.
//Tracing IDs are based on the incremented batch and data items counters.
type PerfTestDataProvider struct {
	options            LoadOptions
	batchesGenerated   *uint64
	dataItemsGenerated *uint64
}

//NewPerfTestDataProvider creates an instance of PerfTestDataProvider which generates test data based on the sizes
//specified in the supplied LoadOptions.
func NewPerfTestDataProvider(options LoadOptions) *PerfTestDataProvider {
	return &PerfTestDataProvider{
		options: options,
	}
}

func (dp *PerfTestDataProvider) SetLoadGeneratorCounters(batchesGenerated *uint64, dataItemsGenerated *uint64) {
	dp.batchesGenerated = batchesGenerated
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *PerfTestDataProvider) GenerateTracesOld() ([]*tracepb.Span, bool) {

	var spans []*tracepb.Span
	traceID := atomic.AddUint64(dp.batchesGenerated, 1)
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()

		spanID := atomic.AddUint64(dp.dataItemsGenerated, 1)

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
	return spans, false
}

func (dp *PerfTestDataProvider) GenerateTraces() (pdata.Traces, bool) {

	traceData := pdata.NewTraces()
	traceData.ResourceSpans().Resize(1)
	ilss := traceData.ResourceSpans().At(0).InstrumentationLibrarySpans()
	ilss.Resize(1)
	spans := ilss.At(0).Spans()
	spans.Resize(dp.options.ItemsPerBatch)

	traceID := atomic.AddUint64(dp.batchesGenerated, 1)
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Duration(time.Millisecond))

		spanID := atomic.AddUint64(dp.dataItemsGenerated, 1)

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
	return traceData, false
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

func (dp *PerfTestDataProvider) GenerateMetricsOld() ([]*metricspb.Metric, bool) {

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

		batchIndex := atomic.AddUint64(dp.batchesGenerated, 1)

		// Generate data points for the metric. We generate timeseries each containing
		// a single data points. This is the most typical payload composition since
		// monitoring libraries typically generated one data point at a time.
		for j := 0; j < dataPointsPerMetric; j++ {
			timeseries := &metricspb.TimeSeries{}

			startTime := time.Now()
			value := atomic.AddUint64(dp.dataItemsGenerated, 1)

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
	return metrics, false
}

func (dp *PerfTestDataProvider) GenerateMetrics() (data.MetricData, bool) {

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
		metricDescriptor.SetType(pdata.MetricTypeInt64)

		batchIndex := atomic.AddUint64(dp.batchesGenerated, 1)

		// Generate data points for the metric.
		metric.Int64DataPoints().Resize(dataPointsPerMetric)
		for j := 0; j < dataPointsPerMetric; j++ {
			dataPoint := metric.Int64DataPoints().At(j)
			dataPoint.SetStartTime(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
			value := atomic.AddUint64(dp.dataItemsGenerated, 1)
			dataPoint.SetValue(int64(value))
			dataPoint.LabelsMap().InitFromMap(map[string]string{
				"item_index":  "item_" + strconv.Itoa(j),
				"batch_index": "batch_" + strconv.Itoa(int(batchIndex)),
			})
		}
	}
	return metricData, false
}

func (dp *PerfTestDataProvider) GetGeneratedSpan(traceID []byte, spanID []byte) *otlptrace.Span {
	// function not supported for this data provider
	return nil
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

//GoldenDataProvider is an implementation of DataProvider for use in correctness tests.
//Provided data from the "Golden" dataset generated using pairwise combinatorial testing techniques.
type GoldenDataProvider struct {
	tracePairsFile     string
	spanPairsFile      string
	random             io.Reader
	batchesGenerated   *uint64
	dataItemsGenerated *uint64
	resourceSpans      []*otlptrace.ResourceSpans
	spansIndex         int
	spansMap           map[string]*otlptrace.Span
}

//NewGoldenDataProvider creates a new instance of GoldenDataProvider which generates test data based
//on the pairwise combinations specified in the tracePairsFile and spanPairsFile input variables.
//The supplied randomSeed is used to initialize the random number generator used in generating tracing IDs.
func NewGoldenDataProvider(tracePairsFile string, spanPairsFile string, randomSeed int64) *GoldenDataProvider {
	return &GoldenDataProvider{
		tracePairsFile: tracePairsFile,
		spanPairsFile:  spanPairsFile,
		random:         io.Reader(rand.New(rand.NewSource(randomSeed))),
	}
}

func (dp *GoldenDataProvider) SetLoadGeneratorCounters(batchesGenerated *uint64, dataItemsGenerated *uint64) {
	dp.batchesGenerated = batchesGenerated
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *GoldenDataProvider) GenerateTraces() (pdata.Traces, bool) {
	if dp.resourceSpans == nil {
		var err error
		dp.resourceSpans, err = goldendataset.GenerateResourceSpans(dp.tracePairsFile, dp.spanPairsFile, dp.random)
		if err != nil {
			log.Printf("cannot generate traces: %s", err)
			dp.resourceSpans = make([]*otlptrace.ResourceSpans, 0)
		}
	}
	atomic.AddUint64(dp.batchesGenerated, 1)
	if dp.spansIndex >= len(dp.resourceSpans) {
		return pdata.TracesFromOtlp(make([]*otlptrace.ResourceSpans, 0)), true
	}
	resourceSpans := make([]*otlptrace.ResourceSpans, 1)
	resourceSpans[0] = dp.resourceSpans[dp.spansIndex]
	dp.spansIndex++
	spanCount := uint64(0)
	for _, libSpans := range resourceSpans[0].InstrumentationLibrarySpans {
		spanCount += uint64(len(libSpans.Spans))
	}
	atomic.AddUint64(dp.dataItemsGenerated, spanCount)
	return pdata.TracesFromOtlp(resourceSpans), false
}

func (dp *GoldenDataProvider) GenerateTracesOld() ([]*tracepb.Span, bool) {
	traces, done := dp.GenerateTraces()
	spans := make([]*tracepb.Span, 0, traces.SpanCount())
	traceDatas := internaldata.TraceDataToOC(traces)
	for _, traceData := range traceDatas {
		spans = append(spans, traceData.Spans...)
	}
	return spans, done
}

func (dp *GoldenDataProvider) GenerateMetrics() (data.MetricData, bool) {
	return data.MetricData{}, true
}

func (dp *GoldenDataProvider) GenerateMetricsOld() ([]*metricspb.Metric, bool) {
	return make([]*metricspb.Metric, 0), true
}

func (dp *GoldenDataProvider) GetGeneratedSpan(traceID []byte, spanID []byte) *otlptrace.Span {
	if dp.spansMap == nil {
		dp.spansMap = populateSpansMap(dp.resourceSpans)
	}
	key := traceIDAndSpanIDToString(traceID, spanID)
	return dp.spansMap[key]
}

func populateSpansMap(resourceSpansList []*otlptrace.ResourceSpans) map[string]*otlptrace.Span {
	spansMap := make(map[string]*otlptrace.Span)
	for _, resourceSpans := range resourceSpansList {
		for _, libSpans := range resourceSpans.InstrumentationLibrarySpans {
			for _, span := range libSpans.Spans {
				key := traceIDAndSpanIDToString(span.TraceId, span.SpanId)
				spansMap[key] = span
			}
		}
	}
	return spansMap
}

func traceIDAndSpanIDToString(traceID []byte, spanID []byte) string {
	return fmt.Sprintf("%s-%s", hex.EncodeToString(traceID), hex.EncodeToString(spanID))
}
