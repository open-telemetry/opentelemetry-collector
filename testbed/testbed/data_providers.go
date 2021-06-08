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

package testbed

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal"
	otlplogscol "go.opentelemetry.io/collector/internal/data/protogen/collector/logs/v1"
	otlpmetricscol "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlptracecol "go.opentelemetry.io/collector/internal/data/protogen/collector/trace/v1"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/idutils"
)

// DataProvider defines the interface for generators of test data used to drive various end-to-end tests.
type DataProvider interface {
	// SetLoadGeneratorCounters supplies pointers to LoadGenerator counters.
	// The data provider implementation should increment these as it generates data.
	SetLoadGeneratorCounters(batchesGenerated *atomic.Uint64, dataItemsGenerated *atomic.Uint64)
	// GenerateTraces returns an internal Traces instance with an OTLP ResourceSpans slice populated with test data.
	GenerateTraces() (pdata.Traces, bool)
	// GenerateMetrics returns an internal MetricData instance with an OTLP ResourceMetrics slice of test data.
	GenerateMetrics() (pdata.Metrics, bool)
	// GenerateLogs returns the internal pdata.Logs format
	GenerateLogs() (pdata.Logs, bool)
}

// PerfTestDataProvider in an implementation of the DataProvider for use in performance tests.
// Tracing IDs are based on the incremented batch and data items counters.
type PerfTestDataProvider struct {
	options            LoadOptions
	batchesGenerated   *atomic.Uint64
	dataItemsGenerated *atomic.Uint64
}

// NewPerfTestDataProvider creates an instance of PerfTestDataProvider which generates test data based on the sizes
// specified in the supplied LoadOptions.
func NewPerfTestDataProvider(options LoadOptions) *PerfTestDataProvider {
	return &PerfTestDataProvider{
		options: options,
	}
}

func (dp *PerfTestDataProvider) SetLoadGeneratorCounters(batchesGenerated *atomic.Uint64, dataItemsGenerated *atomic.Uint64) {
	dp.batchesGenerated = batchesGenerated
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *PerfTestDataProvider) GenerateTraces() (pdata.Traces, bool) {

	traceData := pdata.NewTraces()
	spans := traceData.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans()
	spans.Resize(dp.options.ItemsPerBatch)

	traceID := dp.batchesGenerated.Inc()
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Millisecond)

		spanID := dp.dataItemsGenerated.Inc()

		span := spans.At(i)

		// Create a span.
		span.SetTraceID(idutils.UInt64ToTraceID(0, traceID))
		span.SetSpanID(idutils.UInt64ToSpanID(spanID))
		span.SetName("load-generator-span")
		span.SetKind(pdata.SpanKindClient)
		attrs := span.Attributes()
		attrs.UpsertInt("load_generator.span_seq_num", int64(spanID))
		attrs.UpsertInt("load_generator.trace_seq_num", int64(traceID))
		// Additional attributes.
		for k, v := range dp.options.Attributes {
			attrs.UpsertString(k, v)
		}
		span.SetStartTimestamp(pdata.TimestampFromTime(startTime))
		span.SetEndTimestamp(pdata.TimestampFromTime(endTime))
	}
	return traceData, false
}

func (dp *PerfTestDataProvider) GenerateMetrics() (pdata.Metrics, bool) {

	// Generate 7 data points per metric.
	const dataPointsPerMetric = 7

	md := pdata.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	if dp.options.Attributes != nil {
		attrs := rm.Resource().Attributes()
		attrs.EnsureCapacity(len(dp.options.Attributes))
		for k, v := range dp.options.Attributes {
			attrs.UpsertString(k, v)
		}
	}
	metrics := rm.InstrumentationLibraryMetrics().AppendEmpty().Metrics()
	metrics.Resize(dp.options.ItemsPerBatch)

	for i := 0; i < dp.options.ItemsPerBatch; i++ {
		metric := metrics.At(i)
		metric.SetName("load_generator_" + strconv.Itoa(i))
		metric.SetDescription("Load Generator Counter #" + strconv.Itoa(i))
		metric.SetUnit("1")
		metric.SetDataType(pdata.MetricDataTypeIntGauge)

		batchIndex := dp.batchesGenerated.Inc()

		dps := metric.IntGauge().DataPoints()
		// Generate data points for the metric.
		dps.Resize(dataPointsPerMetric)
		for j := 0; j < dataPointsPerMetric; j++ {
			dataPoint := dps.At(j)
			dataPoint.SetStartTimestamp(pdata.TimestampFromTime(time.Now()))
			value := dp.dataItemsGenerated.Inc()
			dataPoint.SetValue(int64(value))
			dataPoint.LabelsMap().InitFromMap(map[string]string{
				"item_index":  "item_" + strconv.Itoa(j),
				"batch_index": "batch_" + strconv.Itoa(int(batchIndex)),
			})
		}
	}
	return md, false
}

func (dp *PerfTestDataProvider) GenerateLogs() (pdata.Logs, bool) {
	logs := pdata.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	if dp.options.Attributes != nil {
		attrs := rl.Resource().Attributes()
		attrs.EnsureCapacity(len(dp.options.Attributes))
		for k, v := range dp.options.Attributes {
			attrs.UpsertString(k, v)
		}
	}
	logRecords := rl.InstrumentationLibraryLogs().AppendEmpty().Logs()
	logRecords.Resize(dp.options.ItemsPerBatch)

	now := pdata.TimestampFromTime(time.Now())

	batchIndex := dp.batchesGenerated.Inc()

	for i := 0; i < dp.options.ItemsPerBatch; i++ {
		itemIndex := dp.dataItemsGenerated.Inc()
		record := logRecords.At(i)
		record.SetSeverityNumber(pdata.SeverityNumberINFO3)
		record.SetSeverityText("INFO3")
		record.SetName("load_generator_" + strconv.Itoa(i))
		record.Body().SetStringVal("Load Generator Counter #" + strconv.Itoa(i))
		record.SetFlags(uint32(2))
		record.SetTimestamp(now)

		attrs := record.Attributes()
		attrs.UpsertString("batch_index", "batch_"+strconv.Itoa(int(batchIndex)))
		attrs.UpsertString("item_index", "item_"+strconv.Itoa(int(itemIndex)))
		attrs.UpsertString("a", "test")
		attrs.UpsertDouble("b", 5.0)
		attrs.UpsertInt("c", 3)
		attrs.UpsertBool("d", true)
	}
	return logs, false
}

// GoldenDataProvider is an implementation of DataProvider for use in correctness tests.
// Provided data from the "Golden" dataset generated using pairwise combinatorial testing techniques.
type GoldenDataProvider struct {
	tracePairsFile     string
	spanPairsFile      string
	batchesGenerated   *atomic.Uint64
	dataItemsGenerated *atomic.Uint64

	tracesGenerated []pdata.Traces
	tracesIndex     int

	metricPairsFile  string
	metricsGenerated []pdata.Metrics
	metricsIndex     int
}

// NewGoldenDataProvider creates a new instance of GoldenDataProvider which generates test data based
// on the pairwise combinations specified in the tracePairsFile and spanPairsFile input variables.
func NewGoldenDataProvider(tracePairsFile string, spanPairsFile string, metricPairsFile string) *GoldenDataProvider {
	return &GoldenDataProvider{
		tracePairsFile:  tracePairsFile,
		spanPairsFile:   spanPairsFile,
		metricPairsFile: metricPairsFile,
	}
}

func (dp *GoldenDataProvider) SetLoadGeneratorCounters(batchesGenerated *atomic.Uint64, dataItemsGenerated *atomic.Uint64) {
	dp.batchesGenerated = batchesGenerated
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *GoldenDataProvider) GenerateTraces() (pdata.Traces, bool) {
	if dp.tracesGenerated == nil {
		var err error
		dp.tracesGenerated, err = goldendataset.GenerateTraces(dp.tracePairsFile, dp.spanPairsFile)
		if err != nil {
			log.Printf("cannot generate traces: %s", err)
			dp.tracesGenerated = nil
		}
	}
	dp.batchesGenerated.Inc()
	if dp.tracesIndex >= len(dp.tracesGenerated) {
		return pdata.NewTraces(), true
	}
	td := dp.tracesGenerated[dp.tracesIndex]
	dp.tracesIndex++
	dp.dataItemsGenerated.Add(uint64(td.SpanCount()))
	return td, false
}

func (dp *GoldenDataProvider) GenerateMetrics() (pdata.Metrics, bool) {
	if dp.metricsGenerated == nil {
		var err error
		dp.metricsGenerated, err = goldendataset.GenerateMetrics(dp.metricPairsFile)
		if err != nil {
			log.Printf("cannot generate metrics: %s", err)
		}
	}
	dp.batchesGenerated.Inc()
	if dp.metricsIndex == len(dp.metricsGenerated) {
		return pdata.Metrics{}, true
	}
	pdm := dp.metricsGenerated[dp.metricsIndex]
	dp.metricsIndex++
	_, dpCount := pdm.MetricAndDataPointCount()
	dp.dataItemsGenerated.Add(uint64(dpCount))
	return pdm, false
}

func (dp *GoldenDataProvider) GenerateLogs() (pdata.Logs, bool) {
	return pdata.NewLogs(), true
}

// FileDataProvider in an implementation of the DataProvider for use in performance tests.
// The data to send is loaded from a file. The file should contain one JSON-encoded
// Export*ServiceRequest Protobuf message. The file can be recorded using the "file"
// exporter (note: "file" exporter writes one JSON message per line, FileDataProvider
// expects just a single JSON message in the entire file).
type FileDataProvider struct {
	batchesGenerated   *atomic.Uint64
	dataItemsGenerated *atomic.Uint64
	message            proto.Message
	ItemsPerBatch      int
}

// NewFileDataProvider creates an instance of FileDataProvider which generates test data
// loaded from a file.
func NewFileDataProvider(filePath string, dataType config.DataType) (*FileDataProvider, error) {
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	var message proto.Message
	var dataPointCount int

	// Load the message from the file and count the data points.

	switch dataType {
	case config.TracesDataType:
		var msg otlptracecol.ExportTraceServiceRequest
		if err := protobufJSONUnmarshaler.Unmarshal(file, &msg); err != nil {
			return nil, err
		}
		message = &msg

		md := pdata.TracesFromInternalRep(internal.TracesFromOtlp(&msg))
		dataPointCount = md.SpanCount()

	case config.MetricsDataType:
		var msg otlpmetricscol.ExportMetricsServiceRequest
		if err := protobufJSONUnmarshaler.Unmarshal(file, &msg); err != nil {
			return nil, err
		}
		message = &msg

		md := pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(&msg))
		_, dataPointCount = md.MetricAndDataPointCount()

	case config.LogsDataType:
		var msg otlplogscol.ExportLogsServiceRequest
		if err := protobufJSONUnmarshaler.Unmarshal(file, &msg); err != nil {
			return nil, err
		}
		message = &msg

		md := pdata.LogsFromInternalRep(internal.LogsFromOtlp(&msg))
		dataPointCount = md.LogRecordCount()
	}

	return &FileDataProvider{
		message:       message,
		ItemsPerBatch: dataPointCount,
	}, nil
}

func (dp *FileDataProvider) SetLoadGeneratorCounters(batchesGenerated *atomic.Uint64, dataItemsGenerated *atomic.Uint64) {
	dp.batchesGenerated = batchesGenerated
	dp.dataItemsGenerated = dataItemsGenerated
}

// Marshaler configuration used for marhsaling Protobuf to JSON. Use default config.
var protobufJSONUnmarshaler = &jsonpb.Unmarshaler{}

func (dp *FileDataProvider) GenerateTraces() (pdata.Traces, bool) {
	// TODO: implement similar to GenerateMetrics.
	return pdata.NewTraces(), true
}

func (dp *FileDataProvider) GenerateMetrics() (pdata.Metrics, bool) {
	md := pdata.MetricsFromInternalRep(internal.MetricsFromOtlp(dp.message.(*otlpmetricscol.ExportMetricsServiceRequest)))
	dp.batchesGenerated.Inc()
	_, dataPointCount := md.MetricAndDataPointCount()
	dp.dataItemsGenerated.Add(uint64(dataPointCount))
	return md, false
}

func (dp *FileDataProvider) GenerateLogs() (pdata.Logs, bool) {
	// TODO: implement similar to GenerateMetrics.
	return pdata.NewLogs(), true
}
