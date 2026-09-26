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
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeLogs(t *testing.T) {
	lr1 := newLogsRequest(testdata.GenerateLogs(2))
	lr2 := newLogsRequest(testdata.GenerateLogs(3))
	res, err := lr1.MergeSplit(context.Background(), 0, request.SizerTypeItems, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeSplitLogs(t *testing.T) {
	tests := []struct {
		name     string
		szt      request.SizerType
		maxSize  int
		lr1      request.Request
		lr2      request.Request
		expected []request.Request
	}{
		{
			name: "both_requests_empty",
			szt:  request.SizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []request.Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name: "first_request_empty",
			szt:  request.SizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []request.Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name: "first_empty_second_nil",
			szt:  request.SizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []request.Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeItems,
			maxSize: 10,
			lr1:     newLogsRequest(testdata.GenerateLogs(4)),
			lr2:     newLogsRequest(testdata.GenerateLogs(6)),
			expected: []request.Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeItems,
			maxSize: 4,
			lr1:     newLogsRequest(plog.NewLogs()),
			lr2:     newLogsRequest(testdata.GenerateLogs(10)),
			expected: []request.Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     request.SizerTypeItems,
			maxSize: 10,
			lr1:     newLogsRequest(testdata.GenerateLogs(8)),
			lr2:     newLogsRequest(testdata.GenerateLogs(20)),
			expected: []request.Request{
				newLogsRequest(func() plog.Logs {
					logs := testdata.GenerateLogs(8)
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
					return logs
				}()),
				newLogsRequest(testdata.GenerateLogs(10)),
				newLogsRequest(testdata.GenerateLogs(8)),
			},
		},
		{
			name:    "scope_logs_split",
			szt:     request.SizerTypeItems,
			maxSize: 4,
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}()),
			lr2: newLogsRequest(testdata.GenerateLogs(2)),
			expected: []request.Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(func() plog.Logs {
					ld := testdata.GenerateLogs(0)
					ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("extra log")
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(ld.ResourceLogs())
					return ld
				}()),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.lr2)
			require.NoError(t, err)
			assert.Len(t, res, len(tt.expected))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*logsRequest).ld, res[i].(*logsRequest).ld)
			}
		})
	}
}

func TestMergeSplitLogsBasedOnByteSize(t *testing.T) {
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
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []request.Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:     "first_request_empty",
			szt:      request.SizerTypeBytes,
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []request.Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      request.SizerTypeBytes,
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []request.Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:    "merge_only",
			szt:     request.SizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(11)),
			lr1:     newLogsRequest(testdata.GenerateLogs(4)),
			lr2:     newLogsRequest(testdata.GenerateLogs(6)),
			expected: []request.Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name:    "split_only",
			szt:     request.SizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1:     newLogsRequest(plog.NewLogs()),
			lr2:     newLogsRequest(testdata.GenerateLogs(10)),
			expected: []request.Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     request.SizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))/2 + logsMarshaler.LogsSize(testdata.GenerateLogs(11))/2,
			lr1:     newLogsRequest(testdata.GenerateLogs(8)),
			lr2:     newLogsRequest(testdata.GenerateLogs(20)),
			expected: []request.Request{
				newLogsRequest(func() plog.Logs {
					logs := testdata.GenerateLogs(8)
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
					return logs
				}()),
				newLogsRequest(testdata.GenerateLogs(10)),
				newLogsRequest(testdata.GenerateLogs(8)),
			},
		},
		{
			name:    "scope_logs_split",
			szt:     request.SizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}()),
			lr2: newLogsRequest(testdata.GenerateLogs(2)),
			expected: []request.Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(func() plog.Logs {
					ld := testdata.GenerateLogs(0)
					ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("extra log")
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(ld.ResourceLogs())
					return ld
				}()),
			},
		},
		{
			name:    "unsplittable_large_log",
			szt:     request.SizerTypeBytes,
			maxSize: 10,
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(1)
				ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetStr(string(make([]byte, 100)))
				return ld
			}()),
			lr2:                nil,
			expected:           []request.Request{},
			expectPartialError: true,
		},
		{
			name:    "splittable_then_unsplittable_log",
			szt:     request.SizerTypeBytes,
			maxSize: 1000,
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(2)
				ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetStr(string(make([]byte, 10)))
				ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1).Body().SetStr(string(make([]byte, 1001)))
				return ld
			}()),
			lr2: nil,
			expected: []request.Request{newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(1)
				ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().SetStr(string(make([]byte, 10)))
				return ld
			}())},
			expectPartialError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.MergeSplit(context.Background(), tt.maxSize, tt.szt, tt.lr2)
			if tt.expectPartialError {
				require.ErrorContains(t, err, "single log record exceeds the max size limit, dropping")
			} else {
				require.NoError(t, err)
			}
			assert.Len(t, res, len(tt.expected))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*logsRequest).ld, res[i].(*logsRequest).ld)
				assert.Equal(t,
					logsMarshaler.LogsSize(tt.expected[i].(*logsRequest).ld),
					logsMarshaler.LogsSize(res[i].(*logsRequest).ld))
			}
		})
	}
}

func TestMergeSplitLogsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newLogsRequest(testdata.GenerateLogs(18))
	r2 := newTracesRequest(testdata.GenerateTraces(3))
	_, err := r1.MergeSplit(context.Background(), 10, request.SizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestExtractLogs(t *testing.T) {
	for i := range 10 {
		ld := testdata.GenerateLogs(10)
		extractedLogs, _ := extractLogs(ld, i, &sizer.LogsCountSizer{})
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
	}
}

func TestMergeSplitManySmallLogs(t *testing.T) {
	// All requests merge into a single batch.
	merged := []request.Request{newLogsRequest(testdata.GenerateLogs(1))}
	for range 1000 {
		lr2 := newLogsRequest(testdata.GenerateLogs(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestLogsMergeSplitExactBytes(t *testing.T) {
	pb := plog.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), pb.LogsSize(testdata.GenerateLogs(2))-1, request.SizerTypeBytes, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), 1, request.SizerTypeItems, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitUnknownSizerType(t *testing.T) {
	req := newLogsRequest(plog.NewLogs())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, request.SizerType{}, nil)
	require.EqualError(t, err, "unknown sizer type")
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(10))}
		for range 1000 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10010, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnByteSizeManySmallLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(10))}
		for range 1000 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(11000)), request.SizerTypeBytes, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

// BenchmarkByteSizeFlushCheckManySmallLogs mirrors partitionBatcher.consumeInternal:
// each incoming request is merged into the growing batch, then the batch's byte size
// is read to decide whether MinSize is reached. Reading BytesSize() on the whole
// accumulated batch every consume is O(batch) unless the size is cached, making the
// accumulation O(n^2) overall.
func BenchmarkByteSizeFlushCheckManySmallLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	b.ReportAllocs()
	for b.Loop() {
		batch := newLogsRequest(testdata.GenerateLogs(10))
		var total int
		for range 1000 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := batch.MergeSplit(context.Background(), 0, request.SizerTypeBytes, lr2)
			batch = res[len(res)-1]
			// The batcher's MinSize check on the accumulated batch.
			total += batch.BytesSize()
		}
		_ = total
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyAboveLimit(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// Every incoming request results in a split.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(0))}
		for range 10 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyAboveLimit(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// Every incoming request results in a split.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(0))}
		for range 10 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(10000)), request.SizerTypeBytes, lr2)
			assert.Len(b, res, 2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// One request splits into many batches.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, request.SizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// One request splits into many batches.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(10010)), request.SizerTypeBytes, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

// logBodies returns every log record body across the given requests, in order.
func logBodies(reqs []request.Request) []string {
	var out []string
	for _, r := range reqs {
		rls := r.(*logsRequest).ld.ResourceLogs()
		for i := 0; i < rls.Len(); i++ {
			sls := rls.At(i).ScopeLogs()
			for j := 0; j < sls.Len(); j++ {
				lrs := sls.At(j).LogRecords()
				for k := 0; k < lrs.Len(); k++ {
					out = append(out, lrs.At(k).Body().Str())
				}
			}
		}
	}
	return out
}

func newLogsWithBodies(bodies ...string) plog.Logs {
	ld := plog.NewLogs()
	sl := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	for _, b := range bodies {
		sl.LogRecords().AppendEmpty().Body().SetStr(b)
	}
	return ld
}

func TestMergeSplitLogsDropsOnlyOversizedRecord(t *testing.T) {
	oversized := strings.Repeat("x", 1000)

	tests := []struct {
		name         string
		bodies       []string
		wantSurvived []string
		wantDropped  int
	}{
		{
			name:         "oversized_first",
			bodies:       []string{oversized, "a", "b", "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "oversized_in_middle",
			bodies:       []string{"a", "b", oversized, "c", "d"},
			wantSurvived: []string{"a", "b", "c", "d"},
			wantDropped:  1,
		},
		{
			name:         "oversized_last",
			bodies:       []string{"a", "b", "c", oversized},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  1,
		},
		{
			name:         "multiple_oversized",
			bodies:       []string{"a", oversized, "b", oversized, "c"},
			wantSurvived: []string{"a", "b", "c"},
			wantDropped:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newLogsRequest(newLogsWithBodies(tt.bodies...))
			res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)

			wantErr := fmt.Sprintf("single log record exceeds the max size limit, dropping items: %d", tt.wantDropped)
			require.ErrorContains(t, err, wantErr)
			assert.Equal(t, tt.wantSurvived, logBodies(res),
				"records other than the oversized ones must survive")

			for _, r := range res {
				assert.LessOrEqual(t, r.BytesSize(), 100, "no returned batch may exceed max size")
			}
		})
	}
}

func TestMergeSplitLogsAllRecordsOversized(t *testing.T) {
	oversized := strings.Repeat("x", 1000)
	req := newLogsRequest(newLogsWithBodies(oversized, oversized))

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "single log record exceeds the max size limit, dropping items: 2")
	assert.Empty(t, logBodies(res), "nothing can be exported when every record is oversized")
}

func TestMergeSplitLogsItemlessOversizedRequest(t *testing.T) {
	// Resource attributes alone exceed max size and there is no log record at all, so
	// nothing can be exported and nothing is lost that needs reporting.
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().Resource().Attributes().PutStr("big", strings.Repeat("x", 500))
	req := newLogsRequest(ld)
	require.Greater(t, req.BytesSize(), 100, "precondition: request must start oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "nothing was lost, so nothing is due to be reported")
	assert.Empty(t, res, "an oversized request holding no records must not be returned")
}

func TestMergeSplitLogsDropsOnlyOversizedAcrossResourcesAndScopes(t *testing.T) {
	// Only the oversized record is dropped; records in the other scope and resource
	// must survive intact.
	oversized := strings.Repeat("x", 1000)
	ld := plog.NewLogs()
	rl1 := ld.ResourceLogs().AppendEmpty()
	rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(oversized)
	rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("second_scope")
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("second_resource")
	require.Equal(t, 3, ld.LogRecordCount(), "precondition: three records")

	res, err := newLogsRequest(ld).MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "single log record exceeds the max size limit, dropping items: 1")
	assert.ElementsMatch(t, []string{"second_scope", "second_resource"}, logBodies(res),
		"records in the other scope and resource must survive")
}

func TestMergeSplitLogsEmptyOversizedResourceDoesNotStopSplitting(t *testing.T) {
	// A resource with big attributes and no records must not stop splitting of the
	// records behind it, and no record may be reported as dropped.
	ld := plog.NewLogs()
	empty := ld.ResourceLogs().AppendEmpty()
	empty.Resource().Attributes().PutStr("big", strings.Repeat("B", 400))
	empty.ScopeLogs().AppendEmpty()
	sl := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty()
	for range 12 {
		sl.LogRecords().AppendEmpty().Body().SetStr(strings.Repeat("v", 40))
	}
	req := newLogsRequest(ld).(*logsRequest)
	require.Equal(t, 12, req.ld.LogRecordCount(), "precondition: twelve records that each fit")
	require.Greater(t, req.BytesSize(), 100, "precondition: request starts oversized")

	res, err := req.MergeSplit(context.Background(), 100, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no record is oversized, so nothing should be reported")

	marshaler := &plog.ProtoMarshaler{}
	survived := 0
	for _, r := range res {
		lr := r.(*logsRequest)
		survived += lr.ld.LogRecordCount()
		assert.LessOrEqual(t, marshaler.LogsSize(lr.ld), 100, "no batch may exceed max size")
		// The cached size may differ from the marshaled size by a byte, but must not be
		// a stale oversized value.
		assert.LessOrEqual(t, lr.BytesSize(), 100, "a stale cached size makes the batcher over-count")
	}
	assert.Equal(t, 12, survived, "every record must survive")
}

func TestMergeSplitLogsRetriesAfterDiscardingRecordlessResources(t *testing.T) {
	// A small resource holding no record always fits, so extraction moves it out of the
	// source and into a batch that is then discarded for holding no record. The source
	// has shrunk by the time that happens, so a fresh attempt succeeds and no record
	// needs to be given up.
	const maxSize = 462
	fitsAlone := func(body string) bool {
		x := plog.NewLogs()
		rl := x.ResourceLogs().AppendEmpty()
		rl.Resource().Attributes().PutStr("r", strings.Repeat("R", 56))
		rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(body)
		return (&plog.ProtoMarshaler{}).LogsSize(x) <= maxSize
	}
	first, second := strings.Repeat("a", 350), strings.Repeat("b", 276)
	require.True(t, fitsAlone(first), "precondition: the first record fits a batch on its own")
	require.True(t, fitsAlone(second), "precondition: the second record fits a batch on its own")

	ld := plog.NewLogs()
	for range 2 {
		empty := ld.ResourceLogs().AppendEmpty()
		empty.Resource().Attributes().PutStr("e", strings.Repeat("E", 40))
		empty.ScopeLogs().AppendEmpty()
	}
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("r", strings.Repeat("R", 56))
	sl := rl.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr(first)
	sl.LogRecords().AppendEmpty().Body().SetStr(second)

	res, err := newLogsRequest(ld).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
	require.NoError(t, err, "no record is oversized, so none may be dropped")

	marshaler := &plog.ProtoMarshaler{}
	survived := 0
	for _, r := range res {
		lr := r.(*logsRequest)
		survived += lr.ld.LogRecordCount()
		assert.LessOrEqual(t, marshaler.LogsSize(lr.ld), maxSize, "no batch may exceed max size")
	}
	assert.Equal(t, 2, survived, "both records must be exported")
}

func TestMergeSplitLogsStopsWhenNoProgressIsPossible(t *testing.T) {
	// The resource attributes alone fill max size exactly, so no scope or record can be
	// added to any batch. Splitting must stop instead of looping, and keep the records.
	const maxSize = 300
	marshaler := &plog.ProtoMarshaler{}
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	for pad := 0; ; pad++ {
		rl.Resource().Attributes().PutStr("pad", strings.Repeat("p", pad))
		if marshaler.LogsSize(ld) == maxSize {
			break
		}
	}
	sl := rl.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("first")
	sl.LogRecords().AppendEmpty().Body().SetStr("second")

	res, err := newLogsRequest(ld).MergeSplit(context.Background(), maxSize, request.SizerTypeBytes, nil)
	require.ErrorContains(t, err, "request size is greater than max size and cannot be split further")
	assert.ElementsMatch(t, []string{"first", "second"}, logBodies(res), "records must be returned, not lost")
}