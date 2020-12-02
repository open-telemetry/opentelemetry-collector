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

package testbed

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
)

// TestCaseValidator defines the interface for validating and reporting test results.
type TestCaseValidator interface {
	// Validate executes validation routines and test assertions.
	Validate(tc *TestCase)
	// RecordResults updates the TestResultsSummary for the test suite with results of a single test.
	RecordResults(tc *TestCase)
}

// PerfTestValidator implements TestCaseValidator for test suites using PerformanceResults for summarizing results.
type PerfTestValidator struct {
}

func (v *PerfTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t, tc.LoadGenerator.DataItemsSent(), tc.MockBackend.DataItemsReceived(),
		"Received and sent counters do not match.") {
		log.Printf("Sent and received data matches.")
	}
}

func (v *PerfTestValidator) RecordResults(tc *TestCase) {
	rc := tc.agentProc.GetTotalConsumption()

	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:]

	tc.resultsSummary.Add(tc.t.Name(), &PerformanceTestResult{
		testName:          testName,
		result:            result,
		receivedSpanCount: tc.MockBackend.DataItemsReceived(),
		sentSpanCount:     tc.LoadGenerator.DataItemsSent(),
		duration:          time.Since(tc.startTime),
		cpuPercentageAvg:  rc.CPUPercentAvg,
		cpuPercentageMax:  rc.CPUPercentMax,
		ramMibAvg:         rc.RAMMiBAvg,
		ramMibMax:         rc.RAMMiBMax,
		errorCause:        tc.errorCause,
	})
}

// CorrectnessTestValidator implements TestCaseValidator for test suites using CorrectnessResults for summarizing results.
type CorrectnessTestValidator struct {
	dataProvider      DataProvider
	assertionFailures []*TraceAssertionFailure
}

func NewCorrectTestValidator(provider DataProvider) *CorrectnessTestValidator {
	return &CorrectnessTestValidator{
		dataProvider:      provider,
		assertionFailures: make([]*TraceAssertionFailure, 0),
	}
}

func (v *CorrectnessTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t, tc.LoadGenerator.DataItemsSent(), tc.MockBackend.DataItemsReceived(),
		"Received and sent counters do not match.") {
		log.Printf("Sent and received data counters match.")
	}
	if len(tc.MockBackend.ReceivedTraces) > 0 {
		v.assertSentRecdTracingDataEqual(tc.MockBackend.ReceivedTraces)
	}
	assert.EqualValues(tc.t, 0, len(v.assertionFailures), "There are span data mismatches.")
}

func (v *CorrectnessTestValidator) RecordResults(tc *TestCase) {
	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:]
	tc.resultsSummary.Add(tc.t.Name(), &CorrectnessTestResult{
		testName:                   testName,
		result:                     result,
		duration:                   time.Since(tc.startTime),
		receivedSpanCount:          tc.MockBackend.DataItemsReceived(),
		sentSpanCount:              tc.LoadGenerator.DataItemsSent(),
		traceAssertionFailureCount: uint64(len(v.assertionFailures)),
		traceAssertionFailures:     v.assertionFailures,
	})
}

func (v *CorrectnessTestValidator) assertSentRecdTracingDataEqual(tracesList []pdata.Traces) {
	for _, td := range tracesList {
		resourceSpansList := pdata.TracesToOtlp(td)
		for _, rs := range resourceSpansList {
			for _, ils := range rs.InstrumentationLibrarySpans {
				for _, recdSpan := range ils.Spans {
					sentSpan := v.dataProvider.GetGeneratedSpan(pdata.TraceID(recdSpan.TraceId), pdata.SpanID(recdSpan.SpanId))
					v.diffSpan(sentSpan, recdSpan)
				}
			}
		}

	}
}

func (v *CorrectnessTestValidator) diffSpan(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan == nil {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
		return
	}
	v.diffSpanTraceID(sentSpan, recdSpan)
	v.diffSpanSpanID(sentSpan, recdSpan)
	v.diffSpanTraceState(sentSpan, recdSpan)
	v.diffSpanParentSpanID(sentSpan, recdSpan)
	v.diffSpanName(sentSpan, recdSpan)
	v.diffSpanKind(sentSpan, recdSpan)
	v.diffSpanTimestamps(sentSpan, recdSpan)
	v.diffSpanAttributes(sentSpan, recdSpan)
	v.diffSpanEvents(sentSpan, recdSpan)
	v.diffSpanLinks(sentSpan, recdSpan)
	v.diffSpanStatus(sentSpan, recdSpan)
}

func (v *CorrectnessTestValidator) diffSpanTraceID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.TraceId.HexString() != recdSpan.TraceId.HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceId",
			expectedValue: sentSpan.TraceId.HexString(),
			actualValue:   recdSpan.TraceId.HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanSpanID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.SpanId.HexString() != recdSpan.SpanId.HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "SpanId",
			expectedValue: sentSpan.SpanId.HexString(),
			actualValue:   recdSpan.SpanId.HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanTraceState(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.TraceState != recdSpan.TraceState {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceState",
			expectedValue: sentSpan.TraceState,
			actualValue:   recdSpan.TraceState,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanParentSpanID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.ParentSpanId.HexString() != recdSpan.ParentSpanId.HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "ParentSpanId",
			expectedValue: sentSpan.ParentSpanId.HexString(),
			actualValue:   recdSpan.ParentSpanId.HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanName(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	// Because of https://github.com/openzipkin/zipkin-go/pull/166 compare lower cases.
	if !strings.EqualFold(sentSpan.Name, recdSpan.Name) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Name",
			expectedValue: sentSpan.Name,
			actualValue:   recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanKind(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.Kind != recdSpan.Kind {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Kind",
			expectedValue: sentSpan.Kind,
			actualValue:   recdSpan.Kind,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanTimestamps(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if notWithinOneMillisecond(sentSpan.StartTimeUnixNano, recdSpan.StartTimeUnixNano) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "StartTimeUnixNano",
			expectedValue: sentSpan.StartTimeUnixNano,
			actualValue:   recdSpan.StartTimeUnixNano,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if notWithinOneMillisecond(sentSpan.EndTimeUnixNano, recdSpan.EndTimeUnixNano) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "EndTimeUnixNano",
			expectedValue: sentSpan.EndTimeUnixNano,
			actualValue:   recdSpan.EndTimeUnixNano,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanAttributes(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Attributes) != len(recdSpan.Attributes) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Attributes",
			expectedValue: len(sentSpan.Attributes),
			actualValue:   len(recdSpan.Attributes),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		sentAttrs := sentSpan.Attributes
		recdAttrs := recdSpan.Attributes
		v.diffAttributesSlice(sentSpan.Name, recdAttrs, sentAttrs, "Attributes[%s]")
	}
	if sentSpan.DroppedAttributesCount != recdSpan.DroppedAttributesCount {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedAttributesCount",
			expectedValue: sentSpan.DroppedAttributesCount,
			actualValue:   recdSpan.DroppedAttributesCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanEvents(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Events) != len(recdSpan.Events) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Events",
			expectedValue: len(sentSpan.Events),
			actualValue:   len(recdSpan.Events),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		sentEventMap := convertEventsSliceToMap(sentSpan.Events)
		recdEventMap := convertEventsSliceToMap(recdSpan.Events)
		for name, sentEvents := range sentEventMap {
			recdEvents, match := recdEventMap[name]
			if match {
				match = len(sentEvents) == len(recdEvents)
			}
			if !match {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name,
					fieldPath:     fmt.Sprintf("Events[%s]", name),
					expectedValue: len(sentEvents),
					actualValue:   len(recdEvents),
				}
				v.assertionFailures = append(v.assertionFailures, af)
			} else {
				for i, sentEvent := range sentEvents {
					recdEvent := recdEvents[i]
					if notWithinOneMillisecond(sentEvent.TimeUnixNano, recdEvent.TimeUnixNano) {
						af := &TraceAssertionFailure{
							typeName:      "Span",
							dataComboName: sentSpan.Name,
							fieldPath:     fmt.Sprintf("Events[%s].TimeUnixNano", name),
							expectedValue: sentEvent.TimeUnixNano,
							actualValue:   recdEvent.TimeUnixNano,
						}
						v.assertionFailures = append(v.assertionFailures, af)
					}
					v.diffAttributesSlice(sentSpan.Name, sentEvent.Attributes, recdEvent.Attributes,
						"Events["+name+"].Attributes[%s]")
				}
			}
		}
	}
	if sentSpan.DroppedEventsCount != recdSpan.DroppedEventsCount {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedEventsCount",
			expectedValue: sentSpan.DroppedEventsCount,
			actualValue:   recdSpan.DroppedEventsCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanLinks(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Links) != len(recdSpan.Links) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Links",
			expectedValue: len(sentSpan.Links),
			actualValue:   len(recdSpan.Links),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		recdLinksMap := convertLinksSliceToMap(recdSpan.Links)
		for i, sentLink := range sentSpan.Links {
			spanID := sentLink.SpanId.HexString()
			recdLink, ok := recdLinksMap[spanID]
			if ok {
				v.diffAttributesSlice(sentSpan.Name, sentLink.Attributes, recdLink.Attributes,
					"Links["+spanID+"].Attributes[%s]")
			} else {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name,
					fieldPath:     fmt.Sprintf("Links[%d]", i),
					expectedValue: spanID,
					actualValue:   "",
				}
				v.assertionFailures = append(v.assertionFailures, af)
			}

		}
	}
	if sentSpan.DroppedLinksCount != recdSpan.DroppedLinksCount {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedLinksCount",
			expectedValue: sentSpan.DroppedLinksCount,
			actualValue:   recdSpan.DroppedLinksCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanStatus(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.Status.Code != recdSpan.Status.Code {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Status.Code",
			expectedValue: sentSpan.Status.Code,
			actualValue:   recdSpan.Status.Code,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffAttributesSlice(spanName string, recdAttrs []otlpcommon.KeyValue,
	sentAttrs []otlpcommon.KeyValue, fmtStr string) {
	recdAttrsMap := convertAttributesSliceToMap(recdAttrs)
	for _, sentAttr := range sentAttrs {
		recdAttr, ok := recdAttrsMap[sentAttr.Key]
		if ok {
			sentVal := retrieveAttributeValue(sentAttr)
			recdVal := retrieveAttributeValue(recdAttr)
			switch val := sentVal.(type) {
			case *otlpcommon.KeyValueList:
				v.compareKeyValueList(spanName, val, recdVal, fmtStr, sentAttr.Key)
			case *otlpcommon.ArrayValue:
				v.compareArrayList(spanName, val, recdVal, fmtStr, sentAttr.Key)
			default:
				v.compareSimpleValues(spanName, sentVal, recdVal, fmtStr, sentAttr.Key)
			}

		} else {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf("Attributes[%s]", sentAttr.Key),
				expectedValue: retrieveAttributeValue(sentAttr),
				actualValue:   nil,
			}
			v.assertionFailures = append(v.assertionFailures, af)
		}
	}
}

func (v *CorrectnessTestValidator) compareSimpleValues(spanName string, sentVal interface{}, recdVal interface{},
	fmtStr string, attrKey string) {
	if !reflect.DeepEqual(sentVal, recdVal) {
		sentStr := fmt.Sprintf("%v", sentVal)
		recdStr := fmt.Sprintf("%v", recdVal)
		if !strings.EqualFold(sentStr, recdStr) {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf(fmtStr, attrKey),
				expectedValue: sentVal,
				actualValue:   recdVal,
			}
			v.assertionFailures = append(v.assertionFailures, af)
		}
	}
}

func (v *CorrectnessTestValidator) compareKeyValueList(spanName string, sentKVList *otlpcommon.KeyValueList,
	recdVal interface{}, fmtStr string, attrKey string) {
	switch val := recdVal.(type) {
	case *otlpcommon.KeyValueList:
		v.diffAttributesSlice(spanName, val.Values, sentKVList.Values, fmtStr)
	case string:
		jsonStr := convertKVListToJSONString(sentKVList.Values)
		v.compareSimpleValues(spanName, jsonStr, val, fmtStr, attrKey)
	default:
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: spanName,
			fieldPath:     fmt.Sprintf(fmtStr, attrKey),
			expectedValue: sentKVList,
			actualValue:   recdVal,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) compareArrayList(spanName string, sentArray *otlpcommon.ArrayValue,
	recdVal interface{}, fmtStr string, attrKey string) {
	switch val := recdVal.(type) {
	case *otlpcommon.ArrayValue:
		v.compareSimpleValues(spanName, sentArray.Values, val.Values, fmtStr, attrKey)
	case string:
		jsonStr := convertArrayValuesToJSONString(sentArray.Values)
		v.compareSimpleValues(spanName, jsonStr, val, fmtStr, attrKey)
	default:
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: spanName,
			fieldPath:     fmt.Sprintf(fmtStr, attrKey),
			expectedValue: sentArray,
			actualValue:   recdVal,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func convertAttributesSliceToMap(attributes []otlpcommon.KeyValue) map[string]otlpcommon.KeyValue {
	attrMap := make(map[string]otlpcommon.KeyValue)
	for _, attr := range attributes {
		attrMap[attr.Key] = attr
	}
	return attrMap
}

func retrieveAttributeValue(attribute otlpcommon.KeyValue) interface{} {
	if attribute.Value.Value == nil {
		return nil
	}

	var attrVal interface{}
	switch val := attribute.Value.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		// Because of https://github.com/openzipkin/zipkin-go/pull/166 compare lower cases.
		attrVal = strings.ToLower(val.StringValue)
	case *otlpcommon.AnyValue_IntValue:
		attrVal = val.IntValue
	case *otlpcommon.AnyValue_DoubleValue:
		attrVal = val.DoubleValue
	case *otlpcommon.AnyValue_BoolValue:
		attrVal = val.BoolValue
	case *otlpcommon.AnyValue_ArrayValue:
		attrVal = val.ArrayValue
	case *otlpcommon.AnyValue_KvlistValue:
		attrVal = val.KvlistValue
	default:
		attrVal = nil
	}
	return attrVal
}

func convertEventsSliceToMap(events []*otlptrace.Span_Event) map[string][]*otlptrace.Span_Event {
	eventMap := make(map[string][]*otlptrace.Span_Event)
	for _, event := range events {
		evtSlice, ok := eventMap[event.Name]
		if !ok {
			evtSlice = make([]*otlptrace.Span_Event, 0)
		}
		eventMap[event.Name] = append(evtSlice, event)
	}
	for _, eventList := range eventMap {
		sortEventsByTimestamp(eventList)
	}
	return eventMap
}

func sortEventsByTimestamp(eventList []*otlptrace.Span_Event) {
	sort.SliceStable(eventList, func(i, j int) bool { return eventList[i].TimeUnixNano < eventList[j].TimeUnixNano })
}

func convertLinksSliceToMap(links []*otlptrace.Span_Link) map[string]*otlptrace.Span_Link {
	eventMap := make(map[string]*otlptrace.Span_Link)
	for _, link := range links {
		eventMap[link.SpanId.HexString()] = link
	}
	return eventMap
}

func notWithinOneMillisecond(sentNs uint64, recdNs uint64) bool {
	var diff uint64
	if sentNs > recdNs {
		diff = sentNs - recdNs
	} else {
		diff = recdNs - sentNs
	}
	return diff > uint64(1100000)
}

func convertKVListToJSONString(values []otlpcommon.KeyValue) string {
	jsonStr, err := json.Marshal(convertKVListToRawMap(values))
	if err == nil {
		return string(jsonStr)
	}
	return ""
}

func convertArrayValuesToJSONString(values []otlpcommon.AnyValue) string {
	jsonStr, err := json.Marshal(convertArrayValuesToRawSlice(values))
	if err == nil {
		return string(jsonStr)
	}
	return ""
}

func convertKVListToRawMap(values []otlpcommon.KeyValue) map[string]interface{} {
	rawMap := make(map[string]interface{})
	for i := range values {
		kv := &values[i]
		switch val := kv.Value.GetValue().(type) {
		case *otlpcommon.AnyValue_StringValue:
			rawMap[kv.Key] = val.StringValue
		case *otlpcommon.AnyValue_IntValue:
			rawMap[kv.Key] = val.IntValue
		case *otlpcommon.AnyValue_DoubleValue:
			rawMap[kv.Key] = val.DoubleValue
		case *otlpcommon.AnyValue_BoolValue:
			rawMap[kv.Key] = val.BoolValue
		case *otlpcommon.AnyValue_KvlistValue:
			rawMap[kv.Key] = convertKVListToRawMap(val.KvlistValue.Values)
		case *otlpcommon.AnyValue_ArrayValue:
			rawMap[kv.Key] = convertArrayValuesToRawSlice(val.ArrayValue.Values)
		default:
			rawMap[kv.Key] = val
		}
	}
	return rawMap
}

func convertArrayValuesToRawSlice(values []otlpcommon.AnyValue) []interface{} {
	rawSlice := make([]interface{}, 0, len(values))
	for _, v := range values {
		switch val := v.GetValue().(type) {
		case *otlpcommon.AnyValue_StringValue:
			rawSlice = append(rawSlice, val.StringValue)
		case *otlpcommon.AnyValue_IntValue:
			rawSlice = append(rawSlice, val.IntValue)
		case *otlpcommon.AnyValue_DoubleValue:
			rawSlice = append(rawSlice, val.DoubleValue)
		case *otlpcommon.AnyValue_BoolValue:
			rawSlice = append(rawSlice, val.BoolValue)
		}
	}
	return rawSlice
}
