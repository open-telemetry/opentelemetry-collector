// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"

import (
	"context"
	"fmt"
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
	res, err := lr1.MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 0}, lr2)
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
			res, err := tt.lr1.MergeSplit(context.Background(), map[request.SizerType]int64{tt.szt: int64(tt.maxSize)}, tt.lr2)
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
			res, err := tt.lr1.MergeSplit(context.Background(), map[request.SizerType]int64{tt.szt: int64(tt.maxSize)}, tt.lr2)
			if tt.expectPartialError {
				require.ErrorContains(t, err, "one log record size is greater than max size, dropping")
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
	_, err := r1.MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestExtractLogs(t *testing.T) {
	for i := 1; i < 10; i++ {
		ld := testdata.GenerateLogs(10)
		extractedLogs := extractLogs(ld, i, &sizer.LogsCountSizer{})
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
	}
}

func TestMergeSplitManySmallLogs(t *testing.T) {
	// All requests merge into a single batch.
	merged := []request.Request{newLogsRequest(testdata.GenerateLogs(1))}
	for range 1000 {
		lr2 := newLogsRequest(testdata.GenerateLogs(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 10000}, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestLogsMergeSplitExactBytes(t *testing.T) {
	pb := plog.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeBytes: int64(pb.LogsSize(testdata.GenerateLogs(2)) - 1)}, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 1}, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitUnknownSizerType(t *testing.T) {
	req := newLogsRequest(plog.NewLogs())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), map[request.SizerType]int64{{}: 0}, nil)
	require.EqualError(t, err, "unknown sizer type: \"\"")
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	testutil.SkipGCHeavyBench(b)
	// All requests merge into a single batch.
	b.ReportAllocs()
	for b.Loop() {
		merged := []request.Request{newLogsRequest(testdata.GenerateLogs(10))}
		for range 1000 {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 10010}, lr2)
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
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeBytes: int64(logsMarshaler.LogsSize(testdata.GenerateLogs(11000)))}, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
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
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 10000}, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func TestMergeSplitLogsMultiSizerOrder(t *testing.T) {
	pb := plog.ProtoMarshaler{}

	// Create 4 distinct logs in order.
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	for i := range 4 {
		lr := sl.LogRecords().AppendEmpty()
		lr.Body().SetStr(fmt.Sprintf("log-%d", i))
	}

	// Calculate size of 2 logs to set byte limit.
	ld2 := plog.NewLogs()
	rl2 := ld2.ResourceLogs().AppendEmpty()
	sl2 := rl2.ScopeLogs().AppendEmpty()
	for i := range 2 {
		lr := sl2.LogRecords().AppendEmpty()
		lr.Body().SetStr(fmt.Sprintf("log-%d", i))
	}
	limitBytes := int64(pb.LogsSize(ld2) - 1)

	limits := map[request.SizerType]int64{
		request.SizerTypeItems: 2,
		request.SizerTypeBytes: limitBytes,
	}

	req := newLogsRequest(ld)
	res, err := req.MergeSplit(context.Background(), limits, nil)
	require.NoError(t, err)

	// We expect 4 batches, each with 1 log record.
	assert.Len(t, res, 4)

	// Verify order by checking the body of the log record in each batch!
	for i := range 4 {
		lrReq := res[i].(*logsRequest)
		assert.Equal(t, 1, lrReq.ItemsCount())

		// Extract the body string
		body := lrReq.ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str()
		assert.Equal(t, fmt.Sprintf("log-%d", i), body)
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
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeBytes: int64(logsMarshaler.LogsSize(testdata.GenerateLogs(10000)))}, lr2)
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
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeItems: 10000}, lr2)
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
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), map[request.SizerType]int64{request.SizerTypeBytes: int64(logsMarshaler.LogsSize(testdata.GenerateLogs(10010)))}, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
