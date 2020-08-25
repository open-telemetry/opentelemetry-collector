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
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/data/testdata"
)

func TestTraceProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.TraceConsumer, 3)
	for i := range processors {
		processors[i] = new(exportertest.SinkTraceExporter)
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
		m := p.(*exportertest.SinkTraceExporter)
		assert.Equal(t, wantSpansCount, m.SpansCount())
		assert.EqualValues(t, td, m.AllTraces()[0])
	}
}

func TestTraceProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.TraceConsumer, 3)
	for i := range processors {
		processors[i] = new(exportertest.SinkTraceExporter)
	}

	// Make one processor return error
	processors[1].(*exportertest.SinkTraceExporter).SetConsumeTraceError(errors.New("my_error"))

	tfc := NewTracesFanOutConnector(processors)
	td := testdata.GenerateTraceDataOneSpan()

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		err := tfc.ConsumeTraces(context.Background(), td)
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
	}

	assert.Equal(t, 0, processors[1].(*exportertest.SinkTraceExporter).SpansCount())
	assert.Equal(t, wantSpansCount, processors[0].(*exportertest.SinkTraceExporter).SpansCount())
	assert.Equal(t, wantSpansCount, processors[2].(*exportertest.SinkTraceExporter).SpansCount())
}

func TestMetricsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = new(exportertest.SinkMetricsExporter)
	}

	mfc := NewMetricsFanOutConnector(processors)
	md := testdata.GenerateMetricDataOneMetric()
	pmd := pdatautil.MetricsFromInternalMetrics(md)

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), pmd)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*exportertest.SinkMetricsExporter)
		assert.Equal(t, wantMetricsCount, m.MetricsCount())
		assert.EqualValues(t, md, pdatautil.MetricsToInternalMetrics(m.AllMetrics()[0]))
	}
}

func TestMetricsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = new(exportertest.SinkMetricsExporter)
	}

	// Make one processor return error
	processors[1].(*exportertest.SinkMetricsExporter).SetConsumeMetricsError(errors.New("my_error"))

	mfc := NewMetricsFanOutConnector(processors)
	md := testdata.GenerateMetricDataOneMetric()
	pmd := pdatautil.MetricsFromInternalMetrics(md)

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), pmd)
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
	}

	assert.Equal(t, 0, processors[1].(*exportertest.SinkMetricsExporter).MetricsCount())
	assert.Equal(t, wantMetricsCount, processors[0].(*exportertest.SinkMetricsExporter).MetricsCount())
	assert.Equal(t, wantMetricsCount, processors[2].(*exportertest.SinkMetricsExporter).MetricsCount())
}
