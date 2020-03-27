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
	"math/rand"
	"strconv"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/internal/data/testdata"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func TestTraceProcessorCloningMultiplexingOld(t *testing.T) {
	processors := make([]consumer.TraceConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumerOld{}
	}

	tfc := NewTraceCloningFanOutConnectorOld(processors)
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

	for i, p := range processors {
		m := p.(*mockTraceConsumerOld)
		assert.Equal(t, wantSpansCount, m.TotalSpans)
		if i < len(processors)-1 {
			assert.True(t, td.Resource != m.Traces[0].Resource)
		} else {
			assert.True(t, td.Resource == m.Traces[0].Resource)
		}
		assert.EqualValues(t, td.Resource, m.Traces[0].Resource)
	}
}

func TestMetricsProcessorCloningMultiplexingOld(t *testing.T) {
	processors := make([]consumer.MetricsConsumerOld, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumerOld{}
	}

	mfc := NewMetricsCloningFanOutConnectorOld(processors)
	md := consumerdata.MetricsData{
		Metrics: make([]*metricspb.Metric, 7),
		Resource: &resourcepb.Resource{
			Type: "testtype",
		},
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

	for i, p := range processors {
		m := p.(*mockMetricsConsumerOld)
		assert.Equal(t, wantMetricsCount, m.TotalMetrics)
		if i < len(processors)-1 {
			assert.True(t, md.Resource != m.Metrics[0].Resource)
		} else {
			assert.True(t, md.Resource == m.Metrics[0].Resource)
		}
		assert.EqualValues(t, md.Resource, m.Metrics[0].Resource)
	}
}

func Benchmark100SpanCloneOld(b *testing.B) {

	b.StopTimer()

	name := tracepb.TruncatableString{Value: "testspanname"}
	traceData := &consumerdata.TraceData{
		SourceFormat: "test-source-format",
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{
				Name: "servicename",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "resourcetype",
		},
	}
	for i := 0; i < 100; i++ {
		span := &tracepb.Span{
			TraceId: genRandBytes(16),
			SpanId:  genRandBytes(8),
			Name:    &name,
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{},
			},
		}

		for j := 0; j < 5; j++ {
			span.Attributes.AttributeMap["intattr"+strconv.Itoa(j)] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_IntValue{IntValue: int64(i)},
			}
			span.Attributes.AttributeMap["strattr"+strconv.Itoa(j)] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: string(genRandBytes(20))},
				},
			}
		}

		traceData.Spans = append(traceData.Spans, span)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cloneTraceDataOld(traceData)
	}
}

func TestTraceProcessorCloningMultiplexing(t *testing.T) {
	processors := make([]consumer.TraceConsumer, 3)
	for i := range processors {
		processors[i] = &mockTraceConsumer{}
	}

	tfc := NewTraceCloningFanOutConnector(processors)
	td := testdata.GenerateTraceDataSameResourceTwoSpans()

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		err := tfc.ConsumeTrace(context.Background(), td)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for i, p := range processors {
		m := p.(*mockTraceConsumer)
		assert.Equal(t, wantSpansCount, m.TotalSpans)
		spanOrig := td.ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0).Spans().Get(0)
		spanClone := m.Traces[0].ResourceSpans().Get(0).InstrumentationLibrarySpans().Get(0).Spans().Get(0)
		if i < len(processors)-1 {
			assert.True(t, td.ResourceSpans().Get(0).Resource() != m.Traces[0].ResourceSpans().Get(0).Resource())
			assert.True(t, spanOrig != spanClone)
		} else {
			assert.True(t, td.ResourceSpans().Get(0).Resource() == m.Traces[0].ResourceSpans().Get(0).Resource())
			assert.True(t, spanOrig == spanClone)
		}
		assert.EqualValues(t, td.ResourceSpans().Get(0).Resource(), m.Traces[0].ResourceSpans().Get(0).Resource())
		assert.EqualValues(t, spanOrig, spanClone)
	}
}

func TestMetricsProcessorCloningMultiplexing(t *testing.T) {
	processors := make([]consumer.MetricsConsumer, 3)
	for i := range processors {
		processors[i] = &mockMetricsConsumer{}
	}

	mfc := NewMetricsCloningFanOutConnector(processors)
	md := testdata.GenerateMetricDataTwoMetrics()

	var wantMetricsCount = 0
	for i := 0; i < 2; i++ {
		wantMetricsCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), md)
		if err != nil {
			t.Errorf("Wanted nil got error")
			return
		}
	}

	for i, p := range processors {
		m := p.(*mockMetricsConsumer)
		assert.Equal(t, wantMetricsCount, m.TotalMetrics)
		metricOrig := md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
		metricClone := m.Metrics[0].ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0).Metrics().Get(0)
		if i < len(processors)-1 {
			assert.True(t, md.ResourceMetrics().Get(0).Resource() != m.Metrics[0].ResourceMetrics().Get(0).Resource())
			assert.True(t, metricOrig != metricClone)
		} else {
			assert.True(t, md.ResourceMetrics().Get(0).Resource() == m.Metrics[0].ResourceMetrics().Get(0).Resource())
			assert.True(t, metricOrig == metricClone)
		}
		assert.EqualValues(t, md.ResourceMetrics().Get(0).Resource(), m.Metrics[0].ResourceMetrics().Get(0).Resource())
		assert.EqualValues(t, metricOrig, metricClone)
	}
}

func Benchmark100SpanClone(b *testing.B) {

	b.StopTimer()

	name := tracepb.TruncatableString{Value: "testspanname"}
	traceData := &consumerdata.TraceData{
		SourceFormat: "test-source-format",
		Node: &commonpb.Node{
			ServiceInfo: &commonpb.ServiceInfo{
				Name: "servicename",
			},
		},
		Resource: &resourcepb.Resource{
			Type: "resourcetype",
		},
	}
	for i := 0; i < 100; i++ {
		span := &tracepb.Span{
			TraceId: genRandBytes(16),
			SpanId:  genRandBytes(8),
			Name:    &name,
			Attributes: &tracepb.Span_Attributes{
				AttributeMap: map[string]*tracepb.AttributeValue{},
			},
		}

		for j := 0; j < 5; j++ {
			span.Attributes.AttributeMap["intattr"+strconv.Itoa(j)] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_IntValue{IntValue: int64(i)},
			}
			span.Attributes.AttributeMap["strattr"+strconv.Itoa(j)] = &tracepb.AttributeValue{
				Value: &tracepb.AttributeValue_StringValue{
					StringValue: &tracepb.TruncatableString{Value: string(genRandBytes(20))},
				},
			}
		}

		traceData.Spans = append(traceData.Spans, span)
	}

	b.StartTimer()

	for i := 0; i < b.N; i++ {
		cloneTraceDataOld(traceData)
	}
}

func TestCreateTraceCloningFanOutConnectorWithConvertion(t *testing.T) {
	traceConsumerOld := &mockTraceConsumerOld{}
	traceConsumer := &mockTraceConsumer{}
	processors := []consumer.TraceConsumerBase{
		traceConsumerOld,
		traceConsumer,
	}

	resourceTypeName := "good-resource"

	td := testdata.GenerateTraceDataSameResourceTwoSpans()
	resource := td.ResourceSpans().Get(0).Resource()
	resource.Attributes().Upsert(data.NewAttributeKeyValueString(conventions.OCAttributeResourceType, resourceTypeName))

	tfc := CreateTraceCloningFanOutConnector(processors).(consumer.TraceConsumer)

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += td.SpanCount()
		err := tfc.ConsumeTrace(context.Background(), td)
		assert.NoError(t, err)
	}

	assert.Equal(t, wantSpansCount, traceConsumerOld.TotalSpans)
	assert.Equal(t, resourceTypeName, traceConsumerOld.Traces[0].Resource.Type)

	assert.Equal(t, wantSpansCount, traceConsumer.TotalSpans)
	assert.Equal(t, resource, traceConsumer.Traces[0].ResourceSpans().Get(0).Resource())
}

func TestCreateMetricsCloningFanOutConnectorWithConvertion(t *testing.T) {
	metricsConsumerOld := &mockMetricsConsumerOld{}
	metricsConsumer := &mockMetricsConsumer{}
	processors := []consumer.MetricsConsumerBase{
		metricsConsumerOld,
		metricsConsumer,
	}

	resourceTypeName := "good-resource"

	md := testdata.GenerateMetricDataTwoMetrics()
	resource := md.ResourceMetrics().Get(0).Resource()
	resource.Attributes().Upsert(data.NewAttributeKeyValueString(conventions.OCAttributeResourceType, resourceTypeName))

	mfc := CreateMetricsCloningFanOutConnector(processors).(consumer.MetricsConsumer)

	var wantSpansCount = 0
	for i := 0; i < 2; i++ {
		wantSpansCount += md.MetricCount()
		err := mfc.ConsumeMetrics(context.Background(), md)
		assert.NoError(t, err)
	}

	assert.Equal(t, wantSpansCount, metricsConsumerOld.TotalMetrics)
	assert.Equal(t, resourceTypeName, metricsConsumerOld.Metrics[0].Resource.Type)

	assert.Equal(t, wantSpansCount, metricsConsumer.TotalMetrics)
	assert.Equal(t, resource, metricsConsumer.Metrics[0].ResourceMetrics().Get(0).Resource())
}

func genRandBytes(len int) []byte {
	b := make([]byte, len)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}
