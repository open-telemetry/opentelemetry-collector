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

package zipkinv1

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

var _ pdata.ToTracesTranslator = (*toTranslator)(nil)

type toTranslator struct{}

// ToTraces converts converts traceData to pdata.Traces.
func (t toTranslator) ToTraces(src interface{}) (pdata.Traces, error) {
	ocTraces, ok := src.([]traceData)
	if !ok {
		return pdata.Traces{}, pdata.NewErrIncompatibleType([]traceData{}, src)
	}

	td := pdata.NewTraces()

	for _, trace := range ocTraces {
		tmp := internaldata.OCToTraces(trace.Node, trace.Resource, trace.Spans)
		tmp.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
	}

	return td, nil
}
