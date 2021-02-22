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

package probabilisticsamplerprocessor

import (
	"context"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

func TestNewTraceProcessor(t *testing.T) {
	tests := []struct {
		name         string
		nextConsumer consumer.TracesConsumer
		cfg          Config
		want         component.TracesProcessor
		wantErr      bool
	}{
		{
			name:    "nil_nextConsumer",
			wantErr: true,
		},
		{
			name:         "happy_path",
			nextConsumer: consumertest.NewTracesNop(),
			cfg: Config{
				SamplingPercentage: 15.5,
			},
			want: &tracesamplerprocessor{
				nextConsumer: consumertest.NewTracesNop(),
			},
		},
		{
			name:         "happy_path_hash_seed",
			nextConsumer: consumertest.NewTracesNop(),
			cfg: Config{
				SamplingPercentage: 13.33,
				HashSeed:           4321,
			},
			want: &tracesamplerprocessor{
				nextConsumer: consumertest.NewTracesNop(),
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
			got, err := newTraceProcessor(tt.nextConsumer, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTraceProcessor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTraceProcessor() = %v, want %v", got, tt.want)
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
			sink := new(consumertest.TracesSink)
			tsp, err := newTraceProcessor(sink, tt.cfg)
			if err != nil {
				t.Errorf("error when creating tracesamplerprocessor: %v", err)
				return
			}
			for _, td := range genRandomTestData(tt.numBatches, tt.numTracesPerBatch, testSvcName, 1) {
				assert.NoError(t, tsp.ConsumeTraces(context.Background(), td))
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

// Test_tracesamplerprocessor_SamplingPercentageRange_MultipleResourceSpans checks for number of spans sent to xt consumer. This is to avoid duplicate spans
func Test_tracesamplerprocessor_SamplingPercentageRange_MultipleResourceSpans(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  Config
		numBatches           int
		numTracesPerBatch    int
		acceptableDelta      float64
		resourceSpanPerTrace int
	}{
		{
			name: "single_batch_single_trace_two_resource_spans",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			numBatches:           1,
			numTracesPerBatch:    1,
			acceptableDelta:      0.0,
			resourceSpanPerTrace: 2,
		},
		{
			name: "single_batch_two_traces_two_resource_spans",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			numBatches:           1,
			numTracesPerBatch:    2,
			acceptableDelta:      0.0,
			resourceSpanPerTrace: 2,
		},
	}
	const testSvcName = "test-svc"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			tsp, err := newTraceProcessor(sink, tt.cfg)
			if err != nil {
				t.Errorf("error when creating tracesamplerprocessor: %v", err)
				return
			}

			for _, td := range genRandomTestData(tt.numBatches, tt.numTracesPerBatch, testSvcName, tt.resourceSpanPerTrace) {
				assert.NoError(t, tsp.ConsumeTraces(context.Background(), td))
				assert.Equal(t, tt.resourceSpanPerTrace*tt.numTracesPerBatch, sink.SpansCount())
				sink.Reset()
			}

		})
	}
}

// Test_tracesamplerprocessor_SpanSamplingPriority checks if handling of "sampling.priority" is correct.
func Test_tracesamplerprocessor_SpanSamplingPriority(t *testing.T) {
	singleSpanWithAttrib := func(key string, attribValue pdata.AttributeValue) pdata.Traces {
		traces := pdata.NewTraces()
		traces.ResourceSpans().Resize(1)
		rs := traces.ResourceSpans().At(0)
		rs.InstrumentationLibrarySpans().Resize(1)
		instrLibrarySpans := rs.InstrumentationLibrarySpans().At(0)
		instrLibrarySpans.Spans().Append(getSpanWithAttributes(key, attribValue))
		return traces
	}
	tests := []struct {
		name    string
		cfg     Config
		td      pdata.Traces
		sampled bool
	}{
		{
			name: "must_sample",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueInt(2)),
			sampled: true,
		},
		{
			name: "must_sample_double",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueDouble(1)),
			sampled: true,
		},
		{
			name: "must_sample_string",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueString("1")),
			sampled: true,
		},
		{
			name: "must_not_sample",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueInt(0)),
		},
		{
			name: "must_not_sample_double",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueDouble(0)),
		},
		{
			name: "must_not_sample_string",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"sampling.priority",
				pdata.NewAttributeValueString("0")),
		},
		{
			name: "defer_sample_expect_not_sampled",
			cfg: Config{
				SamplingPercentage: 0.0,
			},
			td: singleSpanWithAttrib(
				"no.sampling.priority",
				pdata.NewAttributeValueInt(2)),
		},
		{
			name: "defer_sample_expect_sampled",
			cfg: Config{
				SamplingPercentage: 100.0,
			},
			td: singleSpanWithAttrib(
				"no.sampling.priority",
				pdata.NewAttributeValueInt(2)),
			sampled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := new(consumertest.TracesSink)
			tsp, err := newTraceProcessor(sink, tt.cfg)
			require.NoError(t, err)

			err = tsp.ConsumeTraces(context.Background(), tt.td)
			require.NoError(t, err)

			sampledData := sink.AllTraces()
			require.Equal(t, 1, len(sampledData))
			assert.Equal(t, tt.sampled, sink.SpansCount() == 1)
		})
	}
}

// Test_parseSpanSamplingPriority ensures that the function parsing the attributes is taking "sampling.priority"
// attribute correctly.
func Test_parseSpanSamplingPriority(t *testing.T) {
	tests := []struct {
		name string
		span pdata.Span
		want samplingPriority
	}{
		{
			name: "nil_span",
			span: pdata.NewSpan(),
			want: deferDecision,
		},
		{
			name: "nil_attributes",
			span: pdata.NewSpan(),
			want: deferDecision,
		},
		{
			name: "no_sampling_priority",
			span: getSpanWithAttributes("key", pdata.NewAttributeValueBool(true)),
			want: deferDecision,
		},
		{
			name: "sampling_priority_int_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueInt(0)),
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_int_gt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueInt(1)),
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_int_lt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueInt(-1)),
			want: deferDecision,
		},
		{
			name: "sampling_priority_double_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueDouble(0)),
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_double_gt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueDouble(1)),
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_double_lt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueDouble(-1)),
			want: deferDecision,
		},
		{
			name: "sampling_priority_string_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueString("0.0")),
			want: doNotSampleSpan,
		},
		{
			name: "sampling_priority_string_gt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueString("0.5")),
			want: mustSampleSpan,
		},
		{
			name: "sampling_priority_string_lt_zero",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueString("-0.5")),
			want: deferDecision,
		},
		{
			name: "sampling_priority_string_NaN",
			span: getSpanWithAttributes("sampling.priority", pdata.NewAttributeValueString("NaN")),
			want: deferDecision,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSpanSamplingPriority(tt.span))
		})
	}
}

func getSpanWithAttributes(key string, value pdata.AttributeValue) pdata.Span {
	span := pdata.NewSpan()
	span.SetName("spanName")
	span.Attributes().InitFromMap(map[string]pdata.AttributeValue{key: value})
	return span
}

// Test_hash ensures that the hash function supports different key lengths even if in
// practice it is only expected to receive keys with length 16 (trace id length in OC proto).
func Test_hash(t *testing.T) {
	// Statistically a random selection of such small number of keys should not result in
	// collisions, but, of course it is possible that they happen, a different random source
	// should avoid that.
	r := rand.New(rand.NewSource(1))
	fullKey := tracetranslator.UInt64ToTraceID(r.Uint64(), r.Uint64()).Bytes()
	seen := make(map[uint32]bool)
	for i := 1; i <= len(fullKey); i++ {
		key := fullKey[:i]
		hash := hash(key, 1)
		require.False(t, seen[hash], "Unexpected duplicated hash")
		seen[hash] = true
	}
}

// genRandomTestData generates a slice of pdata.Traces with the numBatches elements which one with
// numTracesPerBatch spans (ie.: each span has a different trace ID). All spans belong to the specified
// serviceName.
func genRandomTestData(numBatches, numTracesPerBatch int, serviceName string, resourceSpanCount int) (tdd []pdata.Traces) {
	r := rand.New(rand.NewSource(1))
	var traceBatches []pdata.Traces
	for i := 0; i < numBatches; i++ {
		traces := pdata.NewTraces()
		traces.ResourceSpans().Resize(resourceSpanCount)
		for j := 0; j < resourceSpanCount; j++ {
			rs := traces.ResourceSpans().At(j)
			rs.Resource().Attributes().InsertString("service.name", serviceName)
			rs.Resource().Attributes().InsertBool("bool", true)
			rs.Resource().Attributes().InsertString("string", "yes")
			rs.Resource().Attributes().InsertInt("int64", 10000000)
			rs.InstrumentationLibrarySpans().Resize(1)
			ils := rs.InstrumentationLibrarySpans().At(0)
			ils.Spans().Resize(numTracesPerBatch)

			for k := 0; k < numTracesPerBatch; k++ {
				span := ils.Spans().At(k)
				span.SetTraceID(tracetranslator.UInt64ToTraceID(r.Uint64(), r.Uint64()))
				span.SetSpanID(tracetranslator.UInt64ToSpanID(r.Uint64()))
				attributes := make(map[string]pdata.AttributeValue)
				attributes[tracetranslator.TagHTTPStatusCode] = pdata.NewAttributeValueInt(404)
				attributes[tracetranslator.TagHTTPStatusMsg] = pdata.NewAttributeValueString("Not Found")
				span.Attributes().InitFromMap(attributes)
			}
		}
		traceBatches = append(traceBatches, traces)
	}

	return traceBatches
}

// assertSampledData checks for no repeated traceIDs and counts the number of spans on the sampled data for
// the given service.
func assertSampledData(t *testing.T, sampled []pdata.Traces, serviceName string) (traceIDs map[[16]byte]bool, spanCount int) {
	traceIDs = make(map[[16]byte]bool)
	for _, td := range sampled {
		rspans := td.ResourceSpans()
		for i := 0; i < rspans.Len(); i++ {
			rspan := rspans.At(i)
			ilss := rspan.InstrumentationLibrarySpans()
			for j := 0; j < ilss.Len(); j++ {
				ils := ilss.At(j)
				if svcNameAttr, _ := rspan.Resource().Attributes().Get("service.name"); svcNameAttr.StringVal() != serviceName {
					continue
				}
				for k := 0; k < ils.Spans().Len(); k++ {
					spanCount++
					span := ils.Spans().At(k)
					key := span.TraceID().Bytes()
					if traceIDs[key] {
						t.Errorf("same traceID used more than once %q", key)
						return
					}
					traceIDs[key] = true
				}
			}
		}
	}
	return
}
