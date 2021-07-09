// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package batchprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestSplitLogs_noop(t *testing.T) {
	td := testdata.GenerateLogsManyLogRecordsSameResource(20)
	splitSize := 40
	split := splitLogs(splitSize, td)
	assert.Equal(t, td, split)

	i := 0
	td.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().RemoveIf(func(_ pdata.LogRecord) bool {
		i++
		return i > 5
	})
	assert.EqualValues(t, td, split)
}

func TestSplitLogs(t *testing.T) {
	ld := testdata.GenerateLogsManyLogRecordsSameResource(20)
	logs := ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetName(getTestLogName(0, i))
	}
	cp := pdata.NewLogs()
	cpLogs := cp.ResourceLogs().AppendEmpty().InstrumentationLibraryLogs().AppendEmpty().Logs()
	cpLogs.EnsureCapacity(5)
	ld.ResourceLogs().At(0).Resource().CopyTo(
		cp.ResourceLogs().At(0).Resource())
	ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).InstrumentationLibrary().CopyTo(
		cp.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).InstrumentationLibrary())
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
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-4", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 10, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-5", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-9", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 5, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-10", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-14", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())

	split = splitLogs(splitSize, ld)
	assert.Equal(t, 5, ld.LogRecordCount())
	assert.Equal(t, "test-log-int-0-15", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-19", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())
}

func TestSplitLogsMultipleResourceLogs(t *testing.T) {
	td := testdata.GenerateLogsManyLogRecordsSameResource(20)
	logs := td.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetName(getTestLogName(0, i))
	}
	// add second index to resource logs
	testdata.GenerateLogsManyLogRecordsSameResource(20).
		ResourceLogs().At(0).CopyTo(td.ResourceLogs().AppendEmpty())
	logs = td.ResourceLogs().At(1).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetName(getTestLogName(1, i))
	}

	splitSize := 5
	split := splitLogs(splitSize, td)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, 35, td.LogRecordCount())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-4", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())
}

func TestSplitLogsMultipleResourceLogs_split_size_greater_than_log_size(t *testing.T) {
	td := testdata.GenerateLogsManyLogRecordsSameResource(20)
	logs := td.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetName(getTestLogName(0, i))
	}
	// add second index to resource logs
	testdata.GenerateLogsManyLogRecordsSameResource(20).
		ResourceLogs().At(0).CopyTo(td.ResourceLogs().AppendEmpty())
	logs = td.ResourceLogs().At(1).InstrumentationLibraryLogs().At(0).Logs()
	for i := 0; i < logs.Len(); i++ {
		logs.At(i).SetName(getTestLogName(1, i))
	}

	splitSize := 25
	split := splitLogs(splitSize, td)
	assert.Equal(t, splitSize, split.LogRecordCount())
	assert.Equal(t, 40-splitSize, td.LogRecordCount())
	assert.Equal(t, 1, td.ResourceLogs().Len())
	assert.Equal(t, "test-log-int-0-0", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-0-19", split.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(19).Name())
	assert.Equal(t, "test-log-int-1-0", split.ResourceLogs().At(1).InstrumentationLibraryLogs().At(0).Logs().At(0).Name())
	assert.Equal(t, "test-log-int-1-4", split.ResourceLogs().At(1).InstrumentationLibraryLogs().At(0).Logs().At(4).Name())
}

func BenchmarkSplitLogs(b *testing.B) {
	md := pdata.NewLogs()
	rms := md.ResourceLogs()
	for i := 0; i < 20; i++ {
		testdata.GenerateLogsManyLogRecordsSameResource(20).ResourceLogs().MoveAndAppendTo(md.ResourceLogs())
		ms := rms.At(rms.Len() - 1).InstrumentationLibraryLogs().At(0).Logs()
		for i := 0; i < ms.Len(); i++ {
			ms.At(i).SetName(getTestLogName(1, i))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cloneReq := md.Clone()
		split := splitLogs(128, cloneReq)
		if split.LogRecordCount() != 128 || cloneReq.LogRecordCount() != 400-128 {
			b.Fail()
		}
	}
}

func BenchmarkCloneLogs(b *testing.B) {
	md := pdata.NewLogs()
	rms := md.ResourceLogs()
	for i := 0; i < 20; i++ {
		testdata.GenerateLogsManyLogRecordsSameResource(20).ResourceLogs().MoveAndAppendTo(md.ResourceLogs())
		ms := rms.At(rms.Len() - 1).InstrumentationLibraryLogs().At(0).Logs()
		for i := 0; i < ms.Len(); i++ {
			ms.At(i).SetName(getTestLogName(1, i))
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cloneReq := md.Clone()
		if cloneReq.LogRecordCount() != 400 {
			b.Fail()
		}
	}
}
