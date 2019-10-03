// Copyright 2019, OpenTelemetry Authors
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

package processor

import (
	"context"
	"fmt"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestTraceProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.TraceConsumer, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumer{}
	}

	tfc := NewTraceFanOutConnector(processors)
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 7),
		Resource: &resourcepb.Resource{
			Type: "testtype",
		},
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += len(td.Spans)
		err := tfc.ConsumeTraceData(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*mockTraceConsumer)
		assert.Equal(t, wantSpansCount, m.TotalSpans)
		assert.True(t, td.Resource == m.Traces[0].Resource)
	}
}

func TestTraceProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.TraceConsumer, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumer{}
	}

	// Make one processor return error
	processors[1].(*mockTraceConsumer).MustFail = true

	tfc := NewTraceFanOutConnector(processors)
	td := consumerdata.TraceData{
		Spans: make([]*tracepb.Span, 5),
	}

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += len(td.Spans)
		err := tfc.ConsumeTraceData(context.Background(), td)
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
	}

	for _, p := range processors {
		m := p.(*mockTraceConsumer)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d spans for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
}

func TestMetricsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumer{}
	}

	mfc := NewMetricsFanOutConnector(processors)
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
	}

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += len(md.Metrics)
		err := mfc.ConsumeMetricsData(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for _, p := range processors {
		m := p.(*mockMetricsConsumer)
		assert.Equal(t, wantMetricsCount, m.TotalMetrics)
		assert.True(t, md.Resource == m.Metrics[0].Resource)
	}
}

func TestMetricsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumer{}
	}

	// Make one processor return error
	processors[1].(*mockMetricsConsumer).MustFail = true

	mfc := NewMetricsFanOutConnector(processors)
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 5),
	}

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += len(md.Metrics)
		err := mfc.ConsumeMetricsData(context.Background(), md)
		if err == nil {
			t.Errorf("Wanted error got nil")
			return
		}
	}

	for _, p := range processors {
		m := p.(*mockMetricsConsumer)
		if m.TotalMetrics != wantMetricsCount {
			t.Errorf("Wanted %d metrics for every processor but got %d", wantMetricsCount, m.TotalMetrics)
			return
		}
	}
}

type mockTraceConsumer struct {
	Traces     []*consumerdata.TraceData
	TotalSpans int
	MustFail   bool
}

var _ consumer.TraceConsumer = &mockTraceConsumer{}

func (p *mockTraceConsumer) ConsumeTraceData(ctx context.Context, td consumerdata.TraceData) error {
	p.Traces = append(p.Traces, &td)
	p.TotalSpans += len(td.Spans)
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}

	return nil
}

type mockMetricsConsumer struct {
	Metrics      []*consumerdata.MetricsData
	TotalMetrics int
	MustFail     bool
}

var _ consumer.MetricsConsumer = &mockMetricsConsumer{}

func (p *mockMetricsConsumer) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	p.Metrics = append(p.Metrics, &md)
	p.TotalMetrics += len(md.Metrics)
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}

	return nil
}
