// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeTraces(t *testing.T) {
	tr1 := newTracesRequest(testdata.GenerateTraces(2))
	tr2 := newTracesRequest(testdata.GenerateTraces(3))
	res, err := tr1.MergeSplit(context.Background(), 0, RequestSizerTypeItems, tr2)
	require.NoError(t, err)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeSplitTraces(t *testing.T) {
	tests := []struct {
		name     string
		szt      RequestSizerType
		maxSize  int
		tr1      Request
		tr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:     "first_request_empty",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      newTracesRequest(testdata.GenerateTraces(5)),
			expected: []Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "second_request_empty",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(testdata.GenerateTraces(5)),
			tr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      RequestSizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      nil,
			expected: []Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeItems,
			maxSize: 10,
			tr1:     newTracesRequest(testdata.GenerateTraces(5)),
			tr2:     newTracesRequest(testdata.GenerateTraces(5)),
			expected: []Request{newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(5)
				testdata.GenerateTraces(5).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeItems,
			maxSize: 4,
			tr1:     newTracesRequest(ptrace.NewTraces()),
			tr2:     newTracesRequest(testdata.GenerateTraces(10)),
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(2)),
			},
		},
		{
			name:    "split_and_merge",
			szt:     RequestSizerTypeItems,
			maxSize: 10,
			tr1:     newTracesRequest(testdata.GenerateTraces(4)),
			tr2:     newTracesRequest(testdata.GenerateTraces(20)),
			expected: []Request{
				newTracesRequest(func() ptrace.Traces {
					td := testdata.GenerateTraces(4)
					testdata.GenerateTraces(6).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
					return td
				}()),
				newTracesRequest(testdata.GenerateTraces(10)),
				newTracesRequest(testdata.GenerateTraces(4)),
			},
		},
		{
			name:    "scope_spans_split",
			szt:     RequestSizerTypeItems,
			maxSize: 10,
			tr1: newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(10)
				extraScopeTraces := testdata.GenerateTraces(5)
				extraScopeTraces.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
				extraScopeTraces.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}()),
			tr2: nil,
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(10)),
				newTracesRequest(func() ptrace.Traces {
					td := testdata.GenerateTraces(5)
					td.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
					return td
				}()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.tr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.tr2)
			require.NoError(t, err)
			assert.Len(t, res, len(tt.expected))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*tracesRequest).td, res[i].(*tracesRequest).td)
			}
		})
	}
}

func TestMergeSplitTracesBasedOnByteSize(t *testing.T) {
	tests := []struct {
		name               string
		szt                RequestSizerType
		maxSize            int
		lr1                Request
		lr2                Request
		expected           []Request
		expectPartialError bool
	}{
		{
			name:     "both_requests_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:     "first_request_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      newTracesRequest(testdata.GenerateTraces(5)),
			expected: []Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      RequestSizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      nil,
			expected: []Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:     newTracesRequest(testdata.GenerateTraces(1)),
			lr2:     newTracesRequest(testdata.GenerateTraces(6)),
			expected: []Request{newTracesRequest(func() ptrace.Traces {
				traces := testdata.GenerateTraces(1)
				testdata.GenerateTraces(6).ResourceSpans().MoveAndAppendTo(traces.ResourceSpans())
				return traces
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(4)),
			lr1:     newTracesRequest(ptrace.NewTraces()),
			lr2:     newTracesRequest(testdata.GenerateTraces(10)),
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     RequestSizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(10))/2 + tracesMarshaler.TracesSize(testdata.GenerateTraces(11))/2,
			lr1:     newTracesRequest(testdata.GenerateTraces(8)),
			lr2:     newTracesRequest(testdata.GenerateTraces(20)),
			expected: []Request{
				newTracesRequest(func() ptrace.Traces {
					traces := testdata.GenerateTraces(8)
					testdata.GenerateTraces(2).ResourceSpans().MoveAndAppendTo(traces.ResourceSpans())
					return traces
				}()),
				newTracesRequest(testdata.GenerateTraces(10)),
				newTracesRequest(testdata.GenerateTraces(8)),
			},
		},
		{
			name:    "scope_spans_split",
			szt:     RequestSizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(4)),
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(4)
				ld.ResourceSpans().At(0).ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().PutStr("attr", "attrvalue")
				return ld
			}()),
			lr2: newTracesRequest(testdata.GenerateTraces(2)),
			expected: []Request{
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(func() ptrace.Traces {
					ld := testdata.GenerateTraces(0)
					ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().AppendEmpty().Attributes().PutStr("attr", "attrvalue")
					testdata.GenerateTraces(2).ResourceSpans().MoveAndAppendTo(ld.ResourceSpans())
					return ld
				}()),
			},
			expectPartialError: false,
		},
		{
			name:    "unsplittable_large_trace",
			szt:     RequestSizerTypeBytes,
			maxSize: 10,
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(1)
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr("large_attr", string(make([]byte, 100)))
				return ld
			}()),
			lr2:                nil,
			expected:           []Request{},
			expectPartialError: true,
		},
		{
			name:    "splittable_then_unsplittable_trace",
			szt:     RequestSizerTypeBytes,
			maxSize: 1000,
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(2)
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr("large_attr", string(make([]byte, 10)))
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(1).Attributes().PutStr("large_attr", string(make([]byte, 1001)))
				return ld
			}()),
			lr2: nil,
			expected: []Request{newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(1)
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr("large_attr", string(make([]byte, 10)))
				return ld
			}())},
			expectPartialError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.lr2)
			if tt.expectPartialError {
				require.ErrorContains(t, err, "one span size is greater than max size, dropping items:")
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, res, len(tt.expected))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*tracesRequest).td, res[i].(*tracesRequest).td)
				assert.Equal(t,
					tracesMarshaler.TracesSize(tt.expected[i].(*tracesRequest).td),
					tracesMarshaler.TracesSize(res[i].(*tracesRequest).td))
			}
		})
	}
}

func TestMergeSplitTracesInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(18))
	r2 := newLogsRequest(testdata.GenerateLogs(3))
	_, err := r1.MergeSplit(context.Background(), 10, RequestSizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestExtractTraces(t *testing.T) {
	for i := 0; i < 10; i++ {
		td := testdata.GenerateTraces(10)
		extractedTraces, removedSize := extractTraces(td, i, &sizer.TracesCountSizer{})
		assert.Equal(t, i, extractedTraces.SpanCount())
		assert.Equal(t, 10-i, td.SpanCount())
		assert.Equal(t, i, removedSize)
	}
}

func TestMergeSplitManySmallTraces(t *testing.T) {
	merged := []Request{newTracesRequest(testdata.GenerateTraces(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newTracesRequest(testdata.GenerateTraces(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestTracesMergeSplitExactBytes(t *testing.T) {
	pb := ptrace.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newTracesRequest(testdata.GenerateTraces(4))
	merged, err := lr.MergeSplit(context.Background(), pb.TracesSize(testdata.GenerateTraces(2))-1, RequestSizerTypeBytes, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestTracesMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newTracesRequest(testdata.GenerateTraces(4))
	merged, err := lr.MergeSplit(context.Background(), 1, RequestSizerTypeItems, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestTracesMergeSplitUnknownSizerType(t *testing.T) {
	req := newTracesRequest(ptrace.NewTraces())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, RequestSizerType{}, nil)
	require.EqualError(t, err, "unknown sizer type")
}

func BenchmarkSplittingBasedOnItemCountManySmallTraces(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newTracesRequest(testdata.GenerateTraces(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10010, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyTracesSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(0))}
		for j := 0; j < 10; j++ {
			lr2 := newTracesRequest(testdata.GenerateTraces(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeTraces(b *testing.B) {
	// One request splits into many batches.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newTracesRequest(testdata.GenerateTraces(0))}
		lr2 := newTracesRequest(testdata.GenerateTraces(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
