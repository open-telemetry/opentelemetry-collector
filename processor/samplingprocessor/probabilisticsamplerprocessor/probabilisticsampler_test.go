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

package probabilisticsamplerprocessor

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"testing"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	tracepb "github.com/census-instrumentation/opencensus-proto/gen-go/trace/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/processor"
	tracetranslator "github.com/open-telemetry/opentelemetry-collector/translator/trace"
)

func TestNewTraceProcessor(t *testing.T) {
	tests := []struct {
		name         string
		nextConsumer consumer.TraceConsumer
		cfg          Config
		want         processor.TraceProcessor
		wantErr      bool
	}{
		{
			name:    "nil_nextConsumer",
			wantErr: true,
		},
		{
			name:         "happy_path",
			nextConsumer: &exportertest.SinkTraceExporter{},
			cfg: Config{
				SamplingPercentage: 15.5,
			},
			want: &tracesamplerprocessor{
				nextConsumer: &exportertest.SinkTraceExporter{},
			},
		},
		{
			name:         "happy_path_hash_seed",
			nextConsumer: &exportertest.SinkTraceExporter{},
			cfg: Config{
				SamplingPercentage: 13.33,
				HashSeed:           4321,
			},
			want: &tracesamplerprocessor{
				nextConsumer: &exportertest.SinkTraceExporter{},
				hashSeed:     4321,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.wantErr {
				// The truncation below with uint32 cannot be defined at initialization (compiler error), performing it at runtime.
				tt.want.(*tracesamplerprocessor).scaledSamplingRate = uint32(tt.cfg.SamplingPercentage * percentageScaleFactor)
			}
			got, err := NewTraceProcessor(tt.nextConsumer, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTraceProcessor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTraceProcessor() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_tracesamplerprocessor_SamplingPercentageRange checks for different sampling rates and ensures
// that they are within acceptable deltas.
func Test_tracesamplerprocessor_SamplingPercentageRange(t *testing.T) {
	tests := []struct {
		name              string
		cfg               Config
		numBatches        int
		numTracesPerBatch int
		acceptableDelta   float64
	}{
		{
			name: "random_sampling_tiny",
			cfg: Config{
				SamplingPercentage: 0.03,
			},
			numBatches:        1e5,
			numTracesPerBatch: 2,
			acceptableDelta:   0.01,
		},
		{
			name: "random_sampling_small",
			cfg: Config{
				SamplingPercentage: 5,
			},
			numBatches:        1e5,
			numTracesPerBatch: 2,
			acceptableDelta:   0.01,
		},
		{
			name: "random_sampling_medium",
			cfg: Config{
				SamplingPercentage: 50.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 4,
			acceptableDelta:   0.1,
		},
		{
			name: "random_sampling_high",
			cfg: Config{
				SamplingPercentage: 90.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 1,
			acceptableDelta:   0.2,
		},
		{
			name: "random_sampling_all",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			numBatches:        1e5,
			numTracesPerBatch: 1,
			acceptableDelta:   0.0,
		},
	}
	const testSvcName = "test-svc"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &exportertest.SinkTraceExporter{}
			tsp, err := NewTraceProcessor(sink, tt.cfg)
			if err != nil {
				t.Errorf("error when creating tracesamplerprocessor: %v", err)
				return
			}
			for _, td := range genRandomTestData(tt.numBatches, tt.numTracesPerBatch, testSvcName) {
				if err := tsp.ConsumeTraceData(context.Background(), td); err != nil {
					t.Errorf("tracesamplerprocessor.ConsumeTraceData() error = %v", err)
					return
				}
			}
			_, sampled := assertSampledData(t, sink.AllTraces(), testSvcName)
			actualPercentageSamplingPercentage := float32(sampled) / float32(tt.numBatches*tt.numTracesPerBatch) * 100.0
			delta := math.Abs(float64(actualPercentageSamplingPercentage - tt.cfg.SamplingPercentage))
			if delta > tt.acceptableDelta {
				t.Errorf(
					"got %f percentage sampling rate, want %f (allowed delta is %f but got %f)",
					actualPercentageSamplingPercentage,
					tt.cfg.SamplingPercentage,
					tt.acceptableDelta,
					delta,
				)
			}
		})
	}
}

// Test_tracesamplerprocessor_SpanSamplingPriority checks if handling of "sampling.priority" is correct.
func Test_tracesamplerprocessor_SpanSamplingPriority(t *testing.T) {
	singleSpanWithAttrib := func(key string, attribValue *tracepb.AttributeValue) consumerdata.TraceData {
		return consumerdata.TraceData{
			Spans: []*tracepb.Span{
				{
					Attributes: &tracepb.Span_Attributes{
						AttributeMap: map[string]*tracepb.AttributeValue{
							key: {Value: attribValue.Value},
						},
					},
				},
			},
		}
	}
	tests := []struct {
		name    string
		cfg     Config
		td      consumerdata.TraceData
		sampled bool
	}{
		{
			name: "must_sample",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_IntValue{IntValue: 2}}),
			sampled: true,
		},
		{
			name: "must_sample_double",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 1}}),
			sampled: true,
		},
		{
			name: "must_sample_string",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "1"}}}),
			sampled: true,
		},
		{
			name: "must_not_sample",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_IntValue{IntValue: 0}}),
		},
		{
			name: "must_not_sample_double",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 0}}),
		},
		{
			name: "must_not_sample_string",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_StringValue{StringValue: &tracepb.TruncatableString{Value: "0"}}}),
		},
		{
			name: "defer_sample_expect_not_sampled",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"no.sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_IntValue{IntValue: 2}}),
		},
		{
			name: "defer_sample_expect_sampled",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"no.sampling.priority",
				&tracepb.AttributeValue{Value: &tracepb.AttributeValue_IntValue{IntValue: 2}}),
			sampled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &exportertest.SinkTraceExporter{}
			tsp, err := NewTraceProcessor(sink, tt.cfg)
			require.NoError(t, err)

			err = tsp.ConsumeTraceData(context.Background(), tt.td)
			require.NoError(t, err)

			sampledData := sink.AllTraces()
			require.Equal(t, 1, len(sampledData))
			assert.Equal(t, tt.sampled, len(sampledData[0].Spans) == 1)
		})
	}
}

// Test_parseSpanSamplingPriority ensures that the function parsing the attributes is taking "sampling.priority"
// attribute correctly.
func Test_parseSpanSamplingPriority(t *testing.T) {
	tests := []struct {
		name string
		span *tracepb.Span
		want samplingPriority
	}{
		{
			name: "nil_span",
			span: &tracepb.Span{},
			want: deferDecision,
		},
		{
			name: "nil_attributes",
			span: &tracepb.Span{},
			want: deferDecision,
		},
		{
			name: "nil_attribute_map",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{},
			},
			want: deferDecision,
		},
		{
			name: "empty_attribute_map",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{},
				},
			},
			want: deferDecision,
		},
		{
			name: "no_sampling_priority",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"key": {Value: &tracepb.AttributeValue_BoolValue{BoolValue: true}},
					},
				},
			},
			want: deferDecision,
		},
		{
			name: "sampling_priority_int_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_IntValue{IntValue: 0}},
					},
				},
			},
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_int_gt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_IntValue{IntValue: 1}},
					},
				},
			},
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_int_lt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_IntValue{IntValue: -1}},
					},
				},
			},
			want: deferDecision,
		},
		{
			name: "sampling_priority_double_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_DoubleValue{}},
					},
				},
			},
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_double_gt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: 1}},
					},
				},
			},
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_double_lt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_DoubleValue{DoubleValue: -1}},
					},
				},
			},
			want: deferDecision,
		},
		{
			name: "sampling_priority_string_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "0.0"},
						}},
					},
				},
			},
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_string_gt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "0.5"},
						}},
					},
				},
			},
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_string_lt_zero",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "-0.5"},
						}},
					},
				},
			},
			want: deferDecision,
		},
		{
			name: "sampling_priority_string_NaN",
			span: &tracepb.Span{
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						"sampling.priority": {Value: &tracepb.AttributeValue_StringValue{
							StringValue: &tracepb.TruncatableString{Value: "NaN"},
						}},
					},
				},
			},
			want: deferDecision,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSpanSamplingPriority(tt.span))
		})
	}
}

// Test_hash ensures that the hash function supports different key lengths even if in
// practice it is only expected to receive keys with length 16 (trace id length in OC proto).
func Test_hash(t *testing.T) {
	// Statistically a random selection of such small number of keys should not result in
	// collisions, but, of course it is possible that they happen, a different random source
	// should avoid that.
	r := rand.New(rand.NewSource(1))
	fullKey := tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64())
	seen := make(map[uint32]bool)
	for i := 1; i <= len(fullKey); i++ {
		key := fullKey[:i]
		hash := hash(key, 1)
		require.False(t, seen[hash], "Unexpected duplicated hash")
		seen[hash] = true
	}
}

// genRandomTestData generates a slice of consumerdata.TraceData with the numBatches elements which one with
// numTracesPerBatch spans (ie.: each span has a different trace ID). All spans belong to the specified
// serviceName.
func genRandomTestData(numBatches, numTracesPerBatch int, serviceName string) (tdd []consumerdata.TraceData) {
	r := rand.New(rand.NewSource(1))

	for i := 0; i < numBatches; i++ {
		var spans []*tracepb.Span
		for j := 0; j < numTracesPerBatch; j++ {
			span := &tracepb.Span{
				TraceId: tracetranslator.UInt64ToByteTraceID(r.Uint64(), r.Uint64()),
				Attributes: &tracepb.Span_Attributes{
					AttributeMap: map[string]*tracepb.AttributeValue{
						tracetranslator.TagHTTPStatusCode: {
							Value: &tracepb.AttributeValue_IntValue{
								IntValue: 404,
							},
						},
						tracetranslator.TagHTTPStatusMsg: {
							Value: &tracepb.AttributeValue_StringValue{
								StringValue: &tracepb.TruncatableString{Value: "NotFound"},
							},
						},
					},
				},
			}
			spans = append(spans, span)
		}
		td := consumerdata.TraceData{
			Node: &commonpb.Node{
				ServiceInfo: &commonpb.ServiceInfo{Name: serviceName},
			},
			Spans: spans,
		}
		tdd = append(tdd, td)
	}

	return tdd
}

// assertSampledData checks for no repeated traceIDs and counts the number of spans on the sampled data for
// the given service.
func assertSampledData(t *testing.T, sampled []consumerdata.TraceData, serviceName string) (traceIDs map[string]bool, spanCount int) {
	traceIDs = make(map[string]bool)
	for _, td := range sampled {
		if processor.ServiceNameForNode(td.Node) != serviceName {
			continue
		}
		for _, span := range td.Spans {
			spanCount++
			key := string(span.TraceId)
			if traceIDs[key] {
				t.Errorf("same traceID used more than once %q", key)
				return
			}
			traceIDs[key] = true
		}
	}
	return
}
