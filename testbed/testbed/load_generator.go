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

package testbed

import (
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/golang/protobuf/ptypes/timestamp"

	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

// LoadGenerator is a simple load generator.
type LoadGenerator struct {
	sender DataSender

	// Number of batches of data items sent.
	batchesSent uint64

	// Number of data items (spans or metric data points) sent.
	dataItemsSent uint64

	stopOnce   sync.Once
	stopWait   sync.WaitGroup
	stopSignal chan struct{}

	options LoadOptions

	// Record information about previous errors to avoid flood of error messages.
	prevErr error
}

// LoadOptions defines the options to use for generating the load.
type LoadOptions struct {
	// DataItemsPerSecond specifies how many spans or metric data points to generate each second.
	DataItemsPerSecond int

	// ItemsPerBatch specifies how many spans or metric data points per batch to generate.
	// Should be greater than zero. The number of batches generated per second will be
	// DataItemsPerSecond/ItemsPerBatch.
	ItemsPerBatch int

	// Attributes to add to each generated data item. Can be empty.
	Attributes map[string]string
}

// NewLoadGenerator creates a load generator that sends data using specified sender.
func NewLoadGenerator(sender DataSender) (*LoadGenerator, error) {
	if sender == nil {
		return nil, fmt.Errorf("cannot create load generator without DataSender")
	}

	lg := &LoadGenerator{
		stopSignal: make(chan struct{}),
		sender:     sender,
	}

	return lg, nil
}

// Start the load.
func (lg *LoadGenerator) Start(options LoadOptions) {
	lg.options = options

	if lg.options.ItemsPerBatch == 0 {
		// 10 items per batch by default.
		lg.options.ItemsPerBatch = 10
	}

	log.Printf("Starting load generator at %d items/sec.", lg.options.DataItemsPerSecond)

	// Indicate that generation is in progress.
	lg.stopWait.Add(1)

	// Begin generation
	go lg.generate()
}

// Stop the load.
func (lg *LoadGenerator) Stop() {
	lg.stopOnce.Do(func() {
		// Signal generate() to stop.
		close(lg.stopSignal)

		// Wait for it to stop.
		lg.stopWait.Wait()

		// Print stats.
		log.Printf("Stopped generator. %s", lg.GetStats())
	})
}

// GetStats returns the stats as a printable string.
func (lg *LoadGenerator) GetStats() string {
	return fmt.Sprintf("Sent:%5d items", atomic.LoadUint64(&lg.dataItemsSent))
}

func (lg *LoadGenerator) DataItemsSent() uint64 {
	return atomic.LoadUint64(&lg.dataItemsSent)
}

// IncDataItemsSent is used when a test bypasses the LoadGenerator and sends data
// directly via TestCases's Sender. This is necessary so that the total number of sent
// items in the end is correct, because the reports are printed from LoadGenerator's
// fields. This is not the best way, a better approach would be to refactor the
// reports to use their own counter and load generator and other sending sources
// to contribute to this counter. This could be done as a future improvement.
func (lg *LoadGenerator) IncDataItemsSent() {
	atomic.AddUint64(&lg.dataItemsSent, 1)
}

func (lg *LoadGenerator) generate() {
	// Indicate that generation is done at the end
	defer lg.stopWait.Done()

	if lg.options.DataItemsPerSecond == 0 {
		return
	}

	err := lg.sender.Start()
	if err != nil {
		log.Printf("Cannot start sender: %v", err)
		return
	}

	t := time.NewTicker(time.Second / time.Duration(lg.options.DataItemsPerSecond/lg.options.ItemsPerBatch))
	defer t.Stop()
	done := false
	for !done {
		select {
		case <-t.C:
			switch lg.sender.(type) {
			case TraceDataSender:
				lg.generateTrace()
			case TraceDataSenderOld:
				lg.generateTraceOld()
			case MetricDataSender:
				lg.generateMetrics()
			case MetricDataSenderOld:
				lg.generateMetricsOld()
			default:
				log.Printf("Invalid type of LoadGenerator sender")
			}

		case <-lg.stopSignal:
			done = true
		}
	}
	// Send all pending generated data.
	lg.sender.Flush()
}

func (lg *LoadGenerator) generateTraceOld() {

	traceSender := lg.sender.(TraceDataSenderOld)

	var spans []*tracepb.Span
	traceID := atomic.AddUint64(&lg.batchesSent, 1)
	for i := 0; i < lg.options.ItemsPerBatch; i++ {

		startTime := time.Now()

		spanID := atomic.AddUint64(&lg.dataItemsSent, 1)

		// Create a span.
		span := &tracepb.Span{
			TraceId: GenerateTraceID(traceID),
			SpanId:  GenerateSpanID(spanID),
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
		for k, v := range lg.options.Attributes {
			span.Attributes.AttributeMap[k] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: v}},
			}
		}

		spans = append(spans, span)
	}

	traceData := consumerdata.TraceData{
		Spans: spans,
	}

	err := traceSender.SendSpans(traceData)
	if err == nil {
		lg.prevErr = nil
	} else if lg.prevErr == nil || lg.prevErr.Error() != err.Error() {
		lg.prevErr = err
		log.Printf("Cannot send traces: %v", err)
	}
}

func (lg *LoadGenerator) generateTrace() {
	traceSender := lg.sender.(TraceDataSender)

	traceData := pdata.NewTraces()
	traceData.ResourceSpans().Resize(1)
	ilss := traceData.ResourceSpans().At(0).InstrumentationLibrarySpans()
	ilss.Resize(1)
	spans := ilss.At(0).Spans()
	spans.Resize(lg.options.ItemsPerBatch)

	traceID := atomic.AddUint64(&lg.batchesSent, 1)
	for i := 0; i < lg.options.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Duration(time.Millisecond))

		spanID := atomic.AddUint64(&lg.dataItemsSent, 1)

		span := spans.At(i)

		// Create a span.
		span.SetTraceID(GenerateTraceID(traceID))
		span.SetSpanID(GenerateSpanID(spanID))
		span.SetName("load-generator-span")
		span.SetKind(pdata.SpanKindCLIENT)
		attrs := span.Attributes()
		attrs.UpsertInt("load_generator.span_seq_num", int64(spanID))
		attrs.UpsertInt("load_generator.trace_seq_num", int64(traceID))
		// Additional attributes.
		for k, v := range lg.options.Attributes {
			attrs.UpsertString(k, v)
		}
		span.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
		span.SetEndTime(pdata.TimestampUnixNano(uint64(endTime.UnixNano())))
	}

	err := traceSender.SendSpans(traceData)
	if err == nil {
		lg.prevErr = nil
	} else if lg.prevErr == nil || lg.prevErr.Error() != err.Error() {
		lg.prevErr = err
		log.Printf("Cannot send traces: %v", err)
	}
}
func GenerateTraceID(id uint64) []byte {
	var traceID [16]byte
	binary.PutUvarint(traceID[:], id)
	return traceID[:]
}

func GenerateSpanID(id uint64) []byte {
	var spanID [8]byte
	binary.PutUvarint(spanID[:], id)
	return spanID[:]
}

func (lg *LoadGenerator) generateMetricsOld() {

	metricSender := lg.sender.(MetricDataSenderOld)

	resource := &resourcepb.Resource{
		Labels: lg.options.Attributes,
	}

	// Generate 7 data points per metric.
	const dataPointsPerMetric = 7

	var metrics []*metricspb.Metric
	for i := 0; i < lg.options.ItemsPerBatch; i++ {

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

		batchIndex := atomic.AddUint64(&lg.batchesSent, 1)

		// Generate data points for the metric. We generate timeseries each containing
		// a single data points. This is the most typical payload composition since
		// monitoring libraries typically generated one data point at a time.
		for j := 0; j < dataPointsPerMetric; j++ {
			timeseries := &metricspb.TimeSeries{}

			startTime := time.Now()
			value := atomic.AddUint64(&lg.dataItemsSent, 1)

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

	metricData := consumerdata.MetricsData{
		Resource: resource,
		Metrics:  metrics,
	}

	err := metricSender.SendMetrics(metricData)
	if err == nil {
		lg.prevErr = nil
	} else if lg.prevErr == nil || lg.prevErr.Error() != err.Error() {
		lg.prevErr = err
		log.Printf("Cannot send metrics: %v", err)
	}
}

func (lg *LoadGenerator) generateMetrics() {

	metricSender := lg.sender.(MetricDataSender)

	// Generate 7 data points per metric.
	const dataPointsPerMetric = 7

	metricData := data.NewMetricData()
	metricData.ResourceMetrics().Resize(1)
	metricData.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	if lg.options.Attributes != nil {
		attrs := metricData.ResourceMetrics().At(0).Resource().Attributes()
		attrs.InitEmptyWithCapacity(len(lg.options.Attributes))
		for k, v := range lg.options.Attributes {
			attrs.UpsertString(k, v)
		}
	}
	metrics := metricData.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	metrics.Resize(lg.options.ItemsPerBatch)

	for i := 0; i < lg.options.ItemsPerBatch; i++ {
		metric := metrics.At(i)
		metricDescriptor := metric.MetricDescriptor()
		metricDescriptor.InitEmpty()
		metricDescriptor.SetName("load_generator_" + strconv.Itoa(i))
		metricDescriptor.SetDescription("Load Generator Counter #" + strconv.Itoa(i))
		metricDescriptor.SetType(pdata.MetricTypeGaugeInt64)

		batchIndex := atomic.AddUint64(&lg.batchesSent, 1)

		// Generate data points for the metric.
		metric.Int64DataPoints().Resize(dataPointsPerMetric)
		for j := 0; j < dataPointsPerMetric; j++ {
			dataPoint := metric.Int64DataPoints().At(j)
			dataPoint.SetStartTime(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
			value := atomic.AddUint64(&lg.dataItemsSent, 1)
			dataPoint.SetValue(int64(value))
			dataPoint.LabelsMap().InitFromMap(map[string]string{
				"item_index":  "item_" + strconv.Itoa(j),
				"batch_index": "batch_" + strconv.Itoa(int(batchIndex)),
			})
		}
	}

	err := metricSender.SendMetrics(metricData)

	if err == nil {
		lg.prevErr = nil
	} else if lg.prevErr == nil || lg.prevErr.Error() != err.Error() {
		lg.prevErr = err
		log.Printf("Cannot send metrics: %v", err)
	}
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
