// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeTraces(t *testing.T) {
	tr1 := newTracesRequest(testdata.GenerateTraces(2))
	tr2 := newTracesRequest(testdata.GenerateTraces(3))
	res, err := tr1.MergeSplit(context.Background(), 0, request.SizerTypeItems, tr2)
	require.NoError(t, err)
	assert.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeSplitTraces(t *testing.T) {
	tests := []struct {
		name     string
		szt      request.SizerType
		maxSize  int
		tr1      request.Request
		tr2      request.Request
		expected []request.Request
	}{
		{
			name:     "both_requests_empty",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []request.Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:     "first_request_empty",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      newTracesRequest(testdata.GenerateTraces(5)),
			expected: []request.Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "second_request_empty",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(testdata.GenerateTraces(5)),
			tr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []request.Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      request.SizerTypeItems,
			maxSize:  10,
			tr1:      newTracesRequest(ptrace.NewTraces()),
			tr2:      nil,
			expected: []request.Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeItems,
			maxSize: 10,
			tr1:     newTracesRequest(testdata.GenerateTraces(5)),
			tr2:     newTracesRequest(testdata.GenerateTraces(5)),
			expected: []request.Request{newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(5)
				testdata.GenerateTraces(5).ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeItems,
			maxSize: 4,
			tr1:     newTracesRequest(ptrace.NewTraces()),
			tr2:     newTracesRequest(testdata.GenerateTraces(10)),
			expected: []request.Request{
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(2)),
			},
		},
		{
			name:    "split_and_merge",
			szt:     request.SizerTypeItems,
			maxSize: 10,
			tr1:     newTracesRequest(testdata.GenerateTraces(4)),
			tr2:     newTracesRequest(testdata.GenerateTraces(20)),
			expected: []request.Request{
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
			szt:     request.SizerTypeItems,
			maxSize: 10,
			tr1: newTracesRequest(func() ptrace.Traces {
				td := testdata.GenerateTraces(10)
				extraScopeTraces := testdata.GenerateTraces(5)
				extraScopeTraces.ResourceSpans().At(0).ScopeSpans().At(0).Scope().SetName("extra scope")
				extraScopeTraces.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
				return td
			}()),
			tr2: nil,
			expected: []request.Request{
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
		szt                request.SizerType
		maxSize            int
		lr1                request.Request
		lr2                request.Request
		expected           []request.Request
		expectPartialError bool
	}{
		{
			name:     "both_requests_empty",
			szt:      request.SizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      newTracesRequest(ptrace.NewTraces()),
			expected: []request.Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:     "first_request_empty",
			szt:      request.SizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      newTracesRequest(testdata.GenerateTraces(5)),
			expected: []request.Request{newTracesRequest(testdata.GenerateTraces(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      request.SizerTypeBytes,
			maxSize:  tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:      newTracesRequest(ptrace.NewTraces()),
			lr2:      nil,
			expected: []request.Request{newTracesRequest(ptrace.NewTraces())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(10)),
			lr1:     newTracesRequest(testdata.GenerateTraces(1)),
			lr2:     newTracesRequest(testdata.GenerateTraces(6)),
			expected: []request.Request{newTracesRequest(func() ptrace.Traces {
				traces := testdata.GenerateTraces(1)
				testdata.GenerateTraces(6).ResourceSpans().MoveAndAppendTo(traces.ResourceSpans())
				return traces
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(4)),
			lr1:     newTracesRequest(ptrace.NewTraces()),
			lr2:     newTracesRequest(testdata.GenerateTraces(10)),
			expected: []request.Request{
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(4)),
				newTracesRequest(testdata.GenerateTraces(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     request.SizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(10))/2 + tracesMarshaler.TracesSize(testdata.GenerateTraces(11))/2,
			lr1:     newTracesRequest(testdata.GenerateTraces(8)),
			lr2:     newTracesRequest(testdata.GenerateTraces(20)),
			expected: []request.Request{
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
			szt:     request.SizerTypeBytes,
			maxSize: tracesMarshaler.TracesSize(testdata.GenerateTraces(4)),
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(4)
				ld.ResourceSpans().At(0).ScopeSpans().AppendEmpty().Spans().AppendEmpty().Attributes().PutStr("attr", "attrvalue")
				return ld
			}()),
			lr2: newTracesRequest(testdata.GenerateTraces(2)),
			expected: []request.Request{
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
			szt:     request.SizerTypeBytes,
			maxSize: 10,
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(1)
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr("large_attr", string(make([]byte, 100)))
				return ld
			}()),
			lr2:                nil,
			expected:           []request.Request{},
			expectPartialError: true,
		},
		{
			name:    "splittable_then_unsplittable_trace",
			szt:     request.SizerTypeBytes,
			maxSize: 1000,
			lr1: newTracesRequest(func() ptrace.Traces {
				ld := testdata.GenerateTraces(2)
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr("large_attr", string(make([]byte, 10)))
				ld.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(1).Attributes().PutStr("large_attr", string(make([]byte, 1001)))
				return ld
			}()),
			lr2: nil,
			expected: []request.Request{newTracesRequest(func() ptrace.Traces {
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
	_, err := r1.MergeSplit(context.Background(), 10, request.SizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestExtractTraces(t *testing.T) {
	for i := range 10 {
		td := testdata.GenerateTraces(10)
		extractedTraces, removedSize := extractTraces(td, i, &sizer.TracesCountSizer{})
		assert.Equal(t, i, extractedTraces.SpanCount())
		assert.Equal(t, 10-i, td.SpanCount())
		assert.Equal(t, i, removedSize)
	}
}

func TestMergeSplitManySmallTraces(t *testing.T) {
	merged := []request.Request{newTracesRequest(testdata.GenerateTraces(1))}
	for range 1000 {
		lr2 := newTracesRequest(testdata.GenerateTraces(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestTracesMergeSplitExactBytes(t *testing.T) {
	pb := ptrace.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newTracesRequest(testdata.GenerateTraces(4))
	merged, err := lr.MergeSplit(context.Background(), pb.TracesSize(testdata.GenerateTraces(2))-1, request.SizerTypeBytes, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestTracesMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newTracesRequest(testdata.GenerateTraces(4))
	merged, err := lr.MergeSplit(context.Background(), 1, request.SizerTypeItems, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestTracesMergeSplitUnknownSizerType(t *testing.T) {
	req := newTracesRequest(ptrace.NewTraces())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, request.SizerType{}, nil)
	require.EqualError(t, err, "unknown sizer type")
}

func BenchmarkSplittingBasedOnItemCountManySmallTraces(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newTracesRequest(testdata.GenerateTraces(10))}
		for range 1000 {
			lr2 := newTracesRequest(testdata.GenerateTraces(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10010, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyTracesSlightlyAboveLimit(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// Every incoming request results in a split.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newTracesRequest(testdata.GenerateTraces(0))}
		for range 10 {
			lr2 := newTracesRequest(testdata.GenerateTraces(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeTraces(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// One request splits into many batches.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newTracesRequest(testdata.GenerateTraces(0))}
		lr2 := newTracesRequest(testdata.GenerateTraces(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

// spanNames returns every span name across the given requests, in order.
func spanNames(reqs []request.Request) []string {
	var out []string
	for _, r := range reqs {
		rss := r.(*tracesRequest).td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			sss := rss.At(i).ScopeSpans()
			for j := 0; j < sss.Len(); j++ {
				spans := sss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					out = append(out, spans.At(k).Name())
				}
			}
		}
	}
	return out
}

// newTracesWithSpans builds one span per name; a name of "BIG" gets an
// attribute large enough that the span cannot fit into any batch.
func newTracesWithSpans(names ...string) ptrace.Traces {
	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for _, n := range names {
		span := ss.Spans().AppendEmpty()
		span.SetName(n)
		if n == "BIG" {
			span.Attributes().PutStr("pad", strings.Repeat("x", 1000))
		}
	}
	return td
}

func TestMergeSplitTracesDropsOnlyOversizedSpan(t *testing.T) {
	tests := []struct {
		name         string
		spans        []string
		wantSurvived []string
		wantDropped  int
	}{
		{"oversized_first", []string{"BIG", "a", "b", "c"}, []string{"a", "b", "c"}, 1},
		{"oversized_in_middle", []string{"a", "b", "BIG", "c", "d"}, []string{"a", "b", "c", "d"}, 1},
		{"oversized_last", []string{"a", "b", "c", "BIG"}, []string{"a", "b", "c"}, 1},
		{"multiple_oversized", []string{"a", "BIG", "b", "BIG", "c"}, []string{"a", "b", "c"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTracesRequest(newTracesWithSpans(tt.spans...))
			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)

			wantErr := fmt.Sprintf("one span size is greater than max size, dropping items: %d", tt.wantDropped)
			require.ErrorContains(t, err, wantErr)
			assert.Equal(t, tt.wantSurvived, spanNames(res),
				"spans other than the oversized ones must survive")

			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100, "no returned batch may exceed max size")
			}
		})
	}
}

func TestMergeSplitTracesAllSpansOversized(t *testing.T) {
	req := newTracesRequest(newTracesWithSpans("BIG", "BIG"))
	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "one span size is greater than max size, dropping items: 2")
	assert.Empty(t, spanNames(res), "nothing can be exported when every span is oversized")
}

func TestMergeSplitTracesItemlessOversizedRequest(t *testing.T) {
	// Resource attributes alone exceed max size, and the request carries no
	// span at all. The drop pass prunes that resource, which leaves nothing to
	// export and nothing to report: no span was lost because there was none.
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newTracesRequest(td)
	require.Greater(t, req.BytesSize(), 100, "precondition: request must start oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "nothing was lost, so nothing is due to be reported")
	assert.Empty(t, res, "an oversized request holding no spans must not be returned")
}

func TestMergeSplitTracesDropsOnlyOversizedAcrossResourcesAndScopes(t *testing.T) {
	// dropOversizedSpans keeps every span that fits. With several
	// resources and scopes, the untouched ones must survive intact.
	oversized := strings.Repeat("x", 1000)
	td := ptrace.NewTraces()
	rs1 := td.ResourceSpans().AppendEmpty()
	rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(oversized)
	rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("second_scope")
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("second_resource")
	require.Equal(t, 3, td.SpanCount(), "precondition: three spans")

	res, err := newTracesRequest(td).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "one span size is greater than max size, dropping items: 1")

	var names []string
	for _, r := range res {
		tr := r.(*tracesRequest)
		for a := 0; a < tr.td.ResourceSpans().Len(); a++ {
			for b := 0; b < tr.td.ResourceSpans().At(a).ScopeSpans().Len(); b++ {
				spans := tr.td.ResourceSpans().At(a).ScopeSpans().At(b).Spans()
				for c := 0; c < spans.Len(); c++ {
					names = append(names, spans.At(c).Name())
				}
			}
		}
	}
	assert.ElementsMatch(t, []string{"second_scope", "second_resource"}, names,
		"spans in the other scope and resource must survive")
}

func TestMergeSplitTracesEmptyOversizedResourceDoesNotStopSplitting(t *testing.T) {
	// See the logs equivalent: a resource pruned without a span being dropped still
	// counts as progress, otherwise the remainder goes out unsplit.
	td := ptrace.NewTraces()
	empty := td.ResourceSpans().AppendEmpty()
	empty.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
	empty.ScopeSpans().AppendEmpty()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for range 12 {
		ss.Spans().AppendEmpty().SetName(strings.Repeat("v", 40))
	}
	req := newTracesRequest(td).(*tracesRequest)
	require.Equal(t, 12, req.td.SpanCount(), "precondition: twelve spans that each fit")
	require.Greater(t, req.BytesSize(), 100, "precondition: request starts oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no span is oversized, so nothing should be reported")

	marshaler := &ptrace.ProtoMarshaler{}
	survived := 0
	for _, r := range res {
		tr := r.(*tracesRequest)
		survived += tr.td.SpanCount()
		assert.LessOrEqual(t, marshaler.TracesSize(tr.td), 100, "no batch may exceed max size")
		// The cached size may differ from the marshaled size by a byte, which the
		// existing delta accounting does on the remainder even without a drop pass.
		// What must not happen is the stale oversized value the pass used to leave.
		assert.LessOrEqual(t, tr.BytesSize(), 100, "a stale cached size makes the batcher over-count")
	}
	assert.Equal(t, 12, survived, "every span must survive")
}

func TestMergeSplitTracesRetriesAfterDiscardingSpanlessResources(t *testing.T) {
	// See the logs equivalent: the source shrinks when extraction discards resources
	// holding no span, so a fresh attempt succeeds and no span is given up.
	const maxSize = 462
	td := ptrace.NewTraces()
	for range 2 {
		empty := td.ResourceSpans().AppendEmpty()
		empty.Resource().Attributes().PutStr("e", strings.Repeat("E", 40))
		empty.ScopeSpans().AppendEmpty()
	}
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("r", strings.Repeat("R", 56))
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName(strings.Repeat("a", 350))
	ss.Spans().AppendEmpty().SetName(strings.Repeat("b", 276))

	res, err := newTracesRequest(td).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no span is oversized, so none may be dropped")

	marshaler := &ptrace.ProtoMarshaler{}
	survived := 0
	for _, r := range res {
		tr := r.(*tracesRequest)
		survived += tr.td.SpanCount()
		assert.LessOrEqual(t, marshaler.TracesSize(tr.td), maxSize, "no batch may exceed max size")
	}
	assert.Equal(t, 2, survived, "both spans must be exported")
}
