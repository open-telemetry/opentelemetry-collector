// Copyright 2018, OpenCensus Authors
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

package processor

import (
	"fmt"
	"strings"

	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
)

// MultiSpanProcessor enables processing on multiple processors.
// For each incoming span batch, it calls ProcessSpans method on each span
// processor one-by-one. It aggregates success/failures/errors from all of
// them and reports the result upstream.
type MultiSpanProcessor []SpanProcessor

var _ SpanProcessor = (*MultiSpanProcessor)(nil)

// NewMultiSpanProcessor creates a MultiSpanProcessor from the variadic
// list of passed SpanProcessors.
func NewMultiSpanProcessor(procs ...SpanProcessor) MultiSpanProcessor {
	return procs
}

// ProcessSpans implements the SpanProcessor interface
func (msp MultiSpanProcessor) ProcessSpans(batch *agenttracepb.ExportTraceServiceRequest, spanFormat string) (uint64, error) {
	var maxFailures uint64
	var errors []error
	for _, sp := range msp {
		failures, err := sp.ProcessSpans(batch, spanFormat)
		if err != nil {
			errors = append(errors, err)
		}

		if failures > maxFailures {
			maxFailures = failures
		}
	}

	var err error
	numErrors := len(errors)
	if numErrors == 1 {
		err = errors[0]
	} else if numErrors > 1 {
		errMsgs := make([]string, numErrors)
		for _, err := range errors {
			errMsgs = append(errMsgs, err.Error())
		}
		err = fmt.Errorf("[%s]", strings.Join(errMsgs, "; "))
	}

	return maxFailures, err
}
