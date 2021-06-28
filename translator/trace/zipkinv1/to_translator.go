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
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func toTraces(ocTraces []traceData) (pdata.Traces, error) {
	td := pdata.NewTraces()

	for _, trace := range ocTraces {
		tmp := internaldata.OCToTraces(trace.Node, trace.Resource, trace.Spans)
		tmp.ResourceSpans().MoveAndAppendTo(td.ResourceSpans())
	}

	return td, nil
}
