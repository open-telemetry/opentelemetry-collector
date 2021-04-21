// Copyright The OpenTelemetry Authors
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

package stanza

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func BenchmarkConvertSimple(b *testing.B) {
	b.StopTimer()
	ent := entry.New()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		convert(ent)
	}
}

func BenchmarkConvertComplex(b *testing.B) {
	b.StopTimer()
	ent := complexEntry()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		convert(ent)
	}
}

func complexEntries(count int) []*entry.Entry {
	return complexEntriesForNDifferentHosts(count, 1)
}

func complexEntriesForNDifferentHosts(count int, n int) []*entry.Entry {
	ret := make([]*entry.Entry, count)
	for i := 0; i < count; i++ {
		e := entry.New()
		e.Severity = entry.Error
		e.AddResourceKey("type", "global")
		e.Resource = map[string]string{
			"host": fmt.Sprintf("host-%d", i%n),
		}
		e.Body = map[string]interface{}{
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
		ret[i] = e
	}
	return ret
}

func complexEntry() *entry.Entry {
	e := entry.New()
	e.Severity = entry.Error
	e.AddResourceKey("type", "global")
	e.AddAttribute("one", "two")
	e.AddAttribute("two", "three")
	e.Body = map[string]interface{}{
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
	return e
}

func TestAllConvertedEntriesAreSentAndReceived(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		entries       int
		maxFlushCount uint
	}{
		{
			entries:       10,
			maxFlushCount: 10,
		},
		{
			entries:       10,
			maxFlushCount: 3,
		},
		{
			entries:       100,
			maxFlushCount: 20,
		},
	}

	for i, tc := range testcases {
		tc := tc

		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			converter := NewConverter(
				WithWorkerCount(1),
				WithMaxFlushCount(tc.maxFlushCount),
				WithFlushInterval(10*time.Millisecond), // To minimize time spent in test
			)
			converter.Start()
			defer converter.Stop()

			go func() {
				for _, ent := range complexEntries(tc.entries) {
					assert.NoError(t, converter.Batch(ent))
				}
			}()

			var (
				actualCount  int
				timeoutTimer = time.NewTimer(10 * time.Second)
				ch           = converter.OutChannel()
			)
			defer timeoutTimer.Stop()

		forLoop:
			for {
				if tc.entries == actualCount {
					break
				}

				select {
				case pLogs, ok := <-ch:
					if !ok {
						break forLoop
					}

					rLogs := pLogs.ResourceLogs()
					require.Equal(t, 1, rLogs.Len())

					rLog := rLogs.At(0)
					ills := rLog.InstrumentationLibraryLogs()
					require.Equal(t, 1, ills.Len())

					ill := ills.At(0)

					actualCount += ill.Logs().Len()

					assert.LessOrEqual(t, uint(ill.Logs().Len()), tc.maxFlushCount,
						"Received more log records in one flush than configured by maxFlushCount",
					)

				case <-timeoutTimer.C:
					break forLoop
				}
			}

			assert.Equal(t, tc.entries, actualCount,
				"didn't receive expected number of entries after conversion",
			)
		})
	}
}

func TestAllConvertedEntriesAreSentAndReceivedWithinAnExpectedTimeDuration(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		entries       int
		hostsCount    int
		maxFlushCount uint
		flushInterval time.Duration
	}{
		{
			entries:       10,
			hostsCount:    1,
			maxFlushCount: 20,
			flushInterval: 100 * time.Millisecond,
		},
		{
			entries:       50,
			hostsCount:    1,
			maxFlushCount: 51,
			flushInterval: 100 * time.Millisecond,
		},
		{
			entries:       500,
			hostsCount:    1,
			maxFlushCount: 501,
			flushInterval: 100 * time.Millisecond,
		},
		{
			entries:       500,
			hostsCount:    1,
			maxFlushCount: 100,
			flushInterval: 100 * time.Millisecond,
		},
		{
			entries:       500,
			hostsCount:    4,
			maxFlushCount: 501,
			flushInterval: 100 * time.Millisecond,
		},
	}

	for i, tc := range testcases {
		tc := tc

		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			converter := NewConverter(
				WithWorkerCount(1),
				WithMaxFlushCount(tc.maxFlushCount),
				WithFlushInterval(tc.flushInterval),
			)
			converter.Start()
			defer converter.Stop()

			go func() {
				for _, ent := range complexEntriesForNDifferentHosts(tc.entries, tc.hostsCount) {
					assert.NoError(t, converter.Batch(ent))
				}
			}()

			var (
				actualCount      int
				actualFlushCount int
				timeoutTimer     = time.NewTimer(10 * time.Second)
				ch               = converter.OutChannel()
			)
			defer timeoutTimer.Stop()

		forLoop:
			for start := time.Now(); ; start = time.Now() {
				if tc.entries == actualCount {
					break
				}

				select {
				case pLogs, ok := <-ch:
					if !ok {
						break forLoop
					}

					assert.WithinDuration(t,
						start.Add(tc.flushInterval),
						time.Now(),
						tc.flushInterval,
					)

					actualFlushCount++

					rLogs := pLogs.ResourceLogs()
					require.Equal(t, 1, rLogs.Len())

					rLog := rLogs.At(0)
					ills := rLog.InstrumentationLibraryLogs()
					require.Equal(t, 1, ills.Len())

					ill := ills.At(0)

					actualCount += ill.Logs().Len()

					assert.LessOrEqual(t, uint(ill.Logs().Len()), tc.maxFlushCount,
						"Received more log records in one flush than configured by maxFlushCount",
					)

				case <-timeoutTimer.C:
					break forLoop
				}
			}

			assert.Equal(t, tc.entries, actualCount,
				"didn't receive expected number of entries after conversion",
			)
		})
	}
}

func TestConverterCancelledContextCancellsTheFlush(t *testing.T) {
	converter := NewConverter(
		WithMaxFlushCount(1),
		WithFlushInterval(time.Millisecond),
	)
	converter.Start()
	defer converter.Stop()
	var wg sync.WaitGroup
	wg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	go func() {
		defer wg.Done()
		pLogs := pdata.NewLogs()
		logs := pLogs.ResourceLogs()
		logs.Resize(1)
		rls := logs.At(0)
		rls.InstrumentationLibraryLogs().Resize(1)
		ills := rls.InstrumentationLibraryLogs().At(0)

		lr := convert(complexEntry())
		ills.Logs().Append(lr)

		assert.Error(t, converter.flush(ctx, pLogs))
	}()
	wg.Wait()
}

func TestConvertMetadata(t *testing.T) {
	now := time.Now()

	e := entry.New()
	e.Timestamp = now
	e.Severity = entry.Error
	e.AddResourceKey("type", "global")
	e.AddAttribute("one", "two")
	e.Body = true

	result := convert(e)

	atts := result.Attributes()
	require.Equal(t, 1, atts.Len(), "expected 1 attribute")
	attVal, ok := atts.Get("one")
	require.True(t, ok, "expected label with key 'one'")
	require.Equal(t, "two", attVal.StringVal(), "expected label to have value 'two'")

	bod := result.Body()
	require.Equal(t, pdata.AttributeValueBOOL, bod.Type())
	require.True(t, bod.BoolVal())
}

func TestConvertSimpleBody(t *testing.T) {

	require.True(t, anyToBody(true).BoolVal())
	require.False(t, anyToBody(false).BoolVal())

	require.Equal(t, "string", anyToBody("string").StringVal())
	require.Equal(t, "bytes", anyToBody([]byte("bytes")).StringVal())

	require.Equal(t, int64(1), anyToBody(1).IntVal())
	require.Equal(t, int64(1), anyToBody(int8(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(int16(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(int32(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(int64(1)).IntVal())

	require.Equal(t, int64(1), anyToBody(uint(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(uint8(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(uint16(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(uint32(1)).IntVal())
	require.Equal(t, int64(1), anyToBody(uint64(1)).IntVal())

	require.Equal(t, float64(1), anyToBody(float32(1)).DoubleVal())
	require.Equal(t, float64(1), anyToBody(float64(1)).DoubleVal())
}

func TestConvertMapBody(t *testing.T) {
	structuredBody := map[string]interface{}{
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

	result := anyToBody(structuredBody).MapVal()

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
	structuredBody := []interface{}{
		true,
		false,
		"string",
		[]byte("bytes"),
		1,
		int8(1),
		int16(1),
		int32(1),
		int64(1),
		uint(1),
		uint8(1),
		uint16(1),
		uint32(1),
		uint64(1),
		float32(1),
		float64(1),
		[]interface{}{"string", 1},
		map[string]interface{}{"one": 1, "yes": true},
	}

	result := anyToBody(structuredBody).ArrayVal()

	require.True(t, result.At(0).BoolVal())
	require.False(t, result.At(1).BoolVal())
	require.Equal(t, "string", result.At(2).StringVal())
	require.Equal(t, "bytes", result.At(3).StringVal())

	require.Equal(t, int64(1), result.At(4).IntVal())  // int
	require.Equal(t, int64(1), result.At(5).IntVal())  // int8
	require.Equal(t, int64(1), result.At(6).IntVal())  // int16
	require.Equal(t, int64(1), result.At(7).IntVal())  // int32
	require.Equal(t, int64(1), result.At(8).IntVal())  // int64
	require.Equal(t, int64(1), result.At(9).IntVal())  // uint
	require.Equal(t, int64(1), result.At(10).IntVal()) // uint8
	require.Equal(t, int64(1), result.At(11).IntVal()) // uint16
	require.Equal(t, int64(1), result.At(12).IntVal()) // uint32
	require.Equal(t, int64(1), result.At(13).IntVal()) // uint64

	require.Equal(t, float64(1), result.At(14).DoubleVal()) // float32
	require.Equal(t, float64(1), result.At(15).DoubleVal()) // float64

	nestedArr := result.At(16).ArrayVal()
	require.Equal(t, "string", nestedArr.At(0).StringVal())
	require.Equal(t, int64(1), nestedArr.At(1).IntVal())

	nestedMap := result.At(17).MapVal()
	v, _ := nestedMap.Get("one")
	require.Equal(t, int64(1), v.IntVal())
	v, _ = nestedMap.Get("yes")
	require.True(t, v.BoolVal())
}

func TestConvertUnknownBody(t *testing.T) {
	unknownType := map[string]int{"0": 0, "1": 1}
	require.Equal(t, fmt.Sprintf("%v", unknownType), anyToBody(unknownType).StringVal())
}

func TestConvertNestedMapBody(t *testing.T) {

	unknownType := map[string]int{"0": 0, "1": 1}

	structuredBody := map[string]interface{}{
		"array":   []interface{}{0, 1},
		"map":     map[string]interface{}{"0": 0, "1": "one"},
		"unknown": unknownType,
	}

	result := anyToBody(structuredBody).MapVal()

	arrayAttVal, _ := result.Get("array")
	a := arrayAttVal.ArrayVal()
	require.Equal(t, int64(0), a.At(0).IntVal())
	require.Equal(t, int64(1), a.At(1).IntVal())

	mapAttVal, _ := result.Get("map")
	m := mapAttVal.MapVal()
	v, _ := m.Get("0")
	require.Equal(t, int64(0), v.IntVal())
	v, _ = m.Get("1")
	require.Equal(t, "one", v.StringVal())

	unknownAttVal, _ := result.Get("unknown")
	require.Equal(t, fmt.Sprintf("%v", unknownType), unknownAttVal.StringVal())
}

func anyToBody(body interface{}) pdata.AttributeValue {
	entry := entry.New()
	entry.Body = body
	return convertAndDrill(entry).Body()
}

func convertAndDrill(entry *entry.Entry) pdata.LogRecord {
	return convert(entry)
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

func TestConvertTrace(t *testing.T) {
	record := convertAndDrill(&entry.Entry{
		TraceId: []byte{
			0x48, 0x01, 0x40, 0xf3, 0xd7, 0x70, 0xa5, 0xae, 0x32, 0xf0, 0xa2, 0x2b, 0x6a, 0x81, 0x2c, 0xff,
		},
		SpanId: []byte{
			0x32, 0xf0, 0xa2, 0x2b, 0x6a, 0x81, 0x2c, 0xff,
		},
		TraceFlags: []byte{
			0x01,
		}})

	require.Equal(t, pdata.NewTraceID(
		[16]byte{
			0x48, 0x01, 0x40, 0xf3, 0xd7, 0x70, 0xa5, 0xae, 0x32, 0xf0, 0xa2, 0x2b, 0x6a, 0x81, 0x2c, 0xff,
		}), record.TraceID())
	require.Equal(t, pdata.NewSpanID(
		[8]byte{
			0x32, 0xf0, 0xa2, 0x2b, 0x6a, 0x81, 0x2c, 0xff,
		}), record.SpanID())
	require.Equal(t, uint32(0x01), record.Flags())
}

func BenchmarkConverter(b *testing.B) {
	const (
		entryCount = 1_000_000
		hostsCount = 4
	)

	var (
		workerCounts = []int{1, 2, 4, 6, 8}
		entries      = complexEntriesForNDifferentHosts(entryCount, hostsCount)
	)

	for _, wc := range workerCounts {
		b.Run(fmt.Sprintf("worker_count=%d", wc), func(b *testing.B) {
			for i := 0; i < b.N; i++ {

				converter := NewConverter(
					WithWorkerCount(wc),
					WithMaxFlushCount(1_000),
					WithFlushInterval(250*time.Millisecond),
				)
				converter.Start()
				defer converter.Stop()
				b.ResetTimer()

				go func() {
					for _, ent := range entries {
						assert.NoError(b, converter.Batch(ent))
					}
				}()

				var (
					timeoutTimer = time.NewTimer(10 * time.Second)
					ch           = converter.OutChannel()
				)
				defer timeoutTimer.Stop()

				var n int
			forLoop:
				for {
					if n == entryCount {
						break
					}

					select {
					case pLogs, ok := <-ch:
						if !ok {
							break forLoop
						}

						rLogs := pLogs.ResourceLogs()
						require.Equal(b, 1, rLogs.Len())

						rLog := rLogs.At(0)
						ills := rLog.InstrumentationLibraryLogs()
						require.Equal(b, 1, ills.Len())

						ill := ills.At(0)

						n += ill.Logs().Len()

					case <-timeoutTimer.C:
						break forLoop
					}
				}

				assert.Equal(b, entryCount, n,
					"didn't receive expected number of entries after conversion",
				)
			}
		})
	}
}
