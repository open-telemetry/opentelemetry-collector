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
	"fmt"
	"testing"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/translator/conventions"
)

func TestTraceProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.TraceConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumerOld{}
	}

	tfc := NewTraceFanOutConnectorOld(processors)
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
		m := p.(*mockTraceConsumerOld)
		assert.Equal(t, wantSpansCount, m.TotalSpans)
		assert.True(t, td.Resource == m.Traces[0].Resource)
	}
}

func TestTraceProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.TraceConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumerOld{}
	}

	// Make one processor return error
	processors[1].(*mockTraceConsumerOld).MustFail = true

	tfc := NewTraceFanOutConnectorOld(processors)
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
		m := p.(*mockTraceConsumerOld)
		if m.TotalSpans != wantSpansCount {
			t.Errorf("Wanted %d spans for every processor but got %d", wantSpansCount, m.TotalSpans)
			return
		}
	}
}

func TestMetricsProcessorMultiplexing(t *testing.T) {
	processors := make([]consumer.MetricsConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumerOld{}
	}

	mfc := NewMetricsFanOutConnectorOld(processors)
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
		m := p.(*mockMetricsConsumerOld)
		assert.Equal(t, wantMetricsCount, m.TotalMetrics)
		assert.True(t, md.Resource == m.Metrics[0].Resource)
	}
}

func TestMetricsProcessorWhenOneErrors(t *testing.T) {
	processors := make([]consumer.MetricsConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumerOld{}
	}

	// Make one processor return error
	processors[1].(*mockMetricsConsumerOld).MustFail = true

	mfc := NewMetricsFanOutConnectorOld(processors)
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
		m := p.(*mockMetricsConsumerOld)
		if m.TotalMetrics != wantMetricsCount {
			t.Errorf("Wanted %d metrics for every processor but got %d", wantMetricsCount, m.TotalMetrics)
			return
		}
	}
}

func TestCreateTraceFanOutConnectorWithConvertion(t *testing.T) {
	traceConsumerOld := &mockTraceConsumerOld{}
	traceConsumer := &mockTraceConsumer{}
	processors := []consumer.TraceConsumerBase{
		traceConsumerOld,
		traceConsumer,
	}

	resourceTypeName := "good-resource"

	td := pdata.NewTraces()
	rss := td.ResourceSpans()
	rss.Resize(1)
	rs0 := rss.At(0)
	res := rs0.Resource()
	res.InitEmpty()
	res.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.OCAttributeResourceType: pdata.NewAttributeValueString(resourceTypeName),
	})
	rs0.InstrumentationLibrarySpans().Resize(1)
	rs0.InstrumentationLibrarySpans().At(0).Spans().Resize(3)

	tfc := CreateTraceFanOutConnector(processors).(consumer.TraceConsumer)

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		err := tfc.ConsumeTraces(context.Background(), td)
		assert.NoError(t, err)
	}

	assert.Equal(t, wantSpansCount, traceConsumerOld.TotalSpans)
	assert.Equal(t, resourceTypeName, traceConsumerOld.Traces[0].Resource.Type)

	assert.Equal(t, wantSpansCount, traceConsumer.TotalSpans)
	assert.Equal(t, pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		conventions.OCAttributeResourceType: pdata.NewAttributeValueString(resourceTypeName),
	}), traceConsumer.Traces[0].ResourceSpans().At(0).Resource().Attributes())
}

func TestCreateMetricsFanOutConnectorWithConvertion(t *testing.T) {
	metricsConsumerOld := &mockMetricsConsumerOld{}
	metricsConsumer := &mockMetricsConsumer{}
	processors := []consumer.MetricsConsumerBase{
		metricsConsumerOld,
		metricsConsumer,
	}

	resourceTypeName := "good-resource"

	md := data.NewMetricData()
	rms := md.ResourceMetrics()
	rms.Resize(1)
	rm0 := rms.At(0)
	res := rm0.Resource()
	res.InitEmpty()
	res.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		conventions.OCAttributeResourceType: pdata.NewAttributeValueString(resourceTypeName),
	})
	rm0.InstrumentationLibraryMetrics().Resize(1)
	rm0.InstrumentationLibraryMetrics().At(0).Metrics().Resize(4)

	mfc := CreateMetricsFanOutConnector(processors).(consumer.MetricsConsumer)

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), pdatautil.MetricsFromInternalMetrics(md))
		assert.NoError(t, err)
	}

	assert.Equal(t, wantSpansCount, metricsConsumerOld.TotalMetrics)
	assert.Equal(t, resourceTypeName, metricsConsumerOld.Metrics[0].Resource.Type)

	assert.Equal(t, wantSpansCount, metricsConsumer.TotalMetrics)
	assert.Equal(t, pdata.NewAttributeMap().InitFromMap(map[string]pdata.AttributeValue{
		conventions.OCAttributeResourceType: pdata.NewAttributeValueString(resourceTypeName),
	}), pdatautil.MetricsToInternalMetrics(*metricsConsumer.Metrics[0]).ResourceMetrics().At(0).Resource().Attributes())
}

type mockTraceConsumerOld struct {
	Traces     []*consumerdata.TraceData
	TotalSpans int
	MustFail   bool
}

var _ consumer.TraceConsumerOld = &mockTraceConsumerOld{}

func (p *mockTraceConsumerOld) ConsumeTraceData(_ context.Context, td consumerdata.TraceData) error {
	p.Traces = append(p.Traces, &td)
	p.TotalSpans += len(td.Spans)
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}

	return nil
}

type mockTraceConsumer struct {
	Traces     []*pdata.Traces
	TotalSpans int
	MustFail   bool
}

var _ consumer.TraceConsumer = &mockTraceConsumer{}

func (p *mockTraceConsumer) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	p.Traces = append(p.Traces, &td)
	p.TotalSpans += td.SpanCount()
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}
	return nil

}

type mockMetricsConsumerOld struct {
	Metrics      []*consumerdata.MetricsData
	TotalMetrics int
	MustFail     bool
}

var _ consumer.MetricsConsumerOld = &mockMetricsConsumerOld{}

func (p *mockMetricsConsumerOld) ConsumeMetricsData(_ context.Context, md consumerdata.MetricsData) error {
	p.Metrics = append(p.Metrics, &md)
	p.TotalMetrics += len(md.Metrics)
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}

	return nil
}

type mockMetricsConsumer struct {
	Metrics      []*pdata.Metrics
	TotalMetrics int
	MustFail     bool
}

var _ consumer.MetricsConsumer = &mockMetricsConsumer{}

func (p *mockMetricsConsumer) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	p.Metrics = append(p.Metrics, &md)
	p.TotalMetrics += pdatautil.MetricCount(md)
	if p.MustFail {
		return fmt.Errorf("this processor must fail")
	}
	return nil
}
