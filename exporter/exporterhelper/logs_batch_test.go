// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/internal"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeLogs(t *testing.T) {
	lr1 := newLogsRequest(testdata.GenerateLogs(2), nil)
	lr2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	res, err := lr1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].ItemsCount())
}

func TestMergeLogsInvalidInput(t *testing.T) {
	lr1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	lr2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := lr1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, lr2)
	require.Error(t, err)
}

func TestMergeSplitLogs(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		lr1      internal.Request
		lr2      internal.Request
		expected []Request
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(plog.NewLogs(), nil),
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      newLogsRequest(testdata.GenerateLogs(5), nil),
			expected: []Request{newLogsRequest(testdata.GenerateLogs(5), nil)},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      newLogsRequest(plog.NewLogs(), nil),
			lr2:      nil,
			expected: []Request{newLogsRequest(plog.NewLogs(), nil)},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
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
	// Magic number is the byte size testdata.GenerateLogs(10)
	tests := []struct {
		name     string
		cfg      int
		lr1      internal.Request
		lr2      internal.Request
		expected []*logsRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: plog.NewLogs()},
			expected: []*logsRequest{{ld: plog.NewLogs()}},
		},
		{
			name:     "first_request_empty",
			cfg:      logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: testdata.GenerateLogs(5)},
			expected: []*logsRequest{{ld: testdata.GenerateLogs(5)}},
		},
		// TODO: add this back once mergeSplitBasedOnByteSize can be triggered from MergeSplit()
		// {
		// 	name:     "first_empty_second_nil",
		// 	cfg:      logsMarshaler.LogsSize(testdata.GenerateLogs(10)),
		// 	lr1:      &logsRequest{ld: plog.NewLogs()},
		// 	lr2:      nil,
		// 	expected: []*logsRequest{{ld: plog.NewLogs()}},
		// },
		{
			name: "merge_only",
			cfg:  logsMarshaler.LogsSize(testdata.GenerateLogs(11)),
			lr1:  &logsRequest{ld: testdata.GenerateLogs(4)},
			lr2:  &logsRequest{ld: testdata.GenerateLogs(6)},
			expected: []*logsRequest{{ld: func() plog.Logs {
				logs := testdata.GenerateLogs(4)
				testdata.GenerateLogs(6).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
				return logs
			}()}},
		},
		{
			name: "split_only",
			cfg:  logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1:  &logsRequest{ld: plog.NewLogs()},
			lr2:  &logsRequest{ld: testdata.GenerateLogs(10)},
			expected: []*logsRequest{
				{ld: testdata.GenerateLogs(4)},
				{ld: testdata.GenerateLogs(4)},
				{ld: testdata.GenerateLogs(2)},
			},
		},
		{
			name: "merge_and_split",
			cfg:  (logsMarshaler.LogsSize(testdata.GenerateLogs(10)) + logsMarshaler.LogsSize(testdata.GenerateLogs(11))) / 2,
			lr1:  &logsRequest{ld: testdata.GenerateLogs(8)},
			lr2:  &logsRequest{ld: testdata.GenerateLogs(20)},
			expected: []*logsRequest{
				{ld: func() plog.Logs {
					logs := testdata.GenerateLogs(8)
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(logs.ResourceLogs())
					return logs
				}()},
				{ld: testdata.GenerateLogs(10)},
				{ld: testdata.GenerateLogs(8)},
			},
		},
		{
			name: "scope_logs_split",
			cfg:  logsMarshaler.LogsSize(testdata.GenerateLogs(4)),
			lr1: &logsRequest{ld: func() plog.Logs {
				ld := testdata.GenerateLogs(4)
				ld.ResourceLogs().At(0).ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("extra log")
				return ld
			}()},
			lr2: &logsRequest{ld: testdata.GenerateLogs(2)},
			expected: []*logsRequest{
				{ld: testdata.GenerateLogs(4)},
				{ld: func() plog.Logs {
					ld := testdata.GenerateLogs(0)
					ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().AppendEmpty().Body().SetStr("extra log")
					testdata.GenerateLogs(2).ResourceLogs().MoveAndAppendTo(ld.ResourceLogs())
					return ld
				}()},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.lr1.(*logsRequest).mergeSplitBasedOnByteSize(tt.cfg, tt.lr2.(*logsRequest))
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i].ld, r.(*logsRequest).ld)
				assert.Equal(t, r.(*logsRequest).ByteSize(), logsMarshaler.LogsSize(r.(*logsRequest).ld))
			}
		})
	}
}

func TestMergeSplitLogsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := newLogsRequest(testdata.GenerateLogs(18), nil)
	r2 := newTracesRequest(testdata.GenerateTraces(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{MaxSizeItems: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestMergeSplitLogsInvalidInput(t *testing.T) {
	r1 := newTracesRequest(testdata.GenerateTraces(2), nil)
	r2 := newLogsRequest(testdata.GenerateLogs(3), nil)
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, r2)
	require.Error(t, err)
}

func TestExtractLogs(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateLogs(10)
		extractedLogs := extractLogsBasedOnItemCount(ld, i)
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
	}
}

func TestExtractLogsBasedOnByteSize(t *testing.T) {
	for i := 0; i < 11; i++ {
		ld := testdata.GenerateLogs(10)
		byteSize := logsMarshaler.LogsSize(testdata.GenerateLogs(i))
		extractedLogs, capacityReached := extractLogsBasedOnByteSize(ld, byteSize)
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
		if i < 10 {
			assert.True(t, capacityReached)
		} else {
			assert.False(t, capacityReached)
		}
		assert.GreaterOrEqual(t, byteSize, logsMarshaler.LogsSize(extractedLogs))
	}
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	// All requests merge into a single batch.
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		merged := []Request{&logsRequest{ld: testdata.GenerateLogs(10)}}
		for j := 0; j < 1000; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10)}
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 1)
	}
}

func BenchmarkSplittingBasedOnByteSizeManySmallLogs(b *testing.B) {
	maxSizeBytes := logsMarshaler.LogsSize(testdata.GenerateLogs(1010000))
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10)}
		for j := 0; j < 1000; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10)}
			lr1.mergeSplitBasedOnByteSize(maxSizeBytes, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyAboveLimit(b *testing.B) {
	// Every incoming request results in a split.
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		merged := []Request{&logsRequest{ld: testdata.GenerateLogs(0)}}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10001)}
			res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
			merged = append(merged[0:len(merged)-1], res...)
		}
		assert.Len(b, merged, 11)
	}
}
func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyAboveLimit(b *testing.B) {
	maxSizeBytes := logsMarshaler.LogsSize(testdata.GenerateLogs(10000))
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10001)}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10001)}
			lr1.mergeSplitBasedOnByteSize(maxSizeBytes, lr2)
		}
	}
}
func BenchmarkSplittingBasedOnItemCountHugeLogs(b *testing.B) {
	// One request splits into many batches.
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		merged := []Request{&logsRequest{ld: testdata.GenerateLogs(0)}}
		lr2 := &logsRequest{ld: testdata.GenerateLogs(100000)}
		res, _ := merged[len(merged)-1].MergeSplit(context.Background(), cfg, lr2)
		merged = append(merged[0:len(merged)-1], res...)
		assert.Len(b, merged, 10)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeLog(b *testing.B) {
	maxSizeBytes := logsMarshaler.LogsSize(testdata.GenerateLogs(10000))
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(1)}
		lr2 := &logsRequest{ld: testdata.GenerateLogs(100000)}
		lr1.mergeSplitBasedOnByteSize(maxSizeBytes, lr2)
	}
}
