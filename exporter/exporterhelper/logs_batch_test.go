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
	lr1 := &logsRequest{ld: testdata.GenerateLogs(2)}
	lr2 := &logsRequest{ld: testdata.GenerateLogs(3)}
	res, err := lr1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, lr2)
	require.NoError(t, err)
	require.Equal(t, 5, res[0].(*logsRequest).ld.LogRecordCount())
}

func TestMergeLogsInvalidInput(t *testing.T) {
	lr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	lr2 := &logsRequest{ld: testdata.GenerateLogs(3)}
	_, err := lr1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{}, lr2)
	require.Error(t, err)
}

func TestMergeSplitLogsBasedOnItemCount(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		lr1      internal.Request
		lr2      internal.Request
		expected []*logsRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: plog.NewLogs()},
			expected: []*logsRequest{{ld: plog.NewLogs()}},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: testdata.GenerateLogs(5)},
			expected: []*logsRequest{{ld: testdata.GenerateLogs(5)}},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      nil,
			expected: []*logsRequest{{ld: plog.NewLogs()}},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeItems: 4},
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
			res, err := tt.lr1.MergeSplit(context.Background(), tt.cfg, tt.lr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i], r.(*logsRequest))
			}
		})
	}
}

func TestMergeSplitLogsInputNotModifiedIfErrorReturned(t *testing.T) {
	r1 := &logsRequest{ld: testdata.GenerateLogs(18)}
	r2 := &tracesRequest{td: testdata.GenerateTraces(3)}
	_, err := r1.MergeSplit(context.Background(), exporterbatcher.MaxSizeConfig{MaxSizeItems: 10}, r2)
	require.Error(t, err)
	assert.Equal(t, 18, r1.ItemsCount())
}

func TestMergeSplitLogsInvalidInput(t *testing.T) {
	r1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	r2 := &logsRequest{ld: testdata.GenerateLogs(3)}
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

func TestMergeSplitLogsBasedOnByteSize(t *testing.T) {
	// Magic number is the byte size testdata.GenerateLogs(10)
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		lr1      internal.Request
		lr2      internal.Request
		expected []*logsRequest
	}{
		{
			name:     "both_requests_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(10).ByteSize()},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: plog.NewLogs()},
			expected: []*logsRequest{{ld: plog.NewLogs()}},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(10).ByteSize()},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: testdata.GenerateLogs(5)},
			expected: []*logsRequest{{ld: testdata.GenerateLogs(5)}},
		},
		{
			name:     "first_empty_second_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(10).ByteSize()},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      nil,
			expected: []*logsRequest{{ld: plog.NewLogs()}},
		},
		{
			name: "merge_only",
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(11).ByteSize()},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(4).ByteSize()},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeBytes: (testdata.GenerateLogs(10).ByteSize() + testdata.GenerateLogs(11).ByteSize()) / 2},
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
			cfg:  exporterbatcher.MaxSizeConfig{MaxSizeBytes: testdata.GenerateLogs(4).ByteSize()},
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
			res, err := tt.lr1.MergeSplit(context.Background(), tt.cfg, tt.lr2)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i], r.(*logsRequest))
			}
		})
	}
}

func BenchmarkSplittingBasedOnItemCountManySmallLogs(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10)}
		for j := 0; j < 1000; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnByteSizeManySmallLogs(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeBytes: 1010000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10)}
		for j := 0; j < 1000; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyAboveLimit(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10001)}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10001)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyAboveLimit(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeBytes: 960052} // 960052 corresponds to 10000 generated logs
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(10001)}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(10001)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnItemCountManyLogsSlightlyBelowLimit(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(9999)}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(9999)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnByteSizeManyLogsSlightlyBelowLimit(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeBytes: 960052} // 960052 corresponds to 10000 generated logs
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(9999)}
		for j := 0; j < 10; j++ {
			lr2 := &logsRequest{ld: testdata.GenerateLogs(9999)}
			lr1.MergeSplit(context.Background(), cfg, lr2)
		}
	}
}

func BenchmarkSplittingBasedOnItemCountHugeLog(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeItems: 10000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(1)}
		lr2 := &logsRequest{ld: testdata.GenerateLogs(100000)} // l2 is of size 9.600054 MB
		lr1.MergeSplit(context.Background(), cfg, lr2)
	}
}

func BenchmarkSplittingBasedOnByteSizeHugeLog(b *testing.B) {
	cfg := exporterbatcher.MaxSizeConfig{MaxSizeBytes: 970000}
	for i := 0; i < b.N; i++ {
		lr1 := &logsRequest{ld: testdata.GenerateLogs(1)}
		lr2 := &logsRequest{ld: testdata.GenerateLogs(100000)}
		lr1.MergeSplit(context.Background(), cfg, lr2)
	}
}
