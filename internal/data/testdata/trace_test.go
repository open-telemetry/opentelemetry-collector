// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
	"github.com/stretchr/testify/assert"
)

type traceTestCase struct {
	name string
	td   data.TraceData
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
			name: "no-spans",
			td:   GenerateTraceDataNoSpans(),
			otlp: generateTraceOtlpNoSpans(),
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
			otlp: generateTraceOtlpSameResourceTwoSpans(),
		},
		{
			name: "two-spans-same-resource-one-different",
			td:   GenerateTraceDataTwoSpansSameResourceOneDifferent(),
			otlp: generateTraceOtlpTwoSpansSameResourceOneDifferent(),
		},
	}
}

func TestGenerateTraceData(t *testing.T) {
	assert.EqualValues(t, GenerateTraceDataOneEmptyResourceSpans(), GenerateTraceDataOneEmptyResourceSpans())
	assert.EqualValues(t, GenerateTraceDataNoLibraries(), GenerateTraceDataNoLibraries())
	assert.EqualValues(t, GenerateTraceDataNoSpans(), GenerateTraceDataNoSpans())
	assert.EqualValues(t, GenerateTraceDataOneSpanNoResource(), GenerateTraceDataOneSpanNoResource())
	assert.EqualValues(t, GenerateTraceDataOneSpan(), GenerateTraceDataOneSpan())
	assert.EqualValues(t, GenerateTraceDataTwoSpansSameResource(), GenerateTraceDataTwoSpansSameResource())
	assert.EqualValues(t, GenerateTraceDataTwoSpansSameResourceOneDifferent(), GenerateTraceDataTwoSpansSameResourceOneDifferent())
}

func TestGeneratedTraceOtlp(t *testing.T) {
	assert.EqualValues(t, generateTraceOtlpNoLibraries(), generateTraceOtlpNoLibraries())
	assert.EqualValues(t, generateTraceOtlpNoSpans(), generateTraceOtlpNoSpans())
	assert.EqualValues(t, generateTraceOtlpOneSpanNoResource(), generateTraceOtlpOneSpanNoResource())
	assert.EqualValues(t, generateTraceOtlpOneSpan(), generateTraceOtlpOneSpan())
	assert.EqualValues(t, generateTraceOtlpSameResourceTwoSpans(), generateTraceOtlpSameResourceTwoSpans())
	assert.EqualValues(t, generateTraceOtlpTwoSpansSameResourceOneDifferent(), generateTraceOtlpTwoSpansSameResourceOneDifferent())
}

func TestToFromOtlp(t *testing.T) {
	allTestCases := generateAllTraceTestCases()
	// Ensure NumTraceTests gets updated.
	assert.EqualValues(t, NumTraceTests, len(allTestCases))
	for i := range allTestCases {
		test := allTestCases[i]
		t.Run(test.name, func(t *testing.T) {
			td := data.TraceDataFromOtlp(test.otlp)
			assert.EqualValues(t, test.td, td)
			otlp := data.TraceDataToOtlp(td)
			assert.EqualValues(t, test.otlp, otlp)
		})
	}
}
