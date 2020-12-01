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

package attributesprocessor

import (
	"context"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type spanAttributesProcessor struct {
	attrProc *processorhelper.AttrProc
	include  filterspan.Matcher
	exclude  filterspan.Matcher
}

// newTraceProcessor returns a processor that modifies attributes of a span.
// To construct the attributes processors, the use of the factory methods are required
// in order to validate the inputs.
func newSpanAttributesProcessor(attrProc *processorhelper.AttrProc, include, exclude filterspan.Matcher) *spanAttributesProcessor {
	return &spanAttributesProcessor{
		attrProc: attrProc,
		include:  include,
		exclude:  exclude,
	}
}

// ProcessTraces implements the TProcessor
func (a *spanAttributesProcessor) ProcessTraces(_ context.Context, td pdata.Traces) (pdata.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		resource := rs.Resource()
		ilss := rs.InstrumentationLibrarySpans()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			spans := ils.Spans()
			library := ils.InstrumentationLibrary()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				if filterspan.SkipSpan(a.include, a.exclude, span, resource, library) {
					continue
				}

				a.attrProc.Process(span.Attributes())
			}
		}
	}
	return td, nil
}
