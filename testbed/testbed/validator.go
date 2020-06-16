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

package testbed

import (
	"encoding/hex"
	"log"
	"time"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"
	"go.opentelemetry.io/collector/translator/internaldata"
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
	assertionFailures []*AssertionFailure
}

func NewCorrectTestValidator(provider DataProvider) *CorrectnessTestValidator {
	return &CorrectnessTestValidator{
		dataProvider:      provider,
		assertionFailures: make([]*AssertionFailure, 0),
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
	if len(tc.MockBackend.ReceivedTracesOld) > 0 {
		tracesList := make([]pdata.Traces, 0, len(tc.MockBackend.ReceivedTracesOld))
		for _, td := range tc.MockBackend.ReceivedTracesOld {
			tracesList = append(tracesList, internaldata.OCToTraceData(td))
		}
		v.assertSentRecdTracingDataEqual(tracesList)
	}
	// TODO enable once identified problems are fixed
	//assert.EqualValues(tc.t, 0, len(v.assertionFailures), "There are span data mismatches.")
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
		testName:              testName,
		result:                result,
		duration:              time.Since(tc.startTime),
		receivedSpanCount:     tc.MockBackend.DataItemsReceived(),
		sentSpanCount:         tc.LoadGenerator.DataItemsSent(),
		assertionFailureCount: uint64(len(v.assertionFailures)),
		assertionFailures:     v.assertionFailures,
	})
}

func (v *CorrectnessTestValidator) assertSentRecdTracingDataEqual(tracesList []pdata.Traces) {
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

func (v *CorrectnessTestValidator) diffSpan(sentSpan *otlptrace.Span, recdSpan *otlptrace.Span) {
	if sentSpan == nil {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
		return
	}
	if hex.EncodeToString(sentSpan.TraceId) != hex.EncodeToString(recdSpan.TraceId) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceId",
			expectedValue: hex.EncodeToString(sentSpan.TraceId),
			actualValue:   hex.EncodeToString(recdSpan.TraceId),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if hex.EncodeToString(sentSpan.SpanId) != hex.EncodeToString(recdSpan.SpanId) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "SpanId",
			expectedValue: hex.EncodeToString(sentSpan.SpanId),
			actualValue:   hex.EncodeToString(recdSpan.SpanId),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.TraceState != recdSpan.TraceState {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "TraceState",
			expectedValue: sentSpan.TraceState,
			actualValue:   recdSpan.TraceState,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if hex.EncodeToString(sentSpan.ParentSpanId) != hex.EncodeToString(recdSpan.ParentSpanId) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "ParentSpanId",
			expectedValue: hex.EncodeToString(sentSpan.ParentSpanId),
			actualValue:   hex.EncodeToString(recdSpan.ParentSpanId),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.Name != recdSpan.Name {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Name",
			expectedValue: sentSpan.Name,
			actualValue:   recdSpan.Name,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.Kind != recdSpan.Kind {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Kind",
			expectedValue: sentSpan.Kind,
			actualValue:   recdSpan.Kind,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.StartTimeUnixNano != recdSpan.StartTimeUnixNano {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "StartTimeUnixNano",
			expectedValue: sentSpan.StartTimeUnixNano,
			actualValue:   recdSpan.StartTimeUnixNano,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.EndTimeUnixNano != recdSpan.EndTimeUnixNano {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "StartTimeUnixNano",
			expectedValue: sentSpan.EndTimeUnixNano,
			actualValue:   recdSpan.EndTimeUnixNano,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if len(sentSpan.Attributes) != len(recdSpan.Attributes) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Attributes",
			expectedValue: len(sentSpan.Attributes),
			actualValue:   len(recdSpan.Attributes),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	//TODO compare keys and values of attributes
	if sentSpan.DroppedAttributesCount != recdSpan.DroppedAttributesCount {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedAttributesCount",
			expectedValue: sentSpan.DroppedAttributesCount,
			actualValue:   recdSpan.DroppedAttributesCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if len(sentSpan.Events) != len(recdSpan.Events) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Events",
			expectedValue: len(sentSpan.Events),
			actualValue:   len(recdSpan.Events),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	//TODO compare contents of events
	if sentSpan.DroppedEventsCount != recdSpan.DroppedEventsCount {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedEventsCount",
			expectedValue: sentSpan.DroppedEventsCount,
			actualValue:   recdSpan.DroppedEventsCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if len(sentSpan.Links) != len(recdSpan.Links) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Links",
			expectedValue: len(sentSpan.Links),
			actualValue:   len(recdSpan.Links),
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	//TODO compare contents of links
	if sentSpan.DroppedLinksCount != recdSpan.DroppedLinksCount {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "DroppedLinksCount",
			expectedValue: sentSpan.DroppedLinksCount,
			actualValue:   recdSpan.DroppedLinksCount,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
	if sentSpan.Status != nil && recdSpan.Status != nil {
		if sentSpan.Status.Code != recdSpan.Status.Code {
			af := &AssertionFailure{
				typeName:      "Span",
				dataComboName: sentSpan.Name,
				fieldPath:     "Status.Code",
				expectedValue: sentSpan.Status.Code,
				actualValue:   recdSpan.Status.Code,
			}
			v.assertionFailures = append(v.assertionFailures, af)
		}
	} else if (sentSpan.Status != nil && recdSpan.Status == nil) || (sentSpan.Status == nil && recdSpan.Status != nil) {
		af := &AssertionFailure{
			typeName:      "Span",
			dataComboName: sentSpan.Name,
			fieldPath:     "Status",
			expectedValue: sentSpan.Status,
			actualValue:   recdSpan.Status,
		}
		v.assertionFailures = append(v.assertionFailures, af)
	}
}
