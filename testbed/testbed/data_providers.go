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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/internal/goldendataset"
	"go.opentelemetry.io/collector/internal/idutils"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

// DataProvider defines the interface for generators of test data used to drive various end-to-end tests.
type DataProvider interface {
	// SetLoadGeneratorCounters supplies pointers to LoadGenerator counters.
	// The data provider implementation should increment these as it generates data.
	SetLoadGeneratorCounters(dataItemsGenerated *atomic.Uint64)
	// GenerateTraces returns an internal Traces instance with an OTLP ResourceSpans slice populated with test data.
	GenerateTraces() (pdata.Traces, bool)
	// GenerateMetrics returns an internal MetricData instance with an OTLP ResourceMetrics slice of test data.
	GenerateMetrics() (pdata.Metrics, bool)
	// GenerateLogs returns the internal pdata.Logs format
	GenerateLogs() (pdata.Logs, bool)
}

// perfTestDataProvider in an implementation of the DataProvider for use in performance tests.
// Tracing IDs are based on the incremented batch and data items counters.
type perfTestDataProvider struct {
	options            LoadOptions
	traceIDSequence    atomic.Uint64
	dataItemsGenerated *atomic.Uint64
}

// NewPerfTestDataProvider creates an instance of perfTestDataProvider which generates test data based on the sizes
// specified in the supplied LoadOptions.
func NewPerfTestDataProvider(options LoadOptions) DataProvider {
	return &perfTestDataProvider{
		options: options,
	}
}

func (dp *perfTestDataProvider) SetLoadGeneratorCounters(dataItemsGenerated *atomic.Uint64) {
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *perfTestDataProvider) GenerateTraces() (pdata.Traces, bool) {
	traceData := pdata.NewTraces()
	spans := traceData.ResourceSpans().AppendEmpty().InstrumentationLibrarySpans().AppendEmpty().Spans()
	spans.EnsureCapacity(dp.options.ItemsPerBatch)

	traceID := dp.traceIDSequence.Inc()
	for i := 0; i < dp.options.ItemsPerBatch; i++ {

		startTime := time.Now()
		endTime := startTime.Add(time.Millisecond)

		spanID := dp.dataItemsGenerated.Inc()

		span := spans.AppendEmpty()

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

func (dp *perfTestDataProvider) GenerateMetrics() (pdata.Metrics, bool) {
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
	metrics.EnsureCapacity(dp.options.ItemsPerBatch)

	for i := 0; i < dp.options.ItemsPerBatch; i++ {
		metric := metrics.AppendEmpty()
		metric.SetName("load_generator_" + strconv.Itoa(i))
		metric.SetDescription("Load Generator Counter #" + strconv.Itoa(i))
		metric.SetUnit("1")
		metric.SetDataType(pdata.MetricDataTypeIntGauge)

		batchIndex := dp.traceIDSequence.Inc()

		dps := metric.IntGauge().DataPoints()
		// Generate data points for the metric.
		dps.EnsureCapacity(dataPointsPerMetric)
		for j := 0; j < dataPointsPerMetric; j++ {
			dataPoint := dps.AppendEmpty()
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

func (dp *perfTestDataProvider) GenerateLogs() (pdata.Logs, bool) {
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
	logRecords.EnsureCapacity(dp.options.ItemsPerBatch)

	now := pdata.TimestampFromTime(time.Now())

	batchIndex := dp.traceIDSequence.Inc()

	for i := 0; i < dp.options.ItemsPerBatch; i++ {
		itemIndex := dp.dataItemsGenerated.Inc()
		record := logRecords.AppendEmpty()
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

// goldenDataProvider is an implementation of DataProvider for use in correctness tests.
// Provided data from the "Golden" dataset generated using pairwise combinatorial testing techniques.
type goldenDataProvider struct {
	tracePairsFile     string
	spanPairsFile      string
	dataItemsGenerated *atomic.Uint64

	tracesGenerated []pdata.Traces
	tracesIndex     int

	metricPairsFile  string
	metricsGenerated []pdata.Metrics
	metricsIndex     int
}

// NewGoldenDataProvider creates a new instance of goldenDataProvider which generates test data based
// on the pairwise combinations specified in the tracePairsFile and spanPairsFile input variables.
func NewGoldenDataProvider(tracePairsFile string, spanPairsFile string, metricPairsFile string) DataProvider {
	return &goldenDataProvider{
		tracePairsFile:  tracePairsFile,
		spanPairsFile:   spanPairsFile,
		metricPairsFile: metricPairsFile,
	}
}

func (dp *goldenDataProvider) SetLoadGeneratorCounters(dataItemsGenerated *atomic.Uint64) {
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *goldenDataProvider) GenerateTraces() (pdata.Traces, bool) {
	if dp.tracesGenerated == nil {
		var err error
		dp.tracesGenerated, err = goldendataset.GenerateTraces(dp.tracePairsFile, dp.spanPairsFile)
		if err != nil {
			log.Printf("cannot generate traces: %s", err)
			dp.tracesGenerated = nil
		}
	}
	if dp.tracesIndex >= len(dp.tracesGenerated) {
		return pdata.NewTraces(), true
	}
	td := dp.tracesGenerated[dp.tracesIndex]
	dp.tracesIndex++
	dp.dataItemsGenerated.Add(uint64(td.SpanCount()))
	return td, false
}

func (dp *goldenDataProvider) GenerateMetrics() (pdata.Metrics, bool) {
	if dp.metricsGenerated == nil {
		var err error
		dp.metricsGenerated, err = goldendataset.GenerateMetrics(dp.metricPairsFile)
		if err != nil {
			log.Printf("cannot generate metrics: %s", err)
		}
	}
	if dp.metricsIndex == len(dp.metricsGenerated) {
		return pdata.Metrics{}, true
	}
	pdm := dp.metricsGenerated[dp.metricsIndex]
	dp.metricsIndex++
	dp.dataItemsGenerated.Add(uint64(pdm.DataPointCount()))
	return pdm, false
}

func (dp *goldenDataProvider) GenerateLogs() (pdata.Logs, bool) {
	return pdata.NewLogs(), true
}

// FileDataProvider in an implementation of the DataProvider for use in performance tests.
// The data to send is loaded from a file. The file should contain one JSON-encoded
// Export*ServiceRequest Protobuf message. The file can be recorded using the "file"
// exporter (note: "file" exporter writes one JSON message per line, FileDataProvider
// expects just a single JSON message in the entire file).
type FileDataProvider struct {
	dataItemsGenerated *atomic.Uint64
	logs               pdata.Logs
	metrics            pdata.Metrics
	traces             pdata.Traces
	ItemsPerBatch      int
}

// NewFileDataProvider creates an instance of FileDataProvider which generates test data
// loaded from a file.
func NewFileDataProvider(filePath string, dataType config.DataType) (*FileDataProvider, error) {
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var buf []byte
	buf, err = ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	dp := &FileDataProvider{}
	// Load the message from the file and count the data points.
	switch dataType {
	case config.TracesDataType:
		if dp.traces, err = otlp.NewJSONTracesUnmarshaler().UnmarshalTraces(buf); err != nil {
			return nil, err
		}
		dp.ItemsPerBatch = dp.traces.SpanCount()
	case config.MetricsDataType:
		if dp.metrics, err = otlp.NewJSONMetricsUnmarshaler().UnmarshalMetrics(buf); err != nil {
			return nil, err
		}
		dp.ItemsPerBatch = dp.metrics.DataPointCount()
	case config.LogsDataType:
		if dp.logs, err = otlp.NewJSONLogsUnmarshaler().UnmarshalLogs(buf); err != nil {
			return nil, err
		}
		dp.ItemsPerBatch = dp.logs.LogRecordCount()
	}

	return dp, nil
}

func (dp *FileDataProvider) SetLoadGeneratorCounters(dataItemsGenerated *atomic.Uint64) {
	dp.dataItemsGenerated = dataItemsGenerated
}

func (dp *FileDataProvider) GenerateTraces() (pdata.Traces, bool) {
	dp.dataItemsGenerated.Add(uint64(dp.ItemsPerBatch))
	return dp.traces, false
}

func (dp *FileDataProvider) GenerateMetrics() (pdata.Metrics, bool) {
	dp.dataItemsGenerated.Add(uint64(dp.ItemsPerBatch))
	return dp.metrics, false
}

func (dp *FileDataProvider) GenerateLogs() (pdata.Logs, bool) {
	dp.dataItemsGenerated.Add(uint64(dp.ItemsPerBatch))
	return dp.logs, false
}
