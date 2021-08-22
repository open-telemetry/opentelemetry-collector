// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package skywalkingexporter

import (
	v3 "skywalking.apache.org/repo/goapi/collect/logging/v3"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/translator/conventions/v1.5.0"
	logpb "skywalking.apache.org/repo/goapi/collect/logging/v3"
)

func getComplexAttributeValueMap() pdata.AttributeValue {
	mapVal := pdata.NewAttributeValueMap()
	mapValReal := mapVal.MapVal()
	mapValReal.InsertBool("result", true)
	mapValReal.InsertString("status", "ok")
	mapValReal.InsertDouble("value", 1.3)
	mapValReal.InsertInt("code", 200)
	mapValReal.InsertNull("null")
	arrayVal := pdata.NewAttributeValueArray()
	arrayVal.ArrayVal().AppendEmpty().SetStringVal("array")
	mapValReal.Insert("array", arrayVal)

	subMapVal := pdata.NewAttributeValueMap()
	subMapVal.MapVal().InsertString("data", "hello world")
	mapValReal.Insert("map", subMapVal)

	mapValReal.InsertString("status", "ok")
	return mapVal
}

func createLogData(numberOfLogs int) pdata.Logs {
	logs := pdata.NewLogs()
	logs.ResourceLogs().AppendEmpty() // Add an empty ResourceLogs
	rl := logs.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().InsertString("resourceKey", "resourceValue")
	rl.Resource().Attributes().InsertString(conventions.AttributeServiceName, "test-service")
	rl.Resource().Attributes().InsertString(conventions.AttributeHostName, "test-host")
	rl.Resource().Attributes().InsertString(conventions.AttributeServiceInstanceID, "test-instance")
	ill := rl.InstrumentationLibraryLogs().AppendEmpty()
	ill.InstrumentationLibrary().SetName("collector")
	ill.InstrumentationLibrary().SetVersion("v0.1.0")

	for i := 0; i < numberOfLogs; i++ {
		ts := pdata.Timestamp(int64(i) * time.Millisecond.Nanoseconds())
		logRecord := ill.Logs().AppendEmpty()
		logRecord.SetTraceID(pdata.NewTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}))
		logRecord.SetSpanID(pdata.NewSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
		logRecord.SetFlags(uint32(0x01))
		logRecord.SetSeverityText("INFO")
		logRecord.SetSeverityNumber(pdata.SeverityNumberINFO)
		logRecord.SetName("test_name")
		logRecord.SetTimestamp(ts)
		switch i {
		case 0:
			// do nothing, left body null
		case 1:
			logRecord.Body().SetBoolVal(true)
		case 2:
			logRecord.Body().SetIntVal(2.0)
		case 3:
			logRecord.Body().SetDoubleVal(3.0)
		case 4:
			logRecord.Body().SetStringVal("4")
		case 5:

			logRecord.Attributes().Insert("map-value", getComplexAttributeValueMap())
			logRecord.Body().SetStringVal("log contents")
		case 6:
			arrayVal := pdata.NewAttributeValueArray()
			arrayVal.ArrayVal().AppendEmpty().SetStringVal("array")
			logRecord.Attributes().Insert("array-value", arrayVal)
			logRecord.Body().SetStringVal("log contents")
		default:
			logRecord.Body().SetStringVal("log contents")
		}
		logRecord.Attributes().InsertString("custom", "custom")
	}

	return logs
}

func TestLogsDataToLogService(t *testing.T) {
	gotLogs := logDataToLogRecode(createLogData(10))
	assert.Equal(t, len(gotLogs), 10)
	for i := 0; i < 10; i++ {
		log := gotLogs[i]

		if i != 0 {
			assert.Equal(t, log.TraceContext.TraceId, "01020304050607080807060504030201")
			assert.Equal(t, searchLogTag(spanIDField, log), "0102030405060708")
			assert.Equal(t, searchLogTag(flags, log), "1")
			assert.Equal(t, searchLogTag(severityText, log), "INFO")
			assert.Equal(t, searchLogTag(severityNumber, log), "9")
			assert.Equal(t, searchLogTag(name, log), "test_name")
			assert.Equal(t, log.Timestamp, pdata.Timestamp(int64(i)*time.Millisecond.Nanoseconds()).AsTime().UnixMilli())
			if i == 1 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "true")
			} else if i == 2 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "2")
			} else if i == 3 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "3")
			} else if i == 4 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "4")
			} else if i == 5 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "log contents")
				assert.Equal(t, searchLogTag("map-value", log), "{\n     -> array: ARRAY([\"array\"])\n     -> code: INT(200)\n     -> map: MAP({\"data\":\"hello world\"})\n     -> null: NULL()\n     -> result: BOOL(true)\n     -> status: STRING(ok)\n     -> value: DOUBLE(1.3)\n}")
			} else if i == 6 {
				assert.Equal(t, log.GetBody().GetText().GetText(), "log contents")
				assert.Equal(t, searchLogTag("array-value", log), "[array]")
			} else {
				assert.Equal(t, log.GetBody().GetText().GetText(), "log contents")
			}
		} else {
			assert.Equal(t, log.TraceContext, (*v3.TraceContext)(nil))
			assert.Equal(t, log.Body, (*v3.LogDataBody)(nil))
		}
		assert.Equal(t, log.Service, "test-service")
		assert.Equal(t, log.ServiceInstance, "test-instance")
		assert.Equal(t, searchLogTag("resourceKey", log), "resourceValue")
		assert.Equal(t, searchLogTag(conventions.AttributeHostName, log), "test-host")
		assert.Equal(t, searchLogTag(instrumentationName, log), "collector")
		assert.Equal(t, searchLogTag(instrumentationVersion, log), "v0.1.0")

		if i != 0 {
			assert.Equal(t, searchLogTag("custom", log), "custom")
		}
	}
}

func searchLogTag(key string, record *logpb.LogData) string {
	for _, tag := range record.GetTags().GetData() {
		if tag.Key == key {
			return tag.GetValue()
		}
	}
	return ""
}
