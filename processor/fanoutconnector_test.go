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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTracesProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewTracesNop()
	tfc := NewTracesFanOutConnector([]consumer.TracesConsumer{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.TracesConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.TracesSink)
	}

	tfc := NewTracesFanOutConnector(processors)
	td := testdata.GenerateTraceDataOneSpan()

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*consumertest.TracesSink)
		assert.Equal(t, wantSpansCount, m.SpansCount())
		assert.EqualValues(t, td, m.AllTraces()[0])
	}
}

func TestTraceProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.TracesConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.TracesSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewTracesErr(errors.New("my error"))

	tfc := NewTracesFanOutConnector(processors)
	td := testdata.GenerateTraceDataOneSpan()

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		assert.Error(t, tfc.ConsumeTraces(context.Background(), td))
	}

	assert.Equal(t, wantSpansCount, processors[0].(*consumertest.TracesSink).SpansCount())
	assert.Equal(t, wantSpansCount, processors[2].(*consumertest.TracesSink).SpansCount())
}

func TestMetricsProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewMetricsNop()
	mfc := NewMetricsFanOutConnector([]consumer.MetricsConsumer{nop})
	assert.Same(t, nop, mfc)
}

func TestMetricsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.MetricsSink)
	}

	mfc := NewMetricsFanOutConnector(processors)
	md := testdata.GenerateMetricsOneMetric()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*consumertest.MetricsSink)
		assert.Equal(t, wantMetricsCount, m.MetricsCount())
		assert.EqualValues(t, md, m.AllMetrics()[0])
	}
}

func TestMetricsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.MetricsSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewMetricsErr(errors.New("my error"))

	mfc := NewMetricsFanOutConnector(processors)
	md := testdata.GenerateMetricsOneMetric()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += md.MetricCount()
		assert.Error(t, mfc.ConsumeMetrics(context.Background(), md))
	}

	assert.Equal(t, wantMetricsCount, processors[0].(*consumertest.MetricsSink).MetricsCount())
	assert.Equal(t, wantMetricsCount, processors[2].(*consumertest.MetricsSink).MetricsCount())
}

func TestLogsProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewLogsNop()
	lfc := NewLogsFanOutConnector([]consumer.LogsConsumer{nop})
	assert.Same(t, nop, lfc)
}

func TestLogsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.LogsConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.LogsSink)
	}

	lfc := NewLogsFanOutConnector(processors)
	ld := testdata.GenerateLogDataOneLog()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += ld.LogRecordCount()
		err := lfc.ConsumeLogs(context.Background(), ld)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*consumertest.LogsSink)
		assert.Equal(t, wantMetricsCount, m.LogRecordsCount())
		assert.EqualValues(t, ld, m.AllLogs()[0])
	}
}

func TestLogsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.LogsConsumer, 3)
	for i := range processors {
		processors[i] = new(consumertest.LogsSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewLogsErr(errors.New("my error"))

	lfc := NewLogsFanOutConnector(processors)
	ld := testdata.GenerateLogDataOneLog()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += ld.LogRecordCount()
		assert.Error(t, lfc.ConsumeLogs(context.Background(), ld))
	}

	assert.Equal(t, wantMetricsCount, processors[0].(*consumertest.LogsSink).LogRecordsCount())
	assert.Equal(t, wantMetricsCount, processors[2].(*consumertest.LogsSink).LogRecordsCount())
}
