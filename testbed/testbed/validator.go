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
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

// TestCaseValidator defines the interface for validating and reporting test results.
type TestCaseValidator interface {
	// Validate executes validation routines and test assertions.
	Validate(tc *TestCase)
	// RecordResults updates the TestResultsSummary for the test suite with results of a single test.
	RecordResults(tc *TestCase)
}

// PerfTestValidator implements TestCaseValidator for test suites using PerformanceResults for summarizing results.
type PerfTestValidator struct{}

func (v *PerfTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t,
		int64(tc.LoadGenerator.DataItemsSent()),
		int64(tc.MockBackend.DataItemsReceived()),
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
	dataProvider         DataProvider
	assertionFailures    []*TraceAssertionFailure
	ignoreSpanLinksAttrs bool
}

func NewCorrectTestValidator(senderName string, receiverName string, provider DataProvider) *CorrectnessTestValidator {
	// TODO: Fix Jaeger span links attributes and tracestate.
	return &CorrectnessTestValidator{
		dataProvider:         provider,
		assertionFailures:    make([]*TraceAssertionFailure, 0),
		ignoreSpanLinksAttrs: senderName == "jaeger" || receiverName == "jaeger",
	}
}

func (v *CorrectnessTestValidator) Validate(tc *TestCase) {
	if assert.EqualValues(tc.t,
		int64(tc.LoadGenerator.DataItemsSent()),
		int64(tc.MockBackend.DataItemsReceived()),
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
	spansMap := make(map[string]pdata.Span)
	// TODO: Remove this hack, and add a way to retrieve all sent data.
	if val, ok := v.dataProvider.(*GoldenDataProvider); ok {
		populateSpansMap(spansMap, val.tracesGenerated)
	}

	for _, td := range tracesList {
		rss := td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			ilss := rss.At(i).InstrumentationLibrarySpans()
			for j := 0; j < ilss.Len(); j++ {
				spans := ilss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					recdSpan := spans.At(k)
					sentSpan := spansMap[traceIDAndSpanIDToString(recdSpan.TraceID(), recdSpan.SpanID())]
					v.diffSpan(sentSpan, recdSpan)
				}
			}
		}
	}
}

func (v *CorrectnessTestValidator) diffSpan(sentSpan pdata.Span, recdSpan pdata.Span) {
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

func (v *CorrectnessTestValidator) diffSpanTraceID(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.TraceID().HexString() != recdSpan.TraceID().HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "TraceId",
			expectedValue: sentSpan.TraceID().HexString(),
			actualValue:   recdSpan.TraceID().HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanSpanID(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.SpanID().HexString() != recdSpan.SpanID().HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "SpanId",
			expectedValue: sentSpan.SpanID().HexString(),
			actualValue:   recdSpan.SpanID().HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanTraceState(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.TraceState() != recdSpan.TraceState() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "TraceState",
			expectedValue: sentSpan.TraceState,
			actualValue:   recdSpan.TraceState,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanParentSpanID(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.ParentSpanID().HexString() != recdSpan.ParentSpanID().HexString() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "ParentSpanId",
			expectedValue: sentSpan.ParentSpanID().HexString(),
			actualValue:   recdSpan.ParentSpanID().HexString(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanName(sentSpan pdata.Span, recdSpan pdata.Span) {
	// Because of https://github.com/openzipkin/zipkin-go/pull/166 compare lower cases.
	if !strings.EqualFold(sentSpan.Name(), recdSpan.Name()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Name",
			expectedValue: sentSpan.Name(),
			actualValue:   recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanKind(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.Kind() != recdSpan.Kind() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Kind",
			expectedValue: sentSpan.Kind,
			actualValue:   recdSpan.Kind,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanTimestamps(sentSpan pdata.Span, recdSpan pdata.Span) {
	if notWithinOneMillisecond(sentSpan.StartTimestamp(), recdSpan.StartTimestamp()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "StartTimestamp",
			expectedValue: sentSpan.StartTimestamp(),
			actualValue:   recdSpan.StartTimestamp(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if notWithinOneMillisecond(sentSpan.EndTimestamp(), recdSpan.EndTimestamp()) {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "EndTimestamp",
			expectedValue: sentSpan.EndTimestamp(),
			actualValue:   recdSpan.EndTimestamp(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanAttributes(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.Attributes().Len() != recdSpan.Attributes().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Attributes",
			expectedValue: sentSpan.Attributes().Len(),
			actualValue:   recdSpan.Attributes().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		v.diffAttributeMap(sentSpan.Name(), sentSpan.Attributes(), recdSpan.Attributes(), "Attributes[%s]")
	}
	if sentSpan.DroppedAttributesCount() != recdSpan.DroppedAttributesCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedAttributesCount",
			expectedValue: sentSpan.DroppedAttributesCount(),
			actualValue:   recdSpan.DroppedAttributesCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanEvents(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.Events().Len() != recdSpan.Events().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Events",
			expectedValue: sentSpan.Events().Len(),
			actualValue:   recdSpan.Events().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		sentEventMap := convertEventsSliceToMap(sentSpan.Events())
		recdEventMap := convertEventsSliceToMap(recdSpan.Events())
		for name, sentEvents := range sentEventMap {
			recdEvents, match := recdEventMap[name]
			if match {
				match = len(sentEvents) == len(recdEvents)
			}
			if !match {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name(),
					fieldPath:     fmt.Sprintf("Events[%s]", name),
					expectedValue: len(sentEvents),
					actualValue:   len(recdEvents),
				}
				v.assertionFailures = append(v.assertionFailures, af)
			} else {
				for i, sentEvent := range sentEvents {
					recdEvent := recdEvents[i]
					if notWithinOneMillisecond(sentEvent.Timestamp(), recdEvent.Timestamp()) {
						af := &TraceAssertionFailure{
							typeName:      "Span",
							dataComboName: sentSpan.Name(),
							fieldPath:     fmt.Sprintf("Events[%s].TimeUnixNano", name),
							expectedValue: sentEvent.Timestamp(),
							actualValue:   recdEvent.Timestamp(),
						}
						v.assertionFailures = append(v.assertionFailures, af)
					}
					v.diffAttributeMap(sentSpan.Name(), sentEvent.Attributes(), recdEvent.Attributes(),
						"Events["+name+"].Attributes[%s]")
				}
			}
		}
	}
	if sentSpan.DroppedEventsCount() != recdSpan.DroppedEventsCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedEventsCount",
			expectedValue: sentSpan.DroppedEventsCount(),
			actualValue:   recdSpan.DroppedEventsCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanLinks(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.Links().Len() != recdSpan.Links().Len() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Links",
			expectedValue: sentSpan.Links().Len(),
			actualValue:   recdSpan.Links().Len(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	} else {
		recdLinksMap := convertLinksSliceToMap(recdSpan.Links())
		sentSpanLinks := sentSpan.Links()
		for i := 0; i < sentSpanLinks.Len(); i++ {
			sentLink := sentSpanLinks.At(i)
			recdLink, ok := recdLinksMap[traceIDAndSpanIDToString(sentLink.TraceID(), sentLink.SpanID())]
			if ok {
				if v.ignoreSpanLinksAttrs {
					return
				}
				if sentLink.TraceState() != recdLink.TraceState() {
					af := &TraceAssertionFailure{
						typeName:      "Span",
						dataComboName: sentSpan.Name(),
						fieldPath:     "Link.TraceState",
						expectedValue: sentLink,
						actualValue:   recdLink,
					}
					v.assertionFailures = append(v.assertionFailures, af)
				}
				v.diffAttributeMap(sentSpan.Name(), sentLink.Attributes(), recdLink.Attributes(),
					"Link.Attributes[%s]")
			} else {
				af := &TraceAssertionFailure{
					typeName:      "Span",
					dataComboName: sentSpan.Name(),
					fieldPath:     fmt.Sprintf("Links[%d]", i),
					expectedValue: sentLink,
					actualValue:   nil,
				}
				v.assertionFailures = append(v.assertionFailures, af)
			}

		}
	}
	if sentSpan.DroppedLinksCount() != recdSpan.DroppedLinksCount() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "DroppedLinksCount",
			expectedValue: sentSpan.DroppedLinksCount(),
			actualValue:   recdSpan.DroppedLinksCount(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffSpanStatus(sentSpan pdata.Span, recdSpan pdata.Span) {
	if sentSpan.Status().Code() != recdSpan.Status().Code() {
		af := &TraceAssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name(),
			fieldPath:     "Status.Code",
			expectedValue: sentSpan.Status().Code(),
			actualValue:   recdSpan.Status().Code(),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func (v *CorrectnessTestValidator) diffAttributeMap(spanName string,
	sentAttrs pdata.AttributeMap, recdAttrs pdata.AttributeMap, fmtStr string) {
	sentAttrs.Range(func(sentKey string, sentVal pdata.AttributeValue) bool {
		recdVal, ok := recdAttrs.Get(sentKey)
		if !ok {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf(fmtStr, sentKey),
				expectedValue: sentVal,
				actualValue:   nil,
			}
			v.assertionFailures = append(v.assertionFailures, af)
			return true
		}
		switch sentVal.Type() {
		case pdata.AttributeValueTypeMap:
			v.compareKeyValueList(spanName, sentVal, recdVal, fmtStr, sentKey)
		default:
			v.compareSimpleValues(spanName, sentVal, recdVal, fmtStr, sentKey)
		}
		return true
	})
}

func (v *CorrectnessTestValidator) compareSimpleValues(spanName string, sentVal pdata.AttributeValue, recdVal pdata.AttributeValue,
	fmtStr string, attrKey string) {
	if !sentVal.Equal(recdVal) {
		sentStr := tracetranslator.AttributeValueToString(sentVal)
		recdStr := tracetranslator.AttributeValueToString(recdVal)
		if !strings.EqualFold(sentStr, recdStr) {
			af := &TraceAssertionFailure{
				typeName:      "Span",
				dataComboName: spanName,
				fieldPath:     fmt.Sprintf(fmtStr, attrKey),
				expectedValue: tracetranslator.AttributeValueToString(sentVal),
				actualValue:   tracetranslator.AttributeValueToString(recdVal),
			}
			v.assertionFailures = append(v.assertionFailures, af)
		}
	}
}

func (v *CorrectnessTestValidator) compareKeyValueList(
	spanName string, sentVal pdata.AttributeValue, recdVal pdata.AttributeValue, fmtStr string, attrKey string) {
	switch recdVal.Type() {
	case pdata.AttributeValueTypeMap:
		v.diffAttributeMap(spanName, sentVal.MapVal(), recdVal.MapVal(), fmtStr)
	case pdata.AttributeValueTypeString:
		v.compareSimpleValues(spanName, sentVal, recdVal, fmtStr, attrKey)
	default:
		af := &TraceAssertionFailure{
			typeName:      "SimpleAttribute",
			dataComboName: spanName,
			fieldPath:     fmt.Sprintf(fmtStr, attrKey),
			expectedValue: sentVal,
			actualValue:   recdVal,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}

func convertEventsSliceToMap(events pdata.SpanEventSlice) map[string][]pdata.SpanEvent {
	eventMap := make(map[string][]pdata.SpanEvent)
	for i := 0; i < events.Len(); i++ {
		event := events.At(i)
		eventMap[event.Name()] = append(eventMap[event.Name()], event)
	}
	for _, eventList := range eventMap {
		sortEventsByTimestamp(eventList)
	}
	return eventMap
}

func sortEventsByTimestamp(eventList []pdata.SpanEvent) {
	sort.SliceStable(eventList, func(i, j int) bool { return eventList[i].Timestamp() < eventList[j].Timestamp() })
}

func convertLinksSliceToMap(links pdata.SpanLinkSlice) map[string]pdata.SpanLink {
	linkMap := make(map[string]pdata.SpanLink)
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		linkMap[traceIDAndSpanIDToString(link.TraceID(), link.SpanID())] = link
	}
	return linkMap
}

func notWithinOneMillisecond(sentNs pdata.Timestamp, recdNs pdata.Timestamp) bool {
	var diff pdata.Timestamp
	if sentNs > recdNs {
		diff = sentNs - recdNs
	} else {
		diff = recdNs - sentNs
	}
	return diff > 1100000
}

func populateSpansMap(spansMap map[string]pdata.Span, tds []pdata.Traces) {
	for _, td := range tds {
		rss := td.ResourceSpans()
		for i := 0; i < rss.Len(); i++ {
			ilss := rss.At(i).InstrumentationLibrarySpans()
			for j := 0; j < ilss.Len(); j++ {
				spans := ilss.At(j).Spans()
				for k := 0; k < spans.Len(); k++ {
					span := spans.At(k)
					key := traceIDAndSpanIDToString(span.TraceID(), span.SpanID())
					spansMap[key] = span
				}
			}
		}
	}
}

func traceIDAndSpanIDToString(traceID pdata.TraceID, spanID pdata.SpanID) string {
	return fmt.Sprintf("%s-%s", traceID.HexString(), spanID.HexString())
}
