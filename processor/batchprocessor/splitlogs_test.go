// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/testdata"
)

func TestSplitLogs_noop(t *testing.T) {
	td := testdata.GenerateLogs(20)
	splitSize := 40
	split := splitLogs(splitSize, td)
	assert.Equal(t, td, split)

	i := 0
	td.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().RemoveIf(func(plog.LogRecord) bool {
		i++
		return i > 5
	})
	assert.Equal(t, td, split)
}

func TestSplitLogs(t *testing.T) {
	ld := testdata.GenerateLogs(20)
	logs := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	cp := plog.NewLogs()
	cpLogs := cp.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()
	cpLogs.EnsureCapacity(5)
	ld.ResourceLogs().At(0).Resource().CopyTo(
		cp.ResourceLogs().At(0).Resource())
	ld.ResourceLogs().At(0).ScopeLogs().At(0).Scope().CopyTo(
		cp.ResourceLogs().At(0).ScopeLogs().At(0).Scope())
	logs.At(0).CopyTo(cpLogs.AppendEmpty())
	logs.At(1).CopyTo(cpLogs.AppendEmpty())
	logs.At(2).CopyTo(cpLogs.AppendEmpty())
	logs.At(3).CopyTo(cpLogs.AppendEmpty())
	logs.At(4).CopyTo(cpLogs.AppendEmpty())

	splitSize := 5
	split := splitLogs(splitSize, ld)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, cp, split)
	assert.Equal(t, 15, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-4", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 10, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-5", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-9", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 5, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-10", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-14", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 5, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-15", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-19", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}

func TestSplitLogsMultipleResourceLogs(t *testing.T) {
	td := testdata.GenerateLogs(20)
	logs := td.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	// add second index to resource logs
	testdata.GenerateLogs(20).
		ResourceLogs().At(0).CopyTo(td.ResourceLogs().AppendEmpty())
	logs = td.ResourceLogs().At(1).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(1, i))
	}

	splitSize := 5
	split := splitLogs(splitSize, td)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, 35, td.LogRecordCount())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-4", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}

func TestSplitLogsMultipleResourceLogs_split_size_greater_than_log_size(t *testing.T) {
	td := testdata.GenerateLogs(20)
	logs := td.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	// add second index to resource logs
	testdata.GenerateLogs(20).
		ResourceLogs().At(0).CopyTo(td.ResourceLogs().AppendEmpty())
	logs = td.ResourceLogs().At(1).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(1, i))
	}

	splitSize := 25
	split := splitLogs(splitSize, td)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, 40-splitSize, td.LogRecordCount())
	assert.Equal(t, 1, td.ResourceLogs().Len())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-19", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(19).SeverityText())
	assert.Equal(t, "test-log-int-1-0", split.ResourceLogs().At(1).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-1-4", split.ResourceLogs().At(1).ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}

func TestSplitLogsMultipleILL(t *testing.T) {
	td := testdata.GenerateLogs(20)
	logs := td.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(0, i))
	}
	// add second index to ILL
	td.ResourceLogs().At(0).ScopeLogs().At(0).
		CopyTo(td.ResourceLogs().At(0).ScopeLogs().AppendEmpty())
	logs = td.ResourceLogs().At(0).ScopeLogs().At(1).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(1, i))
	}

	// add third index to ILL
	td.ResourceLogs().At(0).ScopeLogs().At(0).
		CopyTo(td.ResourceLogs().At(0).ScopeLogs().AppendEmpty())
	logs = td.ResourceLogs().At(0).ScopeLogs().At(2).LogRecords()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetSeverityText(getTestLogSeverityText(2, i))
	}

	splitSize := 40
	split := splitLogs(splitSize, td)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, 20, td.LogRecordCount())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).SeverityText())
	assert.Equal(t, "test-log-int-0-4", split.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(4).SeverityText())
}
