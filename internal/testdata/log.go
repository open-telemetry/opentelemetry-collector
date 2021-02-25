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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlplogs "go.opentelemetry.io/collector/internal/data/protogen/logs/v1"
)

var (
	TestLogTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestLogTimestamp = pdata.TimestampFromTime(TestLogTime)
)

func GenerateLogDataEmpty() pdata.Logs {
	ld := pdata.NewLogs()
	return ld
}

func generateLogOtlpEmpty() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs(nil)
}

func GenerateLogDataOneEmptyResourceLogs() pdata.Logs {
	ld := GenerateLogDataEmpty()
	ld.ResourceLogs().Resize(1)
	return ld
}

func generateLogOtlpOneEmptyResourceLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{},
	}
}

func GenerateLogDataNoLogRecords() pdata.Logs {
	ld := GenerateLogDataOneEmptyResourceLogs()
	rs0 := ld.ResourceLogs().At(0)
	initResource1(rs0.Resource())
	return ld
}

func generateLogOtlpNoLogRecords() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func GenerateLogDataOneEmptyLogs() pdata.Logs {
	ld := GenerateLogDataNoLogRecords()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().Resize(1)
	rs0.InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	return ld
}

func generateLogOtlpOneEmptyLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						{},
					},
				},
			},
		},
	}
}

func GenerateLogDataOneLogNoResource() pdata.Logs {
	ld := GenerateLogDataOneEmptyResourceLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().Resize(1)
	rs0.InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	rs0lr0 := rs0.InstrumentationLibraryLogs().At(0).Logs().At(0)
	fillLogOne(rs0lr0)
	return ld
}

func generateLogOtlpOneLogNoResource() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						generateOtlpLogOne(),
					},
				},
			},
		},
	}
}

func GenerateLogDataOneLog() pdata.Logs {
	ld := GenerateLogDataOneEmptyLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().Resize(1)
	rs0.InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	rs0lr0 := rs0.InstrumentationLibraryLogs().At(0).Logs().At(0)
	fillLogOne(rs0lr0)
	return ld
}

func generateLogOtlpOneLog() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						generateOtlpLogOne(),
					},
				},
			},
		},
	}
}

func GenerateLogDataTwoLogsSameResource() pdata.Logs {
	ld := GenerateLogDataOneEmptyLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().Resize(1)
	rs0.InstrumentationLibraryLogs().At(0).Logs().Resize(2)
	fillLogOne(rs0.InstrumentationLibraryLogs().At(0).Logs().At(0))
	fillLogTwo(rs0.InstrumentationLibraryLogs().At(0).Logs().At(1))
	return ld
}

// GenerateLogOtlpSameResourceTwologs returns the OTLP representation of the GenerateLogOtlpSameResourceTwologs.
func GenerateLogOtlpSameResourceTwoLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						generateOtlpLogOne(),
						generateOtlpLogTwo(),
					},
				},
			},
		},
	}
}

func GenerateLogDataTwoLogsSameResourceOneDifferent() pdata.Logs {
	ld := pdata.NewLogs()
	ld.ResourceLogs().Resize(2)
	rl0 := ld.ResourceLogs().At(0)
	initResource1(rl0.Resource())
	rl0.InstrumentationLibraryLogs().Resize(1)
	rl0.InstrumentationLibraryLogs().At(0).Logs().Resize(2)
	fillLogOne(rl0.InstrumentationLibraryLogs().At(0).Logs().At(0))
	fillLogTwo(rl0.InstrumentationLibraryLogs().At(0).Logs().At(1))
	rl1 := ld.ResourceLogs().At(1)
	initResource2(rl1.Resource())
	rl1.InstrumentationLibraryLogs().Resize(1)
	rl1.InstrumentationLibraryLogs().At(0).Logs().Resize(1)
	fillLogThree(rl1.InstrumentationLibraryLogs().At(0).Logs().At(0))
	return ld
}

func generateLogOtlpTwoLogsSameResourceOneDifferent() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						generateOtlpLogOne(),
						generateOtlpLogTwo(),
					},
				},
			},
		},
		{
			Resource: generateOtlpResource2(),
			InstrumentationLibraryLogs: []*otlplogs.InstrumentationLibraryLogs{
				{
					Logs: []*otlplogs.LogRecord{
						generateOtlpLogThree(),
					},
				},
			},
		},
	}
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

func generateOtlpLogOne() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		Name:                   "logA",
		TimeUnixNano:           uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO,
		SeverityText:           "Info",
		Body:                   otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "This is a log message"}},
		SpanId:                 data.NewSpanID([8]byte{0x01, 0x02, 0x04, 0x08}),
		TraceId:                data.NewTraceID([16]byte{0x08, 0x04, 0x02, 0x01}),
		Attributes: []otlpcommon.KeyValue{
			{
				Key:   "app",
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "server"}},
			},
			{
				Key:   "instance_num",
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_IntValue{IntValue: 1}},
			},
		},
	}
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

func generateOtlpLogTwo() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		Name:                   "logB",
		TimeUnixNano:           uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_SEVERITY_NUMBER_INFO,
		SeverityText:           "Info",
		Body:                   otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "something happened"}},
		Attributes: []otlpcommon.KeyValue{
			{
				Key:   "customer",
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "acme"}},
			},
			{
				Key:   "env",
				Value: otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "dev"}},
			},
		},
	}
}

func fillLogThree(log pdata.LogRecord) {
	log.SetName("logC")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(pdata.SeverityNumberWARN)
	log.SetSeverityText("Warning")

	log.Body().SetStringVal("something else happened")
}

func generateOtlpLogThree() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		Name:                   "logC",
		TimeUnixNano:           uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_SEVERITY_NUMBER_WARN,
		SeverityText:           "Warning",
		Body:                   otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "something else happened"}},
	}
}

func GenerateLogDataManyLogsSameResource(count int) pdata.Logs {
	ld := GenerateLogDataOneEmptyLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.InstrumentationLibraryLogs().Resize(1)
	rs0.InstrumentationLibraryLogs().At(0).Logs().Resize(count)
	for i := 0; i < count; i++ {
		l := rs0.InstrumentationLibraryLogs().At(0).Logs().At(i)
		if i%2 == 0 {
			fillLogOne(l)
		} else {
			fillLogTwo(l)
		}
	}
	return ld
}
