// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"testing"

	"github.com/stretchr/testify/assert"

	otlptrace "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/trace/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type traceTestCase struct {
	name string
	td   pdata.Traces
	otlp []*otlptrace.ResourceSpans
}

func generateAllTraceTestCases() []traceTestCase {
	return []traceTestCase{
		{
			name: "empty",
			td:   GenerateTraceDataEmpty(),
			otlp: generateTraceOtlpEmpty(),
		},
		{
			name: "one-empty-resource-spans",
			td:   GenerateTraceDataOneEmptyResourceSpans(),
			otlp: generateTraceOtlpOneEmptyResourceSpans(),
		},
		{
			name: "no-libraries",
			td:   GenerateTraceDataNoLibraries(),
			otlp: generateTraceOtlpNoLibraries(),
		},
		{
			name: "one-empty-instrumentation-library",
			td:   GenerateTraceDataOneEmptyInstrumentationLibrary(),
			otlp: generateTraceOtlpOneEmptyInstrumentationLibrary(),
		},
		{
			name: "one-span-no-resource",
			td:   GenerateTraceDataOneSpanNoResource(),
			otlp: generateTraceOtlpOneSpanNoResource(),
		},
		{
			name: "one-span",
			td:   GenerateTraceDataOneSpan(),
			otlp: generateTraceOtlpOneSpan(),
		},
		{
			name: "two-spans-same-resource",
			td:   GenerateTraceDataTwoSpansSameResource(),
			otlp: GenerateTraceOtlpSameResourceTwoSpans(),
		},
		{
			name: "two-spans-same-resource-one-different",
			td:   GenerateTraceDataTwoSpansSameResourceOneDifferent(),
			otlp: generateTraceOtlpTwoSpansSameResourceOneDifferent(),
		},
	}
}

func TestToFromOtlpTrace(t *testing.T) {
	allTestCases := generateAllTraceTestCases()
	// Ensure NumTraceTests gets updated.
	for i := range allTestCases {
		test := allTestCases[i]
		t.Run(test.name, func(t *testing.T) {
			td := pdata.TracesFromOtlp(test.otlp)
			assert.EqualValues(t, test.td, td)
			otlp := pdata.TracesToOtlp(td)
			assert.EqualValues(t, test.otlp, otlp)
		})
	}
}
