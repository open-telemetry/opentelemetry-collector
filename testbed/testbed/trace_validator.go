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
	"encoding/hex"
	"fmt"
	"log"
	"reflect"
	"sort"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/common/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// CorrectnessTestTraceValidator implements TestCaseValidator for test suites using CorrectnessResults for summarizing results.
type CorrectnessTestTraceValidator struct {
	dataProvider             DataProvider
	tracingAssertionFailures []*TracingAssertionFailure
}

func NewCorrectnessTestTraceValidator(provider DataProvider) *CorrectnessTestTraceValidator {
	return &CorrectnessTestTraceValidator{
		dataProvider:             provider,
		tracingAssertionFailures: make([]*TracingAssertionFailure, 0),
	}
}

func (v *CorrectnessTestTraceValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t, tc.LoadGenerator.DataItemsSent(), tc.MockBackend.DataItemsReceived(),
		"Received and sent counters do not match.",
	) {
		log.Printf("Sent and received data counters match.")
	}
	if len(tc.MockBackend.ReceivedTraces) > 0 {
		v.assertSentRecdTracingDataEqual(tc.MockBackend.ReceivedTraces)
	}
	if len(tc.MockBackend.ReceivedTracesOld) > 0 {
		tracesList := make([]pdata.Traces, 0, len(tc.MockBackend.ReceivedTracesOld))
		for _, td := range tc.MockBackend.ReceivedTracesOld {
			tracesList = append(tracesList, internaldata.OCToTraceData(td))
		}
		v.assertSentRecdTracingDataEqual(tracesList)
	}
	// TODO enable once identified problems are fixed
	// assert.EqualValues(tc.t, 0, len(v.assertionFailures), "There are span data mismatches.")
}

func (v *CorrectnessTestTraceValidator) RecordResults(tc *TestCase) {
	var result string
	if tc.t.Failed() {
		result = "FAIL"
	} else {
		result = "PASS"
	}

	// Remove "Test" prefix from test name.
	testName := tc.t.Name()[4:]
	tc.resultsSummary.Add(tc.t.Name(), &CorrectnessTestResult{
		testName:                     testName,
		result:                       result,
		duration:                     time.Since(tc.startTime),
		receivedSpanCount:            tc.MockBackend.DataItemsReceived(),
		sentSpanCount:                tc.LoadGenerator.DataItemsSent(),
		tracingAssertionFailureCount: uint64(len(v.tracingAssertionFailures)),
		tracingAssertionFailures:     v.tracingAssertionFailures,
	})
}

func (v *CorrectnessTestTraceValidator) assertSentRecdTracingDataEqual(tracesList []pdata.Traces) {
	for _, td := range tracesList {
		resourceSpansList := pdata.TracesToOtlp(td)
		for _, rs := range resourceSpansList {
			for _, ils := range rs.InstrumentationLibrarySpans {
				for _, recdSpan := range ils.Spans {
					sentSpan := v.dataProvider.GetGeneratedSpan(recdSpan.TraceId, recdSpan.SpanId)
					v.diffSpan(sentSpan, recdSpan)
				}
			}
		}

	}
}

func (v *CorrectnessTestTraceValidator) diffSpan(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan == nil {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: recdSpan.Name,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
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

func (v *CorrectnessTestTraceValidator) diffSpanTraceID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if hex.EncodeToString(sentSpan.TraceId) != hex.EncodeToString(recdSpan.TraceId) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceId",
			expectedValue: hex.EncodeToString(sentSpan.TraceId),
			actualValue:   hex.EncodeToString(recdSpan.TraceId),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanSpanID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if hex.EncodeToString(sentSpan.SpanId) != hex.EncodeToString(recdSpan.SpanId) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "SpanId",
			expectedValue: hex.EncodeToString(sentSpan.SpanId),
			actualValue:   hex.EncodeToString(recdSpan.SpanId),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanTraceState(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.TraceState != recdSpan.TraceState {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceState",
			expectedValue: sentSpan.TraceState,
			actualValue:   recdSpan.TraceState,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanParentSpanID(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if hex.EncodeToString(sentSpan.ParentSpanId) != hex.EncodeToString(recdSpan.ParentSpanId) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "ParentSpanId",
			expectedValue: hex.EncodeToString(sentSpan.ParentSpanId),
			actualValue:   hex.EncodeToString(recdSpan.ParentSpanId),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanName(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.Name != recdSpan.Name {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Name",
			expectedValue: sentSpan.Name,
			actualValue:   recdSpan.Name,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanKind(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.Kind != recdSpan.Kind {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Kind",
			expectedValue: sentSpan.Kind,
			actualValue:   recdSpan.Kind,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanTimestamps(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.StartTimeUnixNano != recdSpan.StartTimeUnixNano {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "StartTimeUnixNano",
			expectedValue: sentSpan.StartTimeUnixNano,
			actualValue:   recdSpan.StartTimeUnixNano,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
	if sentSpan.EndTimeUnixNano != recdSpan.EndTimeUnixNano {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "StartTimeUnixNano",
			expectedValue: sentSpan.EndTimeUnixNano,
			actualValue:   recdSpan.EndTimeUnixNano,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanAttributes(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Attributes) != len(recdSpan.Attributes) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Attributes",
			expectedValue: len(sentSpan.Attributes),
			actualValue:   len(recdSpan.Attributes),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	} else {
		sentAttrs := sentSpan.Attributes
		recdAttrs := recdSpan.Attributes
		v.diffAttributesSlice(sentSpan.Name, recdAttrs, sentAttrs, "Attributes[%s]")
	}
	if sentSpan.DroppedAttributesCount != recdSpan.DroppedAttributesCount {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedAttributesCount",
			expectedValue: sentSpan.DroppedAttributesCount,
			actualValue:   recdSpan.DroppedAttributesCount,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanEvents(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Events) != len(recdSpan.Events) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Events",
			expectedValue: len(sentSpan.Events),
			actualValue:   len(recdSpan.Events),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	} else {
		sentEventMap := convertEventsSliceToMap(sentSpan.Events)
		recdEventMap := convertEventsSliceToMap(recdSpan.Events)
		for name, sentEvents := range sentEventMap {
			recdEvents, match := recdEventMap[name]
			if match {
				match = len(sentEvents) == len(recdEvents)
			}
			if !match {
				af := &TracingAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name,
					fieldPath:     fmt.Sprintf("Events[%s]", name),
					expectedValue: len(sentEvents),
					actualValue:   len(recdEvents),
				}
				v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
			} else {
				for i, sentEvent := range sentEvents {
					recdEvent := recdEvents[i]
					if sentEvent.TimeUnixNano != recdEvent.TimeUnixNano {
						af := &TracingAssertionFailure{
							typeName:      "Span",
							dataComboName: sentSpan.Name,
							fieldPath:     fmt.Sprintf("Events[%s].TimeUnixNano", name),
							expectedValue: sentEvent.TimeUnixNano,
							actualValue:   recdEvent.TimeUnixNano,
						}
						v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
					}
					v.diffAttributesSlice(sentSpan.Name, sentEvent.Attributes, recdEvent.Attributes,
						"Events["+name+"].Attributes[%s]")
				}
			}
		}
	}
	if sentSpan.DroppedEventsCount != recdSpan.DroppedEventsCount {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedEventsCount",
			expectedValue: sentSpan.DroppedEventsCount,
			actualValue:   recdSpan.DroppedEventsCount,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanLinks(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if len(sentSpan.Links) != len(recdSpan.Links) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Links",
			expectedValue: len(sentSpan.Links),
			actualValue:   len(recdSpan.Links),
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	} else {
		recdLinksMap := convertLinksSliceToMap(recdSpan.Links)
		for i, sentLink := range sentSpan.Links {
			spanID := hex.EncodeToString(sentLink.SpanId)
			recdLink, ok := recdLinksMap[spanID]
			if ok {
				v.diffAttributesSlice(sentSpan.Name, sentLink.Attributes, recdLink.Attributes,
					"Links["+spanID+"].Attributes[%s]")
			} else {
				af := &TracingAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name,
					fieldPath:     fmt.Sprintf("Links[%d]", i),
					expectedValue: spanID,
					actualValue:   "",
				}
				v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
			}

		}
	}
	if sentSpan.DroppedLinksCount != recdSpan.DroppedLinksCount {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedLinksCount",
			expectedValue: sentSpan.DroppedLinksCount,
			actualValue:   recdSpan.DroppedLinksCount,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffSpanStatus(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan.Status != nil && recdSpan.Status != nil {
		if sentSpan.Status.Code != recdSpan.Status.Code {
			af := &TracingAssertionFailure{
				typeName:      "Span",
				dataComboName: sentSpan.Name,
				fieldPath:     "Status.Code",
				expectedValue: sentSpan.Status.Code,
				actualValue:   recdSpan.Status.Code,
			}
			v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
		}
	} else if (sentSpan.Status != nil && recdSpan.Status == nil) || (sentSpan.Status == nil && recdSpan.Status != nil) {
		af := &TracingAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Status",
			expectedValue: sentSpan.Status,
			actualValue:   recdSpan.Status,
		}
		v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
	}
}

func (v *CorrectnessTestTraceValidator) diffAttributesSlice(spanName string, recdAttrs []*otlpcommon.KeyValue,
	sentAttrs []*otlpcommon.KeyValue, fmtStr string) {
	recdAttrsMap := convertAttributesSliceToMap(recdAttrs)
	for _, sentAttr := range sentAttrs {
		recdAttr, ok := recdAttrsMap[sentAttr.Key]
		if ok {
			sentVal := retrieveAttributeValue(sentAttr)
			recdVal := retrieveAttributeValue(recdAttr)
			if !reflect.DeepEqual(sentVal, recdVal) {
				af := &TracingAssertionFailure{
					typeName:      "Span",
					dataComboName: spanName,
					fieldPath:     fmt.Sprintf(fmtStr, sentAttr.Key),
					expectedValue: sentVal,
					actualValue:   recdVal,
				}
				v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
			}
		} else {
			af := &TracingAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf("Attributes[%s]", sentAttr.Key),
				expectedValue: retrieveAttributeValue(sentAttr),
				actualValue:   nil,
			}
			v.tracingAssertionFailures = append(v.tracingAssertionFailures, af)
		}
	}
}

func convertAttributesSliceToMap(attributes []*otlpcommon.KeyValue) map[string]*otlpcommon.KeyValue {
	attrMap := make(map[string]*otlpcommon.KeyValue)
	for _, attr := range attributes {
		attrMap[attr.Key] = attr
	}
	return attrMap
}

func retrieveAttributeValue(attribute *otlpcommon.KeyValue) interface{} {
	if attribute.Value == nil || attribute.Value.Value == nil {
		return nil
	}

	var attrVal interface{}
	switch val := attribute.Value.Value.(type) {
	case *otlpcommon.AnyValue_StringValue:
		attrVal = val.StringValue
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
		eventMap[hex.EncodeToString(link.SpanId)] = link
	}
	return eventMap
}
