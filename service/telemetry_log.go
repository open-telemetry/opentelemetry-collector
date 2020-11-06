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
package service

import (
	"bytes"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// metricsToLogExporter allows to periodically dump the values of metrics to logs.
type metricsToLogExporter struct {
	// logger to output to.
	logger *zap.Logger

	// pendingRecords that are accumulated since last output period.
	pendingRecords      metricLogRecords
	pendingRecordsMutex sync.Mutex

	// outputTicker timer to wait until all records are ready for this output period.
	outputTicker *time.Ticker

	// Last time the records were written to logger.
	lastLogged time.Time
}

const (
	// Minimum time interval between printing metric data to logs when debug logs
	// are disabled.
	nonDebugLogMinInterval = 5 * time.Minute

	// Time to wait for ExportView() calls to arrive for record batching purposes.
	outputBounceTimeInterval = 100 * time.Millisecond
)

// metricLogRecord contains a formatted metric data that is ready to be printed as a log
// record.
type metricLogRecord struct {
	name  string
	tags  string
	value string
}

type metricLogRecords []metricLogRecord

func (mls metricLogRecords) Len() int {
	return len(mls)
}

func (mls metricLogRecords) Less(i, j int) bool {
	c := strings.Compare(mls[i].tags, mls[j].tags)
	if c == 0 {
		return mls[i].name < mls[j].name
	}
	return c < 0
}

func (mls metricLogRecords) Swap(i, j int) {
	mls[i], mls[j] = mls[j], mls[i]
}

func newMetricsToLogExporter(logger *zap.Logger) *metricsToLogExporter {
	me := &metricsToLogExporter{
		logger:       logger,
		outputTicker: time.NewTicker(outputBounceTimeInterval),
	}
	me.outputTicker.Stop()
	go me.doPeriodicOutput()
	return me
}

func viewDataToRecord(viewData *view.Data) metricLogRecord {
	mlr := metricLogRecord{
		name: viewData.View.Name,
	}

	var buffer bytes.Buffer

	printer := message.NewPrinter(language.English)

	for _, row := range viewData.Rows {
		buffer.Reset()
		if len(row.Tags) > 0 {
			// Prepare list of tags in the form of key1=value1, key2=value2, ...
			for i, t := range row.Tags {
				if i > 0 {
					buffer.WriteString(", ")
				}
				buffer.WriteString(fmt.Sprintf("%v=%v", t.Key.Name(), t.Value))
			}
			mlr.tags = buffer.String()
		}

		// Format value of the metric as a string.
		buffer.Reset()
		switch v := row.Data.(type) {
		case *view.SumData:
			buffer.WriteString(floatToStr(printer, v.Value, true) + unitToStr(viewData.View.Measure))
		case *view.LastValueData:
			buffer.WriteString(floatToStr(printer, v.Value, true) + unitToStr(viewData.View.Measure))
		case *view.DistributionData:
			buffer.WriteString(floatToStr(printer, v.Min, false) + "/")
			buffer.WriteString(floatToStr(printer, v.Mean, false) + "/")
			buffer.WriteString(floatToStr(printer, v.Max, false) + unitToStr(viewData.View.Measure) + " (min/mean/max)")
			buffer.WriteString("\tOccurrences=" + strconv.FormatInt(v.Count, 10))
		default:
			buffer.WriteString("Unknown metric data type")
		}
		mlr.value = buffer.String()
	}

	return mlr
}

func (m *metricsToLogExporter) ExportView(viewData *view.Data) {
	if viewData == nil || viewData.View == nil {
		return
	}

	// Append a row to pending outputs.
	m.pendingRecordsMutex.Lock()
	m.pendingRecords = append(m.pendingRecords, viewDataToRecord(viewData))
	m.pendingRecordsMutex.Unlock()

	// Start time to wait for more ExportView() calls to accumulate the data in
	// pendingRecords and then output all accumulated. Normally ExportView() calls
	// come in batches that we want to output at once.
	m.outputTicker.Reset(outputBounceTimeInterval)
}

func floatToStr(printer *message.Printer, v float64, padRight bool) string {
	var pattern string
	if v == math.Ceil(v) {
		// It is an integer number, don't print fractional digits.
		if padRight {
			pattern = "%12.0f"
		} else {
			pattern = "%.0f"
		}
	} else {
		if padRight {
			pattern = "%12f"
		} else {
			pattern = "%f"
		}
	}

	return printer.Sprintf(pattern, v)
}

func unitToStr(measure stats.Measure) string {
	// "1" mean unity, i.e. unitless metric, don't print it.
	if measure.Unit() != "1" && measure.Unit() != "" {
		return " " + measure.Unit()
	}
	return ""
}

func recordsToStr(records metricLogRecords) string {
	const patternWithoutTags = "  %-50s| %s"

	var buffer strings.Builder
	buffer.WriteString("\n  Internal Metrics:\n")
	buffer.WriteString(fmt.Sprintf(patternWithoutTags, "Metric", "Value"))
	separator := "\n  --------------------------------------------------|--------------------------------"
	buffer.WriteString(separator)

	// Sort accumulated metrics so that we print related metrics close to each other.
	sort.Sort(records)

	// First print records that have no tags (sorting places them at the beginning of
	// pendingRecords slice.
	i := 0
	for ; i < len(records); i++ {
		mo := records[i]
		if mo.tags != "" {
			break
		}
		buffer.WriteString("\n")
		buffer.WriteString(fmt.Sprintf(patternWithoutTags, mo.name, mo.value))
	}
	buffer.WriteString(separator)

	// Now print records with tags, if there are any.
	if i < len(records) {
		const patternWithTags = "  %-50s| %-40s| %s"
		buffer.WriteString("\n\n")
		buffer.WriteString(fmt.Sprintf(patternWithTags, "Component/Dimensions", "Metric", "Value"))
		separator = "\n  --------------------------------------------------|-----------------------------------------|--------------------------------"
		buffer.WriteString(separator)
		for ; i < len(records); i++ {
			mo := records[i]
			buffer.WriteString("\n")
			buffer.WriteString(fmt.Sprintf(patternWithTags, mo.tags, mo.name, mo.value))
		}
		buffer.WriteString(separator)
	}
	return buffer.String()
}

func (m *metricsToLogExporter) doPeriodicOutput() {
	for range m.outputTicker.C {
		m.pendingRecordsMutex.Lock()

		now := time.Now()
		if m.logger.Core().Enabled(zapcore.DebugLevel) || now.Sub(m.lastLogged) >= nonDebugLogMinInterval {
			// Output if the minimum interval has passed since last output, or if
			// debug level logging is enabled.
			m.logger.Info(recordsToStr(m.pendingRecords))
			m.lastLogged = now
		}

		m.pendingRecords = []metricLogRecord{}
		m.pendingRecordsMutex.Unlock()

		m.outputTicker.Stop()
	}
}
