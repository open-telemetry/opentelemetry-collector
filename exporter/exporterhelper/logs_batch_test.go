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
	lr1 := newLogsRequest(testdata.GenerateLogs(2))
	lr2 := newLogsRequest(testdata.GenerateLogs(3))
	res, err := lr1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems}, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].ItemsCount())
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
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:  newLogsRequest(testdata.GenerateLogs(4)),
			lr2:  newLogsRequest(testdata.GenerateLogs(6)),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
			lr1:  newLogsRequest(plog.NewLogs()),
			lr2:  newLogsRequest(testdata.GenerateLogs(10)),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name: "merge_and_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10},
			lr1:  newLogsRequest(testdata.GenerateLogs(8)),
			lr2:  newLogsRequest(testdata.GenerateLogs(20)),
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
			name: "scope_logs_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 4},
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
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(plog.NewLogs()),
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))},
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      newLogsRequest(testdata.GenerateLogs(5)),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5))},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))},
			lr1:      newLogsRequest(plog.NewLogs()),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs())},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(11))},
			lr1:  newLogsRequest(testdata.GenerateLogs(4)),
			lr2:  newLogsRequest(testdata.GenerateLogs(6)),
			expected: []Request{newLogsRequest(func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}())},
		},
		{
			name: "split_only",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4))},
			lr1:  newLogsRequest(plog.NewLogs()),
			lr2:  newLogsRequest(testdata.GenerateLogs(10)),
			expected: []Request{
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(4)),
				newLogsRequest(testdata.GenerateLogs(2)),
			},
		},
		{
			name: "merge_and_split",
			cfg: exporterbatcher.SizeConfig{
				Sizer:   exporterbatcher.SizerTypeBytes,
				MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(10))/2 + logsMarshaler.LogsSize(testdata.GenerateLogs(11))/2,
			},
			lr1: newLogsRequest(testdata.GenerateLogs(8)),
			lr2: newLogsRequest(testdata.GenerateLogs(20)),
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
			name: "scope_logs_split",
			cfg:  exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: logsMarshaler.LogsSize(testdata.GenerateLogs(4))},
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
	r1 := newLogsRequest(testdata.GenerateLogs(18))
	r2 := newTracesRequest(testdata.GenerateTraces(3))
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10}, r2)
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
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10000}
	merged := []Request{newLogsRequest(testdata.GenerateLogs(1))}
	for j := 0; j < 1000; j++ {
		lr2 := newLogsRequest(testdata.GenerateLogs(10))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
	}
	assert.Len(t, merged, 2)
}

func TestLogsMergeSplitExactBytes(t *testing.T) {
	pb := plog.ProtoMarshaler{}
	// Set max size off by 1, so forces every log to be it's own batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeBytes, MaxSize: pb.LogsSize(testdata.GenerateLogs(2)) - 1}
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func TestLogsMergeSplitExactItems(t *testing.T) {
	// Set max size off by 1, so forces every log to be it's own batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 1}
	lr := newLogsRequest(testdata.GenerateLogs(4))
	merged, err := lr.MergeSplit(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Len(t, merged, 4)
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.SizeConfig{Sizer: exporterbatcher.SizerTypeItems, MaxSize: 10010}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
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
		merged := []Request{newLogsRequest(testdata.GenerateLogs(10))}
		for j := 0; j < 1000; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10))
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
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
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
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		for j := 0; j < 10; j++ {
			lr2 := newLogsRequest(testdata.GenerateLogs(10001))
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
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
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
		merged := []Request{newLogsRequest(testdata.GenerateLogs(0))}
		lr2 := newLogsRequest(testdata.GenerateLogs(100000))
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func TestLogsRequest_MergeSplit_UnknownSizerType(t *testing.T) {
	// Create a logs request
	req := newLogsRequest(plog.NewLogs())

	// Create config with invalid sizer type by using zero value
	cfg := exporterbatcher.SizeConfig{
		Sizer: exporterbatcher.SizerType{}, // Empty struct will have empty string as val
	}

	// Call MergeSplit with invalid sizer
	result, err := req.MergeSplit(context.Background(), cfg, nil)

	// Verify results
	assert.Nil(t, result)
	assert.EqualError(t, err, "unknown sizer type")
}
