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

package fanoutconsumer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTracesProcessorCloningNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewTracesCloning([]consumer.Traces{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesProcessorCloningMultiplexing(t *testing.T) {
	processors := make([]consumer.Traces, 3)
	for i := range processors {
		processors[i] = new(consumertest.TracesSink)
	}

	tfc := NewTracesCloning(processors)
	td := testdata.GenerateTracesTwoSpansSameResource()

	var wantSpanCount = 0
	for i := 0; i < 2; i++ {
		wantSpanCount += td.SpanCount()
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for i, p := range processors {
		m := p.(*consumertest.TracesSink)
		assert.Equal(t, wantSpanCount, m.SpanCount())
		spanOrig := td.ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
		allTraces := m.AllTraces()
		spanClone := allTraces[0].ResourceSpans().At(0).InstrumentationLibrarySpans().At(0).Spans().At(0)
		if i < len(processors)-1 {
			assert.True(t, td.ResourceSpans().At(0).Resource() != allTraces[0].ResourceSpans().At(0).Resource())
			assert.True(t, spanOrig != spanClone)
		} else {
			assert.True(t, td.ResourceSpans().At(0).Resource() == allTraces[0].ResourceSpans().At(0).Resource())
			assert.True(t, spanOrig == spanClone)
		}
		assert.EqualValues(t, td.ResourceSpans().At(0).Resource(), allTraces[0].ResourceSpans().At(0).Resource())
		assert.EqualValues(t, spanOrig, spanClone)
	}
}

func TestMetricsProcessorCloningNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	mfc := NewMetrics([]consumer.Metrics{nop})
	assert.Same(t, nop, mfc)
}

func TestMetricsProcessorCloningMultiplexing(t *testing.T) {
	processors := make([]consumer.Metrics, 3)
	for i := range processors {
		processors[i] = new(consumertest.MetricsSink)
	}

	mfc := NewMetricsCloning(processors)
	md := testdata.GeneratMetricsAllTypesWithSampleDatapoints()

	var wantDataPointCount = 0
	for i := 0; i < 2; i++ {
		wantDataPointCount += md.DataPointCount()
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for i, p := range processors {
		m := p.(*consumertest.MetricsSink)
		assert.Equal(t, wantDataPointCount, m.DataPointCount())
		metricOrig := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
		allMetrics := m.AllMetrics()
		metricClone := allMetrics[0].ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
		if i < len(processors)-1 {
			assert.True(t, md.ResourceMetrics().At(0).Resource() != allMetrics[0].ResourceMetrics().At(0).Resource())
			assert.True(t, metricOrig != metricClone)
		} else {
			assert.True(t, md.ResourceMetrics().At(0).Resource() == allMetrics[0].ResourceMetrics().At(0).Resource())
			assert.True(t, metricOrig == metricClone)
		}
		assert.EqualValues(t, md.ResourceMetrics().At(0).Resource(), allMetrics[0].ResourceMetrics().At(0).Resource())
		assert.EqualValues(t, metricOrig, metricClone)
	}
}

func TestLogsProcessorCloningNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewLogsCloning([]consumer.Logs{nop})
	assert.Same(t, nop, lfc)
}

func TestLogsProcessorCloningMultiplexing(t *testing.T) {
	processors := make([]consumer.Logs, 3)
	for i := range processors {
		processors[i] = new(consumertest.LogsSink)
	}

	mfc := NewLogsCloning(processors)
	ld := testdata.GenerateLogsOneLogRecord()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += ld.LogRecordCount()
		err := mfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for i, p := range processors {
		m := p.(*consumertest.LogsSink)
		assert.Equal(t, wantMetricsCount, m.LogRecordCount())
		metricOrig := ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0)
		allLogs := m.AllLogs()
		metricClone := allLogs[0].ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0)
		if i < len(processors)-1 {
			assert.True(t, ld.ResourceLogs().At(0).Resource() != allLogs[0].ResourceLogs().At(0).Resource())
			assert.True(t, metricOrig != metricClone)
		} else {
			assert.True(t, ld.ResourceLogs().At(0).Resource() == allLogs[0].ResourceLogs().At(0).Resource())
			assert.True(t, metricOrig == metricClone)
		}
		assert.EqualValues(t, ld.ResourceLogs().At(0).Resource(), allLogs[0].ResourceLogs().At(0).Resource())
		assert.EqualValues(t, metricOrig, metricClone)
	}
}
