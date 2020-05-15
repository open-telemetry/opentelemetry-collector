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

package goldendataset

import (
	"fmt"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"
)

func GenerateResourceSpans(tracePairsFile string, spanPairsFile string) ([]*otlptrace.ResourceSpans, error) {
	pairsData, err := loadPictOutputFile(tracePairsFile)
	if err != nil {
		return nil, err
	}
	pairsTotal := len(pairsData) - 1
	spans := make([]*otlptrace.ResourceSpans, pairsTotal)
	for index, values := range pairsData {
		if index == 0 {
			continue
		}
		rscSpan, spanErr := GenerateResourceSpan(PICTInputResource(values[TracesColumnResource]),
			PICTInputInstrumentationLibrary(values[TracesColumnInstrumentationLibrary]),
			PICTInputSpans(values[TracesColumnSpans]), spanPairsFile)
		if spanErr != nil {
			err = spanErr
		}
		spans[index-1] = rscSpan
	}
	return spans, err
}

func GenerateResourceSpan(rscID PICTInputResource, instrLib PICTInputInstrumentationLibrary, spansSize PICTInputSpans,
	spanPairsFile string) (*otlptrace.ResourceSpans, error) {
	libSpans, err := generateLibrarySpansArray(instrLib, spansSize, spanPairsFile)
	return &otlptrace.ResourceSpans{
		Resource:                    GenerateResource(rscID),
		InstrumentationLibrarySpans: libSpans,
	}, err
}

func generateLibrarySpansArray(instrLib PICTInputInstrumentationLibrary, spansSize PICTInputSpans,
	spanPairsFile string) ([]*otlptrace.InstrumentationLibrarySpans, error) {
	var count int
	switch instrLib {
	case LibraryNone:
		count = 1
	case LibraryOne:
		count = 1
	case LibraryTwo:
		count = 2
	}
	var err error
	libSpans := make([]*otlptrace.InstrumentationLibrarySpans, count)
	for i := 0; i < count; i++ {
		libSpans[i], err = generateLibrarySpans(instrLib, spansSize, i, spanPairsFile)
	}
	return libSpans, err
}

func generateLibrarySpans(instrLib PICTInputInstrumentationLibrary, spansSize PICTInputSpans, index int,
	spanPairsFile string) (*otlptrace.InstrumentationLibrarySpans, error) {
	spanCaseCount, err := countTotalSpanCases(spanPairsFile)
	if err != nil {
		return nil, err
	}
	var spans []*otlptrace.Span
	switch spansSize {
	case LibrarySpansNone:
		spans = make([]*otlptrace.Span, 0)
	case LibrarySpansOne:
		spans, _, err = GenerateSpans(1, 0, spanPairsFile)
	case LibrarySpansSeveral:
		spans, _, err = GenerateSpans(spanCaseCount/4, 0, spanPairsFile)
	case LibrarySpansAll:
		spans, _, err = GenerateSpans(spanCaseCount, 0, spanPairsFile)
	default:
		spans, _, err = GenerateSpans(16, 0, spanPairsFile)
	}
	return &otlptrace.InstrumentationLibrarySpans{
		InstrumentationLibrary: generateInstrumentationLibrary(instrLib, index),
		Spans:                  spans,
	}, err
}

func countTotalSpanCases(spanPairsFile string) (int, error) {
	pairsData, err := loadPictOutputFile(spanPairsFile)
	if err != nil {
		return 0, err
	}
	count := len(pairsData) - 1
	return count, err
}

func generateInstrumentationLibrary(instrLib PICTInputInstrumentationLibrary,
	index int) *otlpcommon.InstrumentationLibrary {
	if LibraryNone == instrLib {
		return nil
	}
	nameStr := fmt.Sprintf("OtelTestLib-%d", index)
	verStr := "semver:1.1.7"
	if index > 0 {
		verStr = ""
	}
	return &otlpcommon.InstrumentationLibrary{
		Name:    nameStr,
		Version: verStr,
	}
}
