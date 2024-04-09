// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestMergeLogs(t *testing.T) {
	lr1 := &logsRequest{ld: testdata.GenerateLogs(2)}
	lr2 := &logsRequest{ld: testdata.GenerateLogs(3)}
	res, err := mergeLogs(context.Background(), lr1, lr2)
	assert.Nil(t, err)
	assert.Equal(t, 5, res.(*logsRequest).ld.LogRecordCount())
}

func TestMergeLogsInvalidInput(t *testing.T) {
	lr1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	lr2 := &logsRequest{ld: testdata.GenerateLogs(3)}
	_, err := mergeLogs(context.Background(), lr1, lr2)
	assert.Error(t, err)
}

func TestMergeSplitLogs(t *testing.T) {
	tests := []struct {
		name     string
		cfg      exporterbatcher.MaxSizeConfig
		lr1      Request
		lr2      Request
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
			name:     "both_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      nil,
			lr2:      nil,
			expected: []*logsRequest{},
		},
		{
			name:     "first_request_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      &logsRequest{ld: plog.NewLogs()},
			lr2:      &logsRequest{ld: testdata.GenerateLogs(5)},
			expected: []*logsRequest{{ld: testdata.GenerateLogs(5)}},
		},
		{
			name:     "first_requests_nil",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      nil,
			lr2:      &logsRequest{ld: testdata.GenerateLogs(5)},
			expected: []*logsRequest{{ld: testdata.GenerateLogs(5)}},
		},
		{
			name:     "first_nil_second_empty",
			cfg:      exporterbatcher.MaxSizeConfig{MaxSizeItems: 10},
			lr1:      nil,
			lr2:      &logsRequest{ld: plog.NewLogs()},
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
			lr1:  nil,
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
			res, err := mergeSplitLogs(context.Background(), tt.cfg, tt.lr1, tt.lr2)
			assert.Nil(t, err)
			assert.Equal(t, len(tt.expected), len(res))
			for i, r := range res {
				assert.Equal(t, tt.expected[i], r.(*logsRequest))
			}
		})

	}
}

func TestMergeSplitLogsInvalidInput(t *testing.T) {
	r1 := &tracesRequest{td: testdata.GenerateTraces(2)}
	r2 := &logsRequest{ld: testdata.GenerateLogs(3)}
	_, err := mergeSplitLogs(context.Background(), exporterbatcher.MaxSizeConfig{}, r1, r2)
	assert.Error(t, err)
}

func TestExtractLogs(t *testing.T) {
	for i := 0; i < 10; i++ {
		ld := testdata.GenerateLogs(10)
		extractedLogs := extractLogs(ld, i)
		assert.Equal(t, i, extractedLogs.LogRecordCount())
		assert.Equal(t, 10-i, ld.LogRecordCount())
	}
}
