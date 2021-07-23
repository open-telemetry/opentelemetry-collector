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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
)

func TestTracesProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	tfc := NewTraces([]consumer.Traces{nop})
	assert.Same(t, nop, tfc)
}

func TestTracesProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.Traces, 3)
	for i := range processors {
		processors[i] = new(consumertest.TracesSink)
	}

	tfc := NewTraces(processors)
	td := testdata.GenerateTracesOneSpan()

	var wantSpanCount = 0
	for i := 0; i < 2; i++ {
		wantSpanCount += td.SpanCount()
		err := tfc.ConsumeTraces(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*consumertest.TracesSink)
		assert.Equal(t, wantSpanCount, m.SpanCount())
		assert.EqualValues(t, td, m.AllTraces()[0])
	}
}

func TestTracesProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.Traces, 3)
	for i := range processors {
		processors[i] = new(consumertest.TracesSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewErr(errors.New("my error"))

	tfc := NewTraces(processors)
	td := testdata.GenerateTracesOneSpan()

	var wantSpanCount = 0
	for i := 0; i < 2; i++ {
		wantSpanCount += td.SpanCount()
		assert.Error(t, tfc.ConsumeTraces(context.Background(), td))
	}

	assert.Equal(t, wantSpanCount, processors[0].(*consumertest.TracesSink).SpanCount())
	assert.Equal(t, wantSpanCount, processors[2].(*consumertest.TracesSink).SpanCount())
}

func TestMetricsProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	mfc := NewMetrics([]consumer.Metrics{nop})
	assert.Same(t, nop, mfc)
}

func TestMetricsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.Metrics, 3)
	for i := range processors {
		processors[i] = new(consumertest.MetricsSink)
	}

	mfc := NewMetrics(processors)
	md := testdata.GenerateMetricsOneMetric()

	var wantDataPointCount = 0
	for i := 0; i < 2; i++ {
		wantDataPointCount += md.DataPointCount()
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*consumertest.MetricsSink)
		assert.Equal(t, wantDataPointCount, m.DataPointCount())
		assert.EqualValues(t, md, m.AllMetrics()[0])
	}
}

func TestMetricsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.Metrics, 3)
	for i := range processors {
		processors[i] = new(consumertest.MetricsSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewErr(errors.New("my error"))

	mfc := NewMetrics(processors)
	md := testdata.GenerateMetricsOneMetric()

	var wantDataPointCount = 0
	for i := 0; i < 2; i++ {
		wantDataPointCount += md.DataPointCount()
		assert.Error(t, mfc.ConsumeMetrics(context.Background(), md))
	}

	assert.Equal(t, wantDataPointCount, processors[0].(*consumertest.MetricsSink).DataPointCount())
	assert.Equal(t, wantDataPointCount, processors[2].(*consumertest.MetricsSink).DataPointCount())
}

func TestLogsProcessorNotMultiplexing(t *testing.T) {
	nop := consumertest.NewNop()
	lfc := NewLogs([]consumer.Logs{nop})
	assert.Same(t, nop, lfc)
}

func TestLogsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.Logs, 3)
	for i := range processors {
		processors[i] = new(consumertest.LogsSink)
	}

	lfc := NewLogs(processors)
	ld := testdata.GenerateLogsOneLogRecord()

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
		assert.Equal(t, wantMetricsCount, m.LogRecordCount())
		assert.EqualValues(t, ld, m.AllLogs()[0])
	}
}

func TestLogsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.Logs, 3)
	for i := range processors {
		processors[i] = new(consumertest.LogsSink)
	}

	// Make one processor return error
	processors[1] = consumertest.NewErr(errors.New("my error"))

	lfc := NewLogs(processors)
	ld := testdata.GenerateLogsOneLogRecord()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += ld.LogRecordCount()
		assert.Error(t, lfc.ConsumeLogs(context.Background(), ld))
	}

	assert.Equal(t, wantMetricsCount, processors[0].(*consumertest.LogsSink).LogRecordCount())
	assert.Equal(t, wantMetricsCount, processors[2].(*consumertest.LogsSink).LogRecordCount())
}
