// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0
package sizer // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/sizer"

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestLogsCountSizer(t *testing.T) {
	ld := testdata.GenerateLogs(5)
	sizer := LogsCountSizer{}
	require.Equal(t, 5, sizer.LogsSize(ld))

	rl := ld.ResourceLogs().At(0)
	require.Equal(t, 5, sizer.ResourceLogsSize(rl))

	sl := rl.ScopeLogs().At(0)
	require.Equal(t, 5, sizer.ScopeLogsSize(sl))

	require.Equal(t, 1, sizer.LogRecordSize(sl.LogRecords().At(0)))
	require.Equal(t, 1, sizer.LogRecordSize(sl.LogRecords().At(1)))
	require.Equal(t, 1, sizer.LogRecordSize(sl.LogRecords().At(2)))
	require.Equal(t, 1, sizer.LogRecordSize(sl.LogRecords().At(3)))
	require.Equal(t, 1, sizer.LogRecordSize(sl.LogRecords().At(4)))

	prevSize := sizer.ScopeLogsSize(sl)
	lr := sl.LogRecords().At(2)
	lr.CopyTo(sl.LogRecords().AppendEmpty())
	require.Equal(t, sizer.ScopeLogsSize(sl), prevSize+sizer.DeltaSize(sizer.LogRecordSize(lr)))
}

func TestLogsBytesSizer(t *testing.T) {
	ld := testdata.GenerateLogs(5)
	sizer := LogsBytesSizer{}
	require.Equal(t, 545, sizer.LogsSize(ld))

	rl := ld.ResourceLogs().At(0)
	require.Equal(t, 542, sizer.ResourceLogsSize(rl))

	sl := rl.ScopeLogs().At(0)
	require.Equal(t, 497, sizer.ScopeLogsSize(sl))

	require.Equal(t, 109, sizer.LogRecordSize(sl.LogRecords().At(0)))
	require.Equal(t, 79, sizer.LogRecordSize(sl.LogRecords().At(1)))
	require.Equal(t, 109, sizer.LogRecordSize(sl.LogRecords().At(2)))
	require.Equal(t, 79, sizer.LogRecordSize(sl.LogRecords().At(3)))
	require.Equal(t, 109, sizer.LogRecordSize(sl.LogRecords().At(4)))

	prevSize := sizer.ScopeLogsSize(sl)
	lr := sl.LogRecords().At(2)
	lr.CopyTo(sl.LogRecords().AppendEmpty())
	require.Equal(t, sizer.ScopeLogsSize(sl), prevSize+sizer.DeltaSize(sizer.LogRecordSize(lr)))
}
