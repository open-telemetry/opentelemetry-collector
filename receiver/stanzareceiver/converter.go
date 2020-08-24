// Copyright 2019, OpenTelemetry Authors
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

package stanzareceiver

import (
	"fmt"
	"strconv"

	"github.com/observiq/stanza/entry"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func convert(obsLog *entry.Entry) pdata.Logs {
	out := pdata.NewLogs()
	logs := out.ResourceLogs()
	logs.Resize(1)
	rls := logs.At(0)

	resource := rls.Resource()
	resource.InitEmpty()
	if len(obsLog.Resource) > 0 {
		resourceAtts := resource.Attributes()
		for k, v := range obsLog.Resource {
			resourceAtts.InsertString(k, v)
		}
	}

	rls.InstrumentationLibraryLogs().Resize(1)
	logSlice := rls.InstrumentationLibraryLogs().At(0).Logs()

	lr := pdata.NewLogRecord()
	lr.InitEmpty()

	lr.SetTimestamp(pdata.TimestampUnixNano(obsLog.Timestamp.UnixNano()))

	sevText, sevNum := convertSeverity(obsLog.Severity)
	lr.SetSeverityText(sevText)
	lr.SetSeverityNumber(sevNum)

	if len(obsLog.Labels) > 0 {
		attributes := lr.Attributes()
		for k, v := range obsLog.Labels {
			attributes.InsertString(k, v)
		}
	}

	lr.Body().InitEmpty()
	insertToAttributeVal(obsLog.Record, lr.Body())

	logSlice.Append(&lr)

	return out
}

func insertToAttributeVal(value interface{}, dest pdata.AttributeValue) {
	switch t := value.(type) {
	case bool:
		dest.SetBoolVal(t)
	case string:
		dest.SetStringVal(t)
	case []byte:
		dest.SetStringVal(string(t))
	case int64:
		dest.SetIntVal(t)
	case int32:
		dest.SetIntVal(int64(t))
	case int16:
		dest.SetIntVal(int64(t))
	case int8:
		dest.SetIntVal(int64(t))
	case int:
		dest.SetIntVal(int64(t))
	case uint64:
		dest.SetIntVal(int64(t))
	case uint32:
		dest.SetIntVal(int64(t))
	case uint16:
		dest.SetIntVal(int64(t))
	case uint8:
		dest.SetIntVal(int64(t))
	case uint:
		dest.SetIntVal(int64(t))
	case float64:
		dest.SetDoubleVal(t)
	case float32:
		dest.SetDoubleVal(float64(t))
	case map[string]interface{}:
		dest.SetMapVal(toAttributeMap(t))
	case []interface{}:
		dest.SetMapVal(toAttributeMap(sliceToMap(t)))
	default:
		dest.SetStringVal(fmt.Sprintf("%v", t))
	}
}

func toAttributeMap(obsMap map[string]interface{}) pdata.AttributeMap {
	attMap := pdata.NewAttributeMap()
	attMap.InitEmptyWithCapacity(len(obsMap))
	for k, v := range obsMap {
		switch t := v.(type) {
		case bool:
			attMap.InsertBool(k, t)
		case string:
			attMap.InsertString(k, t)
		case []byte:
			attMap.InsertString(k, string(t))
		case int64:
			attMap.InsertInt(k, t)
		case int32:
			attMap.InsertInt(k, int64(t))
		case int16:
			attMap.InsertInt(k, int64(t))
		case int8:
			attMap.InsertInt(k, int64(t))
		case int:
			attMap.InsertInt(k, int64(t))
		case uint64:
			attMap.InsertInt(k, int64(t))
		case uint32:
			attMap.InsertInt(k, int64(t))
		case uint16:
			attMap.InsertInt(k, int64(t))
		case uint8:
			attMap.InsertInt(k, int64(t))
		case uint:
			attMap.InsertInt(k, int64(t))
		case float64:
			attMap.InsertDouble(k, t)
		case float32:
			attMap.InsertDouble(k, float64(t))
		case map[string]interface{}:
			subMap := toAttributeMap(t)
			subMapVal := pdata.NewAttributeValueMap()
			subMapVal.SetMapVal(subMap)
			attMap.Insert(k, subMapVal)
		case []interface{}:
			subMap := toAttributeMap(sliceToMap(t))
			subMapVal := pdata.NewAttributeValueMap()
			subMapVal.SetMapVal(subMap)
			attMap.Insert(k, subMapVal)
		default:
			attMap.InsertString(k, fmt.Sprintf("%v", t))
		}
	}
	return attMap
}

// This returns a map of stringified index to value,
// rather than an array because the pdata package does not support arrays
func sliceToMap(arr []interface{}) map[string]interface{} {
	sliceAsMap := make(map[string]interface{})
	for i, v := range arr {
		sliceAsMap[strconv.Itoa(i)] = v
	}
	return sliceAsMap
}

func convertSeverity(s entry.Severity) (string, pdata.SeverityNumber) {
	switch {

	// Handle standard severity levels
	case s == entry.Catastrophe:
		return "Fatal", pdata.SeverityNumberFATAL4
	case s == entry.Emergency:
		return "Error", pdata.SeverityNumberFATAL
	case s == entry.Alert:
		return "Error", pdata.SeverityNumberERROR3
	case s == entry.Critical:
		return "Error", pdata.SeverityNumberERROR2
	case s == entry.Error:
		return "Error", pdata.SeverityNumberERROR
	case s == entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s == entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s == entry.Info:
		return "Info", pdata.SeverityNumberINFO
	case s == entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG
	case s == entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE2

	// Handle custom severity levels
	case s > entry.Emergency:
		return "Fatal", pdata.SeverityNumberFATAL2
	case s > entry.Alert:
		return "Error", pdata.SeverityNumberERROR4
	case s > entry.Critical:
		return "Error", pdata.SeverityNumberERROR3
	case s > entry.Error:
		return "Error", pdata.SeverityNumberERROR2
	case s > entry.Warning:
		return "Info", pdata.SeverityNumberINFO4
	case s > entry.Notice:
		return "Info", pdata.SeverityNumberINFO3
	case s > entry.Info:
		return "Info", pdata.SeverityNumberINFO2
	case s > entry.Debug:
		return "Debug", pdata.SeverityNumberDEBUG2
	case s > entry.Trace:
		return "Trace", pdata.SeverityNumberTRACE3
	case s > entry.Default:
		return "Trace", pdata.SeverityNumberTRACE

	default:
		return "Undefined", pdata.SeverityNumberUNDEFINED
	}
}
