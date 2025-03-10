// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeLogs(t *testing.T) {
	lr1 := newLogsRequest(testdata.GenerateLogs(2), nil)
	lr2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	res, err := lr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeLogsInvalidInput(t *testing.T) {
	lr1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	lr2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := lr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, lr2)
	require.Error(t, err)
}

func TestMergeSplitLogs(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.SizeConfig
		lr1      Request
		lr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(plog.NewLogs(), nil),
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(testdata.GenerateLogs(5), nil),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:  newLogsRequest(testdata.GenerateLogs(4), nil),
			lr2:  newLogsRequest(testdata.GenerateLogs(6), nil),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}(), nil)},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
			lr1:  newLogsRequest(plog.NewLogs(), nil),
			lr2:  newLogsRequest(testdata.GenerateLogs(10), nil),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(testdata.GenerateLogs(2), nil),
			},
		},
		{
			name: "merge_and_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:  newLogsRequest(testdata.GenerateLogs(8), nil),
			lr2:  newLogsRequest(testdata.GenerateLogs(20), nil),
			expected: []Request{
				newLogsRequest(func() plog.Logs {
					logs := testdata.GenerateLogs(8)
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
					return logs
				}(), nil),
				newLogsRequest(testdata.GenerateLogs(10), nil),
				newLogsRequest(testdata.GenerateLogs(8), nil),
			},
		},
		{
			name: "scope_logs_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}(), nil),
			lr2: newLogsRequest(testdata.GenerateLogs(2), nil),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(func() plog.Logs {
					ld := testdata.GenerateLogs(0)
					ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("extra log")
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(ld.ResourceLogs())
					return ld
				}(), nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.MergeSplit(context.Background(), tt.cfg, tt.lr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*logsRequest).ld, res[i].(*logsRequest).ld)
			}
		})
	}
}

func TestMergeSplitLogsBasedOnByteSize(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.SizeConfig
		lr1      Request
		lr2      Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(plog.NewLogs(), nil),
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(testdata.GenerateLogs(5), nil),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(11))},
			lr1:  newLogsRequest(testdata.GenerateLogs(4), nil),
			lr2:  newLogsRequest(testdata.GenerateLogs(6), nil),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}(), nil)},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4))},
			lr1:  newLogsRequest(plog.NewLogs(), nil),
			lr2:  newLogsRequest(testdata.GenerateLogs(10), nil),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(testdata.GenerateLogs(2), nil),
			},
		},
		{
			name: "merge_and_split",
			cfg: exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeBytes,
				MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))/2 + logsMarshaler.LogsSize(testdata.GenerateLogs(11))/2,
			},
			lr1: newLogsRequest(testdata.GenerateLogs(8), nil),
			lr2: newLogsRequest(testdata.GenerateLogs(20), nil),
			expected: []Request{
				newLogsRequest(func() plog.Logs {
					logs := testdata.GenerateLogs(8)
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
					return logs
				}(), nil),
				newLogsRequest(testdata.GenerateLogs(10), nil),
				newLogsRequest(testdata.GenerateLogs(8), nil),
			},
		},
		{
			name: "scope_logs_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4))},
			lr1: newLogsRequest(func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}(), nil),
			lr2: newLogsRequest(testdata.GenerateLogs(2), nil),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4), nil),
				newLogsRequest(func() plog.Logs {
					ld := testdata.GenerateLogs(0)
					ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("extra log")
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(ld.ResourceLogs())
					return ld
				}(), nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.MergeSplit(context.Background(), tt.cfg, tt.lr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i := range res {
				assert.Equal(t, tt.expected[i].(*logsRequest).ld, res[i].(*logsRequest).ld)
			}
		})
	}
}

func TestMergeSplitLogsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newLogsRequest(testdata.GenerateLogs(18), nil)
	r2 := newTracesRequest(testdata.GenerateTraces(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestMergeSplitLogsInvalidInput(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	r2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{}, r2)
	require.Error(t, err)
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
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	merged := []Request{newLogsRequest(testdata.GenerateLogs(1), nil)}
	for j := 0; j < 1000; j++ {
		lr2 := newLogsRequest(testdata.GenerateLogs(10), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestMergeSplitExactBytes(t *testing.T) {
	pb := plog.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: pb.LogsSize(testdata.GenerateLogs(2)) - 1}
	lr := newLogsRequest(testdata.GenerateLogs(4), nil)
	merged, err := lr.MergeSplit(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 1}
	lr := newLogsRequest(testdata.GenerateLogs(4), nil)
	merged, err := lr.MergeSplit(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10010}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10), nil)}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnByteSizeManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(11000))}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10), nil)}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0), nil)}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10000))}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0), nil)}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001), nil)
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			assert.Len(b, res, 2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}

func BenchmarkSplittingBasedOnItemCountHugeLogs(b *testing.B) {
	// One request splits into many batches.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0), nil)}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeLogs(b *testing.B) {
	// One request splits into many batches.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10010))}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0), nil)}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000), nil)
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}
