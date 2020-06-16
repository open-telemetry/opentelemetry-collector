// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/internal/data"
	otlplogs "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
	opentelemetry_proto_common_v1 "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
)

var (
	TestLogTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestLogTimestamp = pdata.TimestampUnixNano(TestLogTime.UnixNano())
)

const (
	NumLogTests = 11
)

func GenerateLogDataEmpty() data.Logs {
	ld := data.NewLogs()
	return ld
}

func generateLogOtlpEmpty() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs(nil)
}

func GenerateLogDataOneEmptyResourceLogs() data.Logs {
	ld := GenerateLogDataEmpty()
	ld.ResourceLogs().Resize(1)
	return ld
}

func generateLogOtlpOneEmptyResourceLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{},
	}
}

func GenerateLogDataOneEmptyOneNilResourceLogs() data.Logs {
	return data.LogsFromProto(generateLogOtlpOneEmptyOneNilResourceLogs())

}

func generateLogOtlpOneEmptyOneNilResourceLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{},
		nil,
	}
}

func GenerateLogDataNoLogRecords() data.Logs {
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

func GenerateLogDataOneEmptyLogs() data.Logs {
	ld := GenerateLogDataNoLogRecords()
	rs0 := ld.ResourceLogs().At(0)
	rs0.Logs().Resize(1)
	return ld
}

func generateLogOtlpOneEmptyLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				{},
			},
		},
	}
}

func GenerateLogDataOneEmptyOneNilLogRecord() data.Logs {
	return data.LogsFromProto(generateLogOtlpOneEmptyOneNilLogRecord())
}

func generateLogOtlpOneEmptyOneNilLogRecord() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				{},
				nil,
			},
		},
	}
}

func GenerateLogDataOneLogNoResource() data.Logs {
	ld := GenerateLogDataOneEmptyResourceLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.Logs().Resize(1)
	rs0lr0 := rs0.Logs().At(0)
	fillLogOne(rs0lr0)
	return ld
}

func generateLogOtlpOneLogNoResource() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogOne(),
			},
		},
	}
}

func GenerateLogDataOneLog() data.Logs {
	ld := GenerateLogDataOneEmptyLogs()
	rs0lr0 := ld.ResourceLogs().At(0)
	rs0lr0.Logs().Resize(1)
	fillLogOne(rs0lr0.Logs().At(0))
	return ld
}

func generateLogOtlpOneLog() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogOne(),
			},
		},
	}
}

func GenerateLogDataOneLogOneNil() data.Logs {
	return data.LogsFromProto(generateLogOtlpOneLogOneNil())
}

func generateLogOtlpOneLogOneNil() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogOne(),
				nil,
			},
		},
	}
}

func GenerateLogDataTwoLogsSameResource() data.Logs {
	ld := GenerateLogDataOneEmptyLogs()
	rs0 := ld.ResourceLogs().At(0)
	rs0.Logs().Resize(2)
	fillLogOne(rs0.Logs().At(0))
	fillLogTwo(rs0.Logs().At(1))
	return ld
}

// GenerateLogOtlpSameResourceTwologs returns the OTLP representation of the GenerateLogOtlpSameResourceTwologs.
func GenerateLogOtlpSameResourceTwoLogs() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogOne(),
				generateOtlpLogTwo(),
			},
		},
	}
}

func GenerateLogDataTwoLogsSameResourceOneDifferent() data.Logs {
	ld := data.NewLogs()
	ld.ResourceLogs().Resize(2)
	rl0 := ld.ResourceLogs().At(0)
	initResource1(rl0.Resource())
	rl0.Logs().Resize(2)
	fillLogOne(rl0.Logs().At(0))
	fillLogTwo(rl0.Logs().At(1))
	rl1 := ld.ResourceLogs().At(1)
	initResource2(rl1.Resource())
	rl1.Logs().Resize(1)
	fillLogThree(rl1.Logs().At(0))
	return ld
}

func generateLogOtlpTwoLogsSameResourceOneDifferent() []*otlplogs.ResourceLogs {
	return []*otlplogs.ResourceLogs{
		{
			Resource: generateOtlpResource1(),
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogOne(),
				generateOtlpLogTwo(),
			},
		},
		{
			Resource: generateOtlpResource2(),
			Logs: []*otlplogs.LogRecord{
				generateOtlpLogThree(),
			},
		},
	}
}

func fillLogOne(log pdata.LogRecord) {
	log.SetShortName("logA")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(otlplogs.SeverityNumber_INFO)
	log.SetSeverityText("Info")
	log.SetSpanID([]byte{0x01, 0x02, 0x04, 0x08})
	log.SetTraceID([]byte{0x08, 0x04, 0x02, 0x01})

	attrs := log.Attributes()
	attrs.InsertString("app", "server")
	attrs.InsertInt("instance_num", 1)

	log.SetBody("This is a log message")
}

func generateOtlpLogOne() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		ShortName:              "logA",
		TimestampUnixNano:      uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_INFO,
		SeverityText:           "Info",
		Body:                   "This is a log message",
		SpanId:                 []byte{0x01, 0x02, 0x04, 0x08},
		TraceId:                []byte{0x08, 0x04, 0x02, 0x01},
		Attributes: []*opentelemetry_proto_common_v1.AttributeKeyValue{
			{
				Key:         "app",
				Type:        opentelemetry_proto_common_v1.AttributeKeyValue_STRING,
				StringValue: "server",
			},
			{
				Key:      "instance_num",
				Type:     opentelemetry_proto_common_v1.AttributeKeyValue_INT,
				IntValue: 1,
			},
		},
	}
}

func fillLogTwo(log pdata.LogRecord) {
	log.SetShortName("logB")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(otlplogs.SeverityNumber_INFO)
	log.SetSeverityText("Info")

	attrs := log.Attributes()
	attrs.InsertString("customer", "acme")
	attrs.InsertString("env", "dev")

	log.SetBody("something happened")
}

func generateOtlpLogTwo() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		ShortName:              "logB",
		TimestampUnixNano:      uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_INFO,
		SeverityText:           "Info",
		Body:                   "something happened",
		Attributes: []*opentelemetry_proto_common_v1.AttributeKeyValue{
			{
				Key:         "customer",
				StringValue: "acme",
			},
			{
				Key:         "env",
				StringValue: "dev",
			},
		},
	}
}

func fillLogThree(log pdata.LogRecord) {
	log.SetShortName("logC")
	log.SetTimestamp(TestLogTimestamp)
	log.SetDroppedAttributesCount(1)
	log.SetSeverityNumber(otlplogs.SeverityNumber_WARN)
	log.SetSeverityText("Warning")

	log.SetBody("something else happened")
}

func generateOtlpLogThree() *otlplogs.LogRecord {
	return &otlplogs.LogRecord{
		ShortName:              "logC",
		TimestampUnixNano:      uint64(TestLogTimestamp),
		DroppedAttributesCount: 1,
		SeverityNumber:         otlplogs.SeverityNumber_WARN,
		SeverityText:           "Warning",
		Body:                   "something else happened",
	}
}
