// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeLogs(t *testing.T) {
	lr1 := newLogsRequest(testdata.GenerateLogs(2))
	lr2 := newLogsRequest(testdata.GenerateLogs(3))
	res, err := lr1.MergeSplit(context.Background(), 0, RequestSizerTypeItems, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeSplitLogs(t *testing.T) {
	tests := []struct {
		name     string
		szt      RequestSizerType
		maxSize  int
		lr1      Request
		lr2      Request
		expected []Request
	}{
		{
			name: "both_requests_empty",
			szt:  RequestSizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name: "first_request_empty",
			szt:  RequestSizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name: "first_empty_second_nil",
			szt:  RequestSizerTypeItems, maxSize: 10,
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeItems,
			maxSize: 10,
			lr1:     newLogsRequest(testdata.GenerateLogs(4)),
			lr2:     newLogsRequest(testdata.GenerateLogs(6)),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeItems,
			maxSize: 4,
			lr1:     newLogsRequest(plog.NewLogs()),
			lr2:     newLogsRequest(testdata.GenerateLogs(10)),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     RequestSizerTypeItems,
			maxSize: 10,
			lr1:     newLogsRequest(testdata.GenerateLogs(8)),
			lr2:     newLogsRequest(testdata.GenerateLogs(20)),
			expected: []Request{
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
			szt:     RequestSizerTypeItems,
			maxSize: 4,
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}()),
			lr2: newLogsRequest(testdata.GenerateLogs(2)),
			expected: []Request{
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
		name     string
		szt      RequestSizerType
		maxSize  int
		lr1      Request
		lr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:     "first_request_empty",
			szt:      RequestSizerTypeBytes,
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name:     "first_empty_second_nil",
			szt:      RequestSizerTypeBytes,
			maxSize:  logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:    "merge_only",
			szt:     RequestSizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(11)),
			lr1:     newLogsRequest(testdata.GenerateLogs(4)),
			lr2:     newLogsRequest(testdata.GenerateLogs(6)),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name:    "split_only",
			szt:     RequestSizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1:     newLogsRequest(plog.NewLogs()),
			lr2:     newLogsRequest(testdata.GenerateLogs(10)),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name:    "merge_and_split",
			szt:     RequestSizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))/2 + logsMarshaler.LogsSize(testdata.GenerateLogs(11))/2,
			lr1:     newLogsRequest(testdata.GenerateLogs(8)),
			lr2:     newLogsRequest(testdata.GenerateLogs(20)),
			expected: []Request{
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
			szt:     RequestSizerTypeBytes,
			maxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}()),
			lr2: newLogsRequest(testdata.GenerateLogs(2)),
			expected: []Request{
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

func TestMergeSplitLogsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newLogsRequest(testdata.GenerateLogs(18))
	r2 := newTracesRequest(testdata.GenerateTraces(3))
	_, err := r1.MergeSplit(context.Background(), 10, RequestSizerTypeItems, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestExtractLogs(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateLogs(10)
		extractedLogs, _ := extractLogs(ld, i, &sizer.LogsCountSizer{})
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
	}
}

func TestMergeSplitManySmallLogs(t *testing.T) {
	// All requests merge into a single batch.
	merged := []Request{newLogsRequest(testdata.GenerateLogs(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newLogsRequest(testdata.GenerateLogs(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestLogsMergeSplitExactBytes(t *testing.T) {
	pb := plog.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), pb.LogsSize(testdata.GenerateLogs(2))-1, RequestSizerTypeBytes, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), 1, RequestSizerTypeItems, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitUnknownSizerType(t *testing.T) {
	req := newLogsRequest(plog.NewLogs())
	// Call MergeSplit with invalid sizer
	_, err := req.MergeSplit(context.Background(), 0, RequestSizerType{}, nil)
	require.EqualError(t, err, "unknown sizer type")
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10010, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnByteSizeManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(11000)), RequestSizerTypeBytes, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(10000)), RequestSizerTypeBytes, lr2)
			assert.Len(b, res, 2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeLogs(b *testing.B) {
	// One request splits into many batches.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), 10000, RequestSizerTypeItems, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeLogs(b *testing.B) {
	// One request splits into many batches.
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), logsMarshaler.LogsSize(testdata.GenerateLogs(10010)), RequestSizerTypeBytes, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
