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

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/model/pdata/logs"
)

var (
	TestLogTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestLogTimestamp = pdata.NewTimestampFromTime(TestLogTime)
)

func GenerateLogsOneEmptyResourceLogs() logs.Logs {
	ld := logs.New()
	ld.ResourceLogs().AppendEmpty()
	return ld
}

func GenerateLogsNoLogRecords() logs.Logs {
	ld := GenerateLogsOneEmptyResourceLogs()
	initResource1(ld.ResourceLogs().At(0).Resource())
	return ld
}

func GenerateLogsOneEmptyLogRecord() logs.Logs {
	ld := GenerateLogsNoLogRecords()
	rs0 := ld.ResourceLogs().At(0)
	rs0.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	return ld
}

func GenerateLogsOneLogRecord() logs.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	fillLogOne(ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0))
	return ld
}

func GenerateLogsTwoLogRecordsSameResource() logs.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	logs := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	fillLogOne(logs.At(0))
	fillLogTwo(logs.AppendEmpty())
	return ld
}

func GenerateLogsTwoLogRecordsSameResourceOneDifferent() logs.Logs {
	ld := logs.New()
	rl0 := ld.ResourceLogs().AppendEmpty()
	initResource1(rl0.Resource())
	logs := rl0.ScopeLogs().AppendEmpty().LogRecords()
	fillLogOne(logs.AppendEmpty())
	fillLogTwo(logs.AppendEmpty())
	rl1 := ld.ResourceLogs().AppendEmpty()
	initResource2(rl1.Resource())
	fillLogThree(rl1.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty())
	return ld
}
func fillLogOne(log logs.LogRecord) {
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(logs.SeverityNumberINFO)
	log.SetSeverityText("Info")
	log.SetSpanID(pdata.NewSpanID([8]byte{0x01, 0x02, 0x04, 0x08}))
	log.SetTraceID(pdata.NewTraceID([16]byte{0x08, 0x04, 0x02, 0x01}))

	attrs := log.Attributes()
	attrs.InsertString("app", "server")
	attrs.InsertInt("instance_num", 1)

	log.Body().SetStringVal("This is a log message")
}

func fillLogTwo(log logs.LogRecord) {
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(logs.SeverityNumberINFO)
	log.SetSeverityText("Info")

	attrs := log.Attributes()
	attrs.InsertString("customer", "acme")
	attrs.InsertString("env", "dev")

	log.Body().SetStringVal("something happened")
}

func fillLogThree(log logs.LogRecord) {
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(logs.SeverityNumberWARN)
	log.SetSeverityText("Warning")

	log.Body().SetStringVal("something else happened")
}

func GenerateLogsManyLogRecordsSameResource(count int) logs.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	rls := ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	rls.EnsureCapacity(count)
	for i := 0; i < count; i++ {
		var l logs.LogRecord
		if i < rls.Len() {
			l = rls.At(i)
		} else {
			l = rls.AppendEmpty()
		}

		if i%2 == 0 {
			fillLogOne(l)
		} else {
			fillLogTwo(l)
		}
	}
	return ld
}
