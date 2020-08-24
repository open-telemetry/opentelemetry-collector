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
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func BenchmarkConvertSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		convert(entry.New())
	}
}

func BenchmarkConvertComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		e := complexEntry()
		b.StartTimer()
		convert(e)
	}
}

func complexEntry() *entry.Entry {
	entry := entry.New()
	entry.Severity = entry.Error
	entry.AddResourceKey("type", "global")
	entry.AddLabel("one", "two")
	entry.AddLabel("two", "three")
	entry.Record = map[string]interface{}{
		"bool":   true,
		"int":    123,
		"double": 12.34,
		"string": "hello",
		"bytes":  []byte("asdf"),
		"object": map[string]interface{}{
			"bool":   true,
			"int":    123,
			"double": 12.34,
			"string": "hello",
			"bytes":  []byte("asdf"),
			"object": map[string]interface{}{
				"bool":   true,
				"int":    123,
				"double": 12.34,
				"string": "hello",
				"bytes":  []byte("asdf"),
			},
		},
	}
	return entry
}

func TestConvertMetadata(t *testing.T) {

	now := time.Now()

	e := entry.New()
	e.Timestamp = now
	e.Severity = entry.Error
	e.AddResourceKey("type", "global")
	e.AddLabel("one", "two")
	e.Record = true

	result := convert(e)

	resourceLogs := result.ResourceLogs()
	require.Equal(t, 1, resourceLogs.Len(), "expected 1 resource")

	libLogs := resourceLogs.At(0).InstrumentationLibraryLogs()
	require.Equal(t, 1, libLogs.Len(), "expected 1 library")

	logSlice := libLogs.At(0).Logs()
	require.Equal(t, 1, logSlice.Len(), "expected 1 log")

	log := logSlice.At(0)
	require.Equal(t, now.UnixNano(), int64(log.Timestamp()))

	require.Equal(t, pdata.SeverityNumberERROR, log.SeverityNumber())
	require.Equal(t, "Error", log.SeverityText())

	atts := log.Attributes()
	require.Equal(t, 1, atts.Len(), "expected 1 attribute")
	attVal, ok := atts.Get("one")
	require.True(t, ok, "expected label with key 'one'")
	require.Equal(t, "two", attVal.StringVal(), "expected label to have value 'two'")

	bod := log.Body()
	require.Equal(t, pdata.AttributeValueBOOL, int(bod.Type()))
	require.True(t, bod.BoolVal())
}

func TestConvertSimpleBody(t *testing.T) {

	require.True(t, recordToBody(true).BoolVal())
	require.False(t, recordToBody(false).BoolVal())

	require.Equal(t, "string", recordToBody("string").StringVal())
	require.Equal(t, "bytes", recordToBody([]byte("bytes")).StringVal())

	require.Equal(t, int64(1), recordToBody(1).IntVal())
	require.Equal(t, int64(1), recordToBody(int8(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int16(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int32(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(int64(1)).IntVal())

	require.Equal(t, int64(1), recordToBody(uint(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint8(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint16(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint32(1)).IntVal())
	require.Equal(t, int64(1), recordToBody(uint64(1)).IntVal())

	require.Equal(t, float64(1), recordToBody(float32(1)).DoubleVal())
	require.Equal(t, float64(1), recordToBody(float64(1)).DoubleVal())
}

func TestConvertMapBody(t *testing.T) {
	structuredRecord := map[string]interface{}{
		"true":    true,
		"false":   false,
		"string":  "string",
		"bytes":   []byte("bytes"),
		"int":     1,
		"int8":    int8(1),
		"int16":   int16(1),
		"int32":   int32(1),
		"int64":   int64(1),
		"uint":    uint(1),
		"uint8":   uint8(1),
		"uint16":  uint16(1),
		"uint32":  uint32(1),
		"uint64":  uint64(1),
		"float32": float32(1),
		"float64": float64(1),
	}

	result := recordToBody(structuredRecord).MapVal()

	v, _ := result.Get("true")
	require.True(t, v.BoolVal())
	v, _ = result.Get("false")
	require.False(t, v.BoolVal())

	for _, k := range []string{"string", "bytes"} {
		v, _ = result.Get(k)
		require.Equal(t, k, v.StringVal())
	}
	for _, k := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"} {
		v, _ = result.Get(k)
		require.Equal(t, int64(1), v.IntVal())
	}
	for _, k := range []string{"float32", "float64"} {
		v, _ = result.Get(k)
		require.Equal(t, float64(1), v.DoubleVal())
	}
}

func TestConvertArrayBody(t *testing.T) {
	result := recordToBody([]interface{}{0, 1}).MapVal()

	v, _ := result.Get("0")
	require.Equal(t, int64(0), v.IntVal())

	v, _ = result.Get("1")
	require.Equal(t, int64(1), v.IntVal())
}

func TestConvertUnknownBody(t *testing.T) {
	unknownType := map[string]int{"0": 0, "1": 1}
	require.Equal(t, fmt.Sprintf("%v", unknownType), recordToBody(unknownType).StringVal())
}

func TestConvertNestedMapBody(t *testing.T) {

	unknownType := map[string]int{"0": 0, "1": 1}

	structuredRecord := map[string]interface{}{
		"array":   []interface{}{0, 1},
		"map":     map[string]interface{}{"0": 0, "1": "one"},
		"unknown": unknownType,
	}

	result := recordToBody(structuredRecord).MapVal()

	arrayAttVal, _ := result.Get("array")
	a := arrayAttVal.MapVal()
	v, _ := a.Get("0")
	require.Equal(t, int64(0), v.IntVal())
	v, _ = a.Get("1")
	require.Equal(t, int64(1), v.IntVal())

	mapAttVal, _ := result.Get("map")
	m := mapAttVal.MapVal()
	v, _ = m.Get("0")
	require.Equal(t, int64(0), v.IntVal())
	v, _ = m.Get("1")
	require.Equal(t, "one", v.StringVal())

	unknownAttVal, _ := result.Get("unknown")
	require.Equal(t, fmt.Sprintf("%v", unknownType), unknownAttVal.StringVal())
}

func recordToBody(record interface{}) pdata.AttributeValue {
	entry := entry.New()
	entry.Record = record
	return convertAndDrill(entry).Body()
}

func convertAndDrill(entry *entry.Entry) pdata.LogRecord {
	return convert(entry).ResourceLogs().At(0).InstrumentationLibraryLogs().At(0).Logs().At(0)
}

func TestConvertSeverity(t *testing.T) {
	cases := []struct {
		severity       entry.Severity
		expectedNumber pdata.SeverityNumber
		expectedText   string
	}{
		{entry.Default, pdata.SeverityNumberUNDEFINED, "Undefined"},
		{5, pdata.SeverityNumberTRACE, "Trace"},
		{entry.Trace, pdata.SeverityNumberTRACE2, "Trace"},
		{15, pdata.SeverityNumberTRACE3, "Trace"},
		{entry.Debug, pdata.SeverityNumberDEBUG, "Debug"},
		{25, pdata.SeverityNumberDEBUG2, "Debug"},
		{entry.Info, pdata.SeverityNumberINFO, "Info"},
		{35, pdata.SeverityNumberINFO2, "Info"},
		{entry.Notice, pdata.SeverityNumberINFO3, "Info"},
		{45, pdata.SeverityNumberINFO3, "Info"},
		{entry.Warning, pdata.SeverityNumberINFO4, "Info"},
		{55, pdata.SeverityNumberINFO4, "Info"},
		{entry.Error, pdata.SeverityNumberERROR, "Error"},
		{65, pdata.SeverityNumberERROR2, "Error"},
		{entry.Critical, pdata.SeverityNumberERROR2, "Error"},
		{75, pdata.SeverityNumberERROR3, "Error"},
		{entry.Alert, pdata.SeverityNumberERROR3, "Error"},
		{85, pdata.SeverityNumberERROR4, "Error"},
		{entry.Emergency, pdata.SeverityNumberFATAL, "Error"},
		{95, pdata.SeverityNumberFATAL2, "Fatal"},
		{entry.Catastrophe, pdata.SeverityNumberFATAL4, "Fatal"},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.severity), func(t *testing.T) {
			entry := entry.New()
			entry.Severity = tc.severity
			log := convertAndDrill(entry)
			require.Equal(t, tc.expectedNumber, log.SeverityNumber())
			require.Equal(t, tc.expectedText, log.SeverityText())
		})
	}
}
