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
				assert.Equal(t, tt.expected[i], res[i])
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
		extractedLogs := extractLogs(ld, i)
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
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
