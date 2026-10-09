// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	utf8InvalidStr   = "bad\xff\xfestring"
	utf8SanitizedStr = "bad�string"
)

func utf8StrValue(s string) AnyValue {
	return AnyValue{Value: &AnyValue_StringValue{StringValue: s}}
}

func utf8KeyValues(s string) []KeyValue {
	return []KeyValue{
		{Key: s, Value: utf8StrValue(s)},
		{Key: "int", Value: AnyValue{Value: &AnyValue_IntValue{IntValue: 1}}},
		{Key: "bytes", Value: AnyValue{Value: &AnyValue_BytesValue{BytesValue: []byte(utf8InvalidStr)}}},
		{Key: "array", Value: AnyValue{Value: &AnyValue_ArrayValue{ArrayValue: &ArrayValue{Values: []AnyValue{utf8StrValue(s), {}}}}}},
		{Key: "kvlist", Value: AnyValue{Value: &AnyValue_KvlistValue{KvlistValue: &KeyValueList{Values: []KeyValue{{Key: s, Value: utf8StrValue(s)}}}}}},
		{Key: "empty"},
	}
}

func utf8Resource(s string) Resource {
	return Resource{
		Attributes: utf8KeyValues(s),
		EntityRefs: []*EntityRef{{SchemaUrl: s, Type: s, IdKeys: []string{s, "ok"}, DescriptionKeys: []string{s}}},
	}
}

func utf8Scope(s string) InstrumentationScope {
	return InstrumentationScope{Name: s, Version: s, Attributes: utf8KeyValues(s)}
}

func TestSanitizeUTF8String(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "ascii", in: "hello", want: "hello"},
		{name: "multibyte", in: "héllo 世界 🌍", want: "héllo 世界 🌍"},
		{name: "single invalid byte", in: "a\xffb", want: "a�b"},
		{name: "run of invalid bytes", in: "a\xff\xfe\xfdb", want: "a�b"},
		{name: "truncated multibyte", in: "a\xe4\xb8", want: "a�"},
		{name: "only invalid", in: "\xff", want: "�"},
		{name: "surrogate half", in: "a\xed\xa0\x80b", want: "a�b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SanitizeUTF8String(tt.in))
		})
	}
}

func TestSanitizeUTF8StringNoAllocWhenValid(t *testing.T) {
	in := strings.Repeat("valid", 10)
	allocs := testing.AllocsPerRun(100, func() {
		if SanitizeUTF8String(in) != in {
			t.Fatal("valid string must be returned unchanged")
		}
	})
	assert.Zero(t, allocs)
}

func TestSanitizeUTF8AnyValue(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		v := AnyValue{}
		SanitizeUTF8AnyValue(&v)
		assert.Equal(t, AnyValue{}, v)
	})

	t.Run("nil array", func(t *testing.T) {
		v := AnyValue{Value: &AnyValue_ArrayValue{}}
		SanitizeUTF8AnyValue(&v)
		assert.Equal(t, AnyValue{Value: &AnyValue_ArrayValue{}}, v)
	})

	t.Run("nil kvlist", func(t *testing.T) {
		v := AnyValue{Value: &AnyValue_KvlistValue{}}
		SanitizeUTF8AnyValue(&v)
		assert.Equal(t, AnyValue{Value: &AnyValue_KvlistValue{}}, v)
	})

	t.Run("bytes untouched", func(t *testing.T) {
		v := AnyValue{Value: &AnyValue_BytesValue{BytesValue: []byte(utf8InvalidStr)}}
		SanitizeUTF8AnyValue(&v)
		assert.Equal(t, []byte(utf8InvalidStr), v.Value.(*AnyValue_BytesValue).BytesValue)
	})

	t.Run("string table index untouched", func(t *testing.T) {
		v := AnyValue{Value: &AnyValue_StringValueStrindex{StringValueStrindex: 3}}
		SanitizeUTF8AnyValue(&v)
		assert.Equal(t, AnyValue{Value: &AnyValue_StringValueStrindex{StringValueStrindex: 3}}, v)
	})

	t.Run("nested", func(t *testing.T) {
		kvs := utf8KeyValues(utf8InvalidStr)
		SanitizeUTF8KeyValueSlice(kvs)
		assert.Equal(t, utf8KeyValues(utf8SanitizedStr), kvs)
	})
}

func TestSanitizeUTF8Resource(t *testing.T) {
	r := utf8Resource(utf8InvalidStr)
	SanitizeUTF8Resource(&r)
	assert.Equal(t, utf8Resource(utf8SanitizedStr), r)
}

func TestSanitizeUTF8InstrumentationScope(t *testing.T) {
	s := utf8Scope(utf8InvalidStr)
	SanitizeUTF8InstrumentationScope(&s)
	assert.Equal(t, utf8Scope(utf8SanitizedStr), s)
}

func utf8LogsRequest(s string) *ExportLogsServiceRequest {
	return &ExportLogsServiceRequest{
		ResourceLogs: []*ResourceLogs{{
			Resource:  utf8Resource(s),
			SchemaUrl: s,
			ScopeLogs: []*ScopeLogs{{
				Scope:     utf8Scope(s),
				SchemaUrl: s,
				LogRecords: []*LogRecord{
					{SeverityText: s, EventName: s, Body: utf8StrValue(s), Attributes: utf8KeyValues(s)},
					{Body: AnyValue{Value: &AnyValue_KvlistValue{KvlistValue: &KeyValueList{Values: utf8KeyValues(s)}}}},
					{Body: AnyValue{Value: &AnyValue_BytesValue{BytesValue: []byte(utf8InvalidStr)}}},
				},
			}},
			DeprecatedScopeLogs: []*ScopeLogs{{
				Scope:      utf8Scope(s),
				LogRecords: []*LogRecord{{SeverityText: s}},
			}},
		}},
	}
}

func TestSanitizeUTF8ExportLogsServiceRequest(t *testing.T) {
	req := utf8LogsRequest(utf8InvalidStr)
	SanitizeUTF8ExportLogsServiceRequest(req)
	assert.Equal(t, utf8LogsRequest(utf8SanitizedStr), req)

	t.Run("valid request unchanged", func(t *testing.T) {
		req := utf8LogsRequest("valid")
		SanitizeUTF8ExportLogsServiceRequest(req)
		assert.Equal(t, utf8LogsRequest("valid"), req)
	})

	t.Run("empty request", func(t *testing.T) {
		req := &ExportLogsServiceRequest{}
		SanitizeUTF8ExportLogsServiceRequest(req)
		assert.Equal(t, &ExportLogsServiceRequest{}, req)
	})
}

func utf8TracesRequest(s string) *ExportTraceServiceRequest {
	return &ExportTraceServiceRequest{
		ResourceSpans: []*ResourceSpans{{
			Resource:  utf8Resource(s),
			SchemaUrl: s,
			ScopeSpans: []*ScopeSpans{{
				Scope:     utf8Scope(s),
				SchemaUrl: s,
				Spans: []*Span{{
					Name:       s,
					TraceState: s,
					Attributes: utf8KeyValues(s),
					Events:     []*SpanEvent{{Name: s, Attributes: utf8KeyValues(s)}},
					Links:      []*SpanLink{{TraceState: s, Attributes: utf8KeyValues(s)}},
					Status:     Status{Message: s, Code: 2},
				}},
			}},
			DeprecatedScopeSpans: []*ScopeSpans{{
				Scope: utf8Scope(s),
				Spans: []*Span{{Name: s}},
			}},
		}},
	}
}

func TestSanitizeUTF8ExportTraceServiceRequest(t *testing.T) {
	req := utf8TracesRequest(utf8InvalidStr)
	SanitizeUTF8ExportTraceServiceRequest(req)
	assert.Equal(t, utf8TracesRequest(utf8SanitizedStr), req)

	t.Run("empty request", func(t *testing.T) {
		req := &ExportTraceServiceRequest{}
		SanitizeUTF8ExportTraceServiceRequest(req)
		assert.Equal(t, &ExportTraceServiceRequest{}, req)
	})
}

func utf8Exemplars(s string) []Exemplar {
	return []Exemplar{{FilteredAttributes: utf8KeyValues(s), Value: &Exemplar_AsInt{AsInt: 1}}}
}

func utf8MetricsRequest(s string) *ExportMetricsServiceRequest {
	return &ExportMetricsServiceRequest{
		ResourceMetrics: []*ResourceMetrics{{
			Resource:  utf8Resource(s),
			SchemaUrl: s,
			ScopeMetrics: []*ScopeMetrics{{
				Scope:     utf8Scope(s),
				SchemaUrl: s,
				Metrics: []*Metric{
					{Name: s, Description: s, Unit: s, Metadata: utf8KeyValues(s), Data: &Metric_Gauge{Gauge: &Gauge{
						DataPoints: []*NumberDataPoint{{Attributes: utf8KeyValues(s), Exemplars: utf8Exemplars(s)}},
					}}},
					{Name: s, Data: &Metric_Sum{Sum: &Sum{
						DataPoints: []*NumberDataPoint{{Attributes: utf8KeyValues(s), Exemplars: utf8Exemplars(s)}},
					}}},
					{Name: s, Data: &Metric_Histogram{Histogram: &Histogram{
						DataPoints: []*HistogramDataPoint{{Attributes: utf8KeyValues(s), Exemplars: utf8Exemplars(s)}},
					}}},
					{Name: s, Data: &Metric_ExponentialHistogram{ExponentialHistogram: &ExponentialHistogram{
						DataPoints: []*ExponentialHistogramDataPoint{{Attributes: utf8KeyValues(s), Exemplars: utf8Exemplars(s)}},
					}}},
					{Name: s, Data: &Metric_Summary{Summary: &Summary{
						DataPoints: []*SummaryDataPoint{{Attributes: utf8KeyValues(s)}},
					}}},
					{Name: s},
					{Name: s, Data: &Metric_Gauge{}},
					{Name: s, Data: &Metric_Sum{}},
					{Name: s, Data: &Metric_Histogram{}},
					{Name: s, Data: &Metric_ExponentialHistogram{}},
					{Name: s, Data: &Metric_Summary{}},
				},
			}},
			DeprecatedScopeMetrics: []*ScopeMetrics{{
				Scope:   utf8Scope(s),
				Metrics: []*Metric{{Name: s}},
			}},
		}},
	}
}

func TestSanitizeUTF8ExportMetricsServiceRequest(t *testing.T) {
	req := utf8MetricsRequest(utf8InvalidStr)
	SanitizeUTF8ExportMetricsServiceRequest(req)
	assert.Equal(t, utf8MetricsRequest(utf8SanitizedStr), req)

	t.Run("empty request", func(t *testing.T) {
		req := &ExportMetricsServiceRequest{}
		SanitizeUTF8ExportMetricsServiceRequest(req)
		assert.Equal(t, &ExportMetricsServiceRequest{}, req)
	})
}

func utf8ProfilesRequest(s string) *ExportProfilesServiceRequest {
	return &ExportProfilesServiceRequest{
		Dictionary: ProfilesDictionary{
			StringTable:    []string{"", s, "ok"},
			AttributeTable: []*KeyValueAndUnit{{KeyStrindex: 1, Value: utf8StrValue(s), UnitStrindex: 2}},
		},
		ResourceProfiles: []*ResourceProfiles{{
			Resource:  utf8Resource(s),
			SchemaUrl: s,
			ScopeProfiles: []*ScopeProfiles{{
				Scope:     utf8Scope(s),
				SchemaUrl: s,
				Profiles: []*Profile{{
					OriginalPayloadFormat: s,
					OriginalPayload:       []byte(utf8InvalidStr),
					Samples:               []*Sample{{StackIndex: 1}},
				}},
			}},
		}},
	}
}

func TestSanitizeUTF8ExportProfilesServiceRequest(t *testing.T) {
	req := utf8ProfilesRequest(utf8InvalidStr)
	SanitizeUTF8ExportProfilesServiceRequest(req)
	assert.Equal(t, utf8ProfilesRequest(utf8SanitizedStr), req)

	t.Run("empty request", func(t *testing.T) {
		req := &ExportProfilesServiceRequest{}
		SanitizeUTF8ExportProfilesServiceRequest(req)
		assert.Equal(t, &ExportProfilesServiceRequest{}, req)
	})
}

func BenchmarkSanitizeUTF8ExportLogsServiceRequest(b *testing.B) {
	const records = 10_000
	newRequest := func(invalidEvery int) *ExportLogsServiceRequest {
		sl := &ScopeLogs{Scope: InstrumentationScope{Name: "scope", Version: "1.0.0"}}
		for i := range records {
			body := "log body with some text in it"
			if invalidEvery > 0 && i%invalidEvery == 0 {
				body = utf8InvalidStr
			}
			sl.LogRecords = append(sl.LogRecords, &LogRecord{
				SeverityText: "INFO",
				Body:         utf8StrValue(body),
				Attributes: []KeyValue{
					{Key: "http.method", Value: utf8StrValue("GET")},
					{Key: "http.route", Value: utf8StrValue("/api/v1/resource")},
					{Key: "http.status_code", Value: AnyValue{Value: &AnyValue_IntValue{IntValue: 200}}},
				},
			})
		}
		return &ExportLogsServiceRequest{ResourceLogs: []*ResourceLogs{{
			Resource:  Resource{Attributes: []KeyValue{{Key: "service.name", Value: utf8StrValue("svc")}}},
			ScopeLogs: []*ScopeLogs{sl},
		}}}
	}

	for _, bc := range []struct {
		name         string
		invalidEvery int
	}{
		{name: "AllValid", invalidEvery: 0},
		{name: "OneInvalid", invalidEvery: records},
		{name: "AllInvalid", invalidEvery: 1},
	} {
		b.Run(bc.name, func(b *testing.B) {
			req := newRequest(bc.invalidEvery)
			b.ReportAllocs()
			for b.Loop() {
				SanitizeUTF8ExportLogsServiceRequest(req)
				if bc.invalidEvery > 0 {
					b.StopTimer()
					req = newRequest(bc.invalidEvery)
					b.StartTimer()
				}
			}
		})
	}
}

// TestSanitizeUTF8CoversEveryStringField guards against drift between the hand-written walkers and the
// generated structs: it poisons every reachable string field of a generated test request through
// reflection, runs the sanitizer, and asserts that no invalid string is left behind.
func TestSanitizeUTF8CoversEveryStringField(t *testing.T) {
	tests := []struct {
		name    string
		request any
	}{
		{name: "logs", request: GenTestExportLogsServiceRequest()},
		{name: "traces", request: GenTestExportTraceServiceRequest()},
		{name: "metrics", request: GenTestExportMetricsServiceRequest()},
		{name: "profiles", request: GenTestExportProfilesServiceRequest()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poisoned := utf8PoisonStrings(reflect.ValueOf(tt.request))
			require.Positive(t, poisoned, "generated test request must contain string fields")

			switch req := tt.request.(type) {
			case *ExportLogsServiceRequest:
				SanitizeUTF8ExportLogsServiceRequest(req)
			case *ExportTraceServiceRequest:
				SanitizeUTF8ExportTraceServiceRequest(req)
			case *ExportMetricsServiceRequest:
				SanitizeUTF8ExportMetricsServiceRequest(req)
			case *ExportProfilesServiceRequest:
				SanitizeUTF8ExportProfilesServiceRequest(req)
			}

			var invalid []string
			utf8CollectInvalidStrings(reflect.ValueOf(tt.request), "", &invalid)
			assert.Empty(t, invalid, "string fields not sanitized")
		})
	}
}

// utf8PoisonStrings sets every settable string reachable from v to utf8InvalidStr and returns how many it set.
func utf8PoisonStrings(v reflect.Value) int {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return 0
		}
		return utf8PoisonStrings(v.Elem())
	case reflect.String:
		if !v.CanSet() {
			return 0
		}
		v.SetString(utf8InvalidStr)
		return 1
	case reflect.Struct:
		n := 0
		for _, fv := range v.Fields() {
			n += utf8PoisonStrings(fv)
		}
		return n
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return 0
		}
		n := 0
		for i := range v.Len() {
			n += utf8PoisonStrings(v.Index(i))
		}
		return n
	default:
		return 0
	}
}

func utf8CollectInvalidStrings(v reflect.Value, path string, invalid *[]string) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			utf8CollectInvalidStrings(v.Elem(), path, invalid)
		}
	case reflect.String:
		if !utf8.ValidString(v.String()) {
			*invalid = append(*invalid, path)
		}
	case reflect.Struct:
		for sf, fv := range v.Fields() {
			utf8CollectInvalidStrings(fv, path+"."+sf.Name, invalid)
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return
		}
		for i := range v.Len() {
			utf8CollectInvalidStrings(v.Index(i), fmt.Sprintf("%s[%d]", path, i), invalid)
		}
	}
}
