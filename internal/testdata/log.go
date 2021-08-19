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
)

var (
	TestLogTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestLogTimestamp = pdata.NewTimestampFromTime(TestLogTime)
)

func GenerateLogsOneEmptyResourceLogs() pdata.Logs {
	ld := pdata.NewLogs()
	ld.ResourceLogs().AppendEmpty()
	return ld
}

func GenerateLogsNoLogRecords() pdata.Logs {
	ld := GenerateLogsOneEmptyResourceLogs()
	initResource1(ld.ResourceLogs().At(0).Resource())
	return ld
}

func GenerateLogsOneEmptyLogRecord() pdata.Logs {
	ld := GenerateLogsNoLogRecords()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().AppendEmpty().Logs().AppendEmpty()
	return ld
}

func GenerateLogsOneLogRecordNoResource() pdata.Logs {
	ld := GenerateLogsOneEmptyResourceLogs()
	rs0 := ld.ResourceLogs().At(0)
	fillLogOne(rs0.InstrumentationLibraryLogs().AppendEmpty().Logs().AppendEmpty())
	return ld
}

func GenerateLogsOneLogRecord() pdata.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	fillLogOne(ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0))
	return ld
}

func GenerateLogsTwoLogRecordsSameResource() pdata.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	logs := ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	fillLogOne(logs.At(0))
	fillLogTwo(logs.AppendEmpty())
	return ld
}

func GenerateLogsTwoLogRecordsSameResourceOneDifferent() pdata.Logs {
	ld := pdata.NewLogs()
	rl0 := ld.ResourceLogs().AppendEmpty()
	initResource1(rl0.Resource())
	logs := rl0.InstrumentationLibraryLogs().AppendEmpty().Logs()
	fillLogOne(logs.AppendEmpty())
	fillLogTwo(logs.AppendEmpty())
	rl1 := ld.ResourceLogs().AppendEmpty()
	initResource2(rl1.Resource())
	fillLogThree(rl1.InstrumentationLibraryLogs().AppendEmpty().Logs().AppendEmpty())
	return ld
}
func fillLogOne(log pdata.LogRecord) {
	log.SetName("logA")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(pdata.SeverityNumberINFO)
	log.SetSeverityText("Info")
	log.SetSpanID(pdata.NewSpanID([8]byte{0x01, 0x02, 0x04, 0x08}))
	log.SetTraceID(pdata.NewTraceID([16]byte{0x08, 0x04, 0x02, 0x01}))

	attrs := log.Attributes()
	attrs.InsertString("app", "server")
	attrs.InsertInt("instance_num", 1)

	log.Body().SetStringVal("This is a log message")
}

func fillLogTwo(log pdata.LogRecord) {
	log.SetName("logB")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(pdata.SeverityNumberINFO)
	log.SetSeverityText("Info")

	attrs := log.Attributes()
	attrs.InsertString("customer", "acme")
	attrs.InsertString("env", "dev")

	log.Body().SetStringVal("something happened")
}

func fillLogThree(log pdata.LogRecord) {
	log.SetName("logC")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(pdata.SeverityNumberWARN)
	log.SetSeverityText("Warning")

	log.Body().SetStringVal("something else happened")
}

func GenerateLogsManyLogRecordsSameResource(count int) pdata.Logs {
	ld := GenerateLogsOneEmptyLogRecord()
	logs := ld.ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs()
	logs.EnsureCapacity(count)
	for i := 0; i < count; i++ {
		var l pdata.LogRecord
		if i < logs.Len() {
			l = logs.At(i)
		} else {
			l = logs.AppendEmpty()
		}

		if i%2 == 0 {
			fillLogOne(l)
		} else {
			fillLogTwo(l)
		}
	}
	return ld
}
