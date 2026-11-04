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
				require.ErrorContains(t, err, "single span exceeds the max size limit, dropping items:")
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

func newTracesWithSpanNames(names ...string) ptrace.Traces {
	td := ptrace.NewTraces()
	ss := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()
	for _, n := range names {
		ss.Spans().AppendEmpty().SetName(n)
	}
	return td
}

func TestMergeSplitTracesDropsOnlyOversizedSpan(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	// More spans than one batch holds, so the scope still has spans after the pass
	// that drops the oversized one.
	many := make([]string, 30)
	for i := range many {
		many[i] = fmt.Sprintf("span-%02d", i)
	}

	tests := []struct {
		name         string
		names        []string
		wantSurvived []string
		wantDropped  int
	}{
		{
			name:         "oversized_first",
			names:        []string{oversized, "a", "b", "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "oversized_in_middle",
			names:        []string{"a", "b", oversized, "c", "d"},
			wantSurvived: []string{"a", "b", "c", "d"},
			wantDropped:  1,
		},
		{
			name:         "oversized_last",
			names:        []string{"a", "b", "c", oversized},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "multiple_oversized",
			names:        []string{"a", oversized, "b", oversized, "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  2,
		},
		{
			name:         "oversized_followed_by_several_batches",
			names:        append([]string{oversized}, many...),
			wantSurvived: many,
			wantDropped:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTracesRequest(newTracesWithSpanNames(tt.names...))
			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)

			wantErr := fmt.Sprintf("single span exceeds the max size limit, dropping items: %d", tt.wantDropped)
			require.ErrorContains(t, err, wantErr)
			assert.Equal(t, tt.wantSurvived, spanNames(res),
				"spans other than the oversized ones must survive")

			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100, "no returned batch may exceed max size")
				assert.Equal(t, tracesMarshaler.TracesSize(r.(*tracesRequest).td), r.BytesSize(),
					"the cached size must stay exact after dropping spans")
			}
		})
	}
}

func TestMergeSplitTracesAllSpansOversized(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	req := newTracesRequest(newTracesWithSpanNames(oversized, oversized))

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "single span exceeds the max size limit, dropping items: 2")
	assert.Empty(t, spanNames(res), "nothing can be exported when every span is oversized")
}

func TestMergeSplitTracesItemlessOversizedRequest(t *testing.T) {
	// Resource attributes alone exceed max size and there is no span at all, so
	// nothing can be exported and nothing is lost that needs reporting.
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newTracesRequest(td)
	require.Greater(t, req.BytesSize(), 100, "precondition: request must start oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "nothing was lost, so nothing is due to be reported")
	assert.Empty(t, res, "an oversized request holding no spans must not be returned")
}

func TestMergeSplitTracesDropsOnlyOversizedAcrossResourcesAndScopes(t *testing.T) {
	// Only the oversized span is dropped; spans in the other scope and resource
	// must survive intact.
	oversized := strings.Repeat("x", 1000)
	td := ptrace.NewTraces()
	rs1 := td.ResourceSpans().AppendEmpty()
	rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(oversized)
	rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("second_scope")
	td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName("second_resource")
	require.Equal(t, 3, td.SpanCount(), "precondition: three spans")

	res, err := newTracesRequest(td).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "single span exceeds the max size limit, dropping items: 1")
	assert.ElementsMatch(t, []string{"second_scope", "second_resource"}, spanNames(res),
		"spans in the other scope and resource must survive")
}

func TestMergeSplitTracesEmptyOversizedResourceDoesNotStopSplitting(t *testing.T) {
	// A resource with big attributes and no spans must not stop splitting of the
	// spans behind it, and no span may be reported as dropped.
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

	survived := 0
	for _, r := range res {
		tr := r.(*tracesRequest)
		survived += tr.td.SpanCount()
		assert.LessOrEqual(t, tracesMarshaler.TracesSize(tr.td), 100, "no batch may exceed max size")
		assert.Equal(t, tracesMarshaler.TracesSize(tr.td), tr.BytesSize(),
			"the cached size must stay exact after removing a span-less resource")
	}
	assert.Equal(t, 12, survived, "every span must survive")
}

func TestMergeSplitTracesRetriesAfterDiscardingSpanlessResources(t *testing.T) {
	// A small resource holding no span always fits, so extraction moves it out of the
	// source and into a batch that is then discarded for holding no span. The source
	// has shrunk by the time that happens, so a fresh attempt succeeds and no span
	// needs to be given up.
	const maxSize = 462
	fitsAlone := func(name string) bool {
		x := ptrace.NewTraces()
		rs := x.ResourceSpans().AppendEmpty()
		rs.Resource().Attributes().PutStr("r", strings.Repeat("R", 56))
		rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty().SetName(name)
		return tracesMarshaler.TracesSize(x) <= maxSize
	}
	first, second := strings.Repeat("a", 350), strings.Repeat("b", 276)
	require.True(t, fitsAlone(first), "precondition: the first span fits a batch on its own")
	require.True(t, fitsAlone(second), "precondition: the second span fits a batch on its own")

	td := ptrace.NewTraces()
	for range 2 {
		empty := td.ResourceSpans().AppendEmpty()
		empty.Resource().Attributes().PutStr("e", strings.Repeat("E", 40))
		empty.ScopeSpans().AppendEmpty()
	}
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("r", strings.Repeat("R", 56))
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName(first)
	ss.Spans().AppendEmpty().SetName(second)

	res, err := newTracesRequest(td).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no span is oversized, so none may be dropped")

	survived := 0
	for _, r := range res {
		tr := r.(*tracesRequest)
		survived += tr.td.SpanCount()
		assert.LessOrEqual(t, tracesMarshaler.TracesSize(tr.td), maxSize, "no batch may exceed max size")
	}
	assert.Equal(t, 2, survived, "both spans must be exported")
}

func TestMergeSplitTracesStopsWhenNoProgressIsPossible(t *testing.T) {
	// The resource attributes alone fill max size exactly, so no scope or span can be
	// added to any batch. Splitting must stop instead of looping, and must not return a
	// batch larger than max size.
	const maxSize = 300
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	for pad := 0; ; pad++ {
		rs.Resource().Attributes().PutStr("pad", strings.Repeat("p", pad))
		if tracesMarshaler.TracesSize(td) == maxSize {
			break
		}
	}
	ss := rs.ScopeSpans().AppendEmpty()
	ss.Spans().AppendEmpty().SetName("first")
	ss.Spans().AppendEmpty().SetName("second")

	res, err := newTracesRequest(td).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "request size is greater than max size and cannot be split further, dropping items: 2")
	assert.Empty(t, res, "spans that cannot fit any batch must not be returned")
}
