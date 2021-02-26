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

package goldendataset

import (
	"fmt"
	"io"
	"math/rand"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlptrace "go.opentelemetry.io/collector/internal/data/protogen/trace/v1"
)

// GenerateTraces generates a slice of OTLP ResourceSpans objects based on the PICT-generated pairwise
// parameters defined in the parameters file specified by the tracePairsFile parameter. The pairs to generate
// spans for for defined in the file specified by the spanPairsFile parameter.
// The slice of ResourceSpans are returned. If an err is returned, the slice elements will be nil.
func GenerateTraces(tracePairsFile string, spanPairsFile string) ([]pdata.Traces, error) {
	random := io.Reader(rand.New(rand.NewSource(42)))
	pairsData, err := loadPictOutputFile(tracePairsFile)
	if err != nil {
		return nil, err
	}
	pairsTotal := len(pairsData) - 1
	traces := make([]pdata.Traces, pairsTotal)
	for index, values := range pairsData {
		if index == 0 {
			continue
		}
		tracingInputs := &PICTTracingInputs{
			Resource:               PICTInputResource(values[TracesColumnResource]),
			InstrumentationLibrary: PICTInputInstrumentationLibrary(values[TracesColumnInstrumentationLibrary]),
			Spans:                  PICTInputSpans(values[TracesColumnSpans]),
		}
		rscSpan, spanErr := generateResourceSpan(tracingInputs, spanPairsFile, random)
		if spanErr != nil {
			err = spanErr
		}
		traces[index-1] = pdata.TracesFromOtlp([]*otlptrace.ResourceSpans{rscSpan})
	}
	return traces, err
}

// generateResourceSpan generates a single OTLP ResourceSpans populated based on the provided inputs. They are:
//   tracingInputs - the pairwise combination of field value variations for this ResourceSpans
//   spanPairsFile - the file with the PICT-generated parameter combinations to generate spans for
//   random - the random number generator to use in generating ID values
//
// The generated resource spans. If err is not nil, some or all of the resource spans fields will be nil.
func generateResourceSpan(tracingInputs *PICTTracingInputs, spanPairsFile string,
	random io.Reader) (*otlptrace.ResourceSpans, error) {
	libSpans, err := generateLibrarySpansArray(tracingInputs, spanPairsFile, random)
	return &otlptrace.ResourceSpans{
		Resource:                    GenerateResource(tracingInputs.Resource),
		InstrumentationLibrarySpans: libSpans,
	}, err
}

func generateLibrarySpansArray(tracingInputs *PICTTracingInputs, spanPairsFile string,
	random io.Reader) ([]*otlptrace.InstrumentationLibrarySpans, error) {
	var count int
	switch tracingInputs.InstrumentationLibrary {
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
		libSpans[i], err = generateLibrarySpans(tracingInputs, i, spanPairsFile, random)
	}
	return libSpans, err
}

func generateLibrarySpans(tracingInputs *PICTTracingInputs, index int, spanPairsFile string,
	random io.Reader) (*otlptrace.InstrumentationLibrarySpans, error) {
	spanCaseCount, err := countTotalSpanCases(spanPairsFile)
	if err != nil {
		return nil, err
	}
	var spans []*otlptrace.Span
	switch tracingInputs.Spans {
	case LibrarySpansNone:
		spans = make([]*otlptrace.Span, 0)
	case LibrarySpansOne:
		spans, _, err = GenerateSpans(1, 0, spanPairsFile, random)
	case LibrarySpansSeveral:
		spans, _, err = GenerateSpans(spanCaseCount/4, 0, spanPairsFile, random)
	case LibrarySpansAll:
		spans, _, err = GenerateSpans(spanCaseCount, 0, spanPairsFile, random)
	default:
		spans, _, err = GenerateSpans(16, 0, spanPairsFile, random)
	}
	return &otlptrace.InstrumentationLibrarySpans{
		InstrumentationLibrary: generateInstrumentationLibrary(tracingInputs, index),
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

func generateInstrumentationLibrary(tracingInputs *PICTTracingInputs, index int) otlpcommon.InstrumentationLibrary {
	if LibraryNone == tracingInputs.InstrumentationLibrary {
		return otlpcommon.InstrumentationLibrary{}
	}
	nameStr := fmt.Sprintf("%s-%s-%s-%d", tracingInputs.Resource, tracingInputs.InstrumentationLibrary,
		tracingInputs.Spans, index)
	verStr := "semver:1.1.7"
	if index > 0 {
		verStr = ""
	}
	return otlpcommon.InstrumentationLibrary{
		Name:    nameStr,
		Version: verStr,
	}
}
