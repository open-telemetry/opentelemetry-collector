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

package zipkin

import (
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
)

func V1ThriftBatchToInternalTraces(zSpans []*zipkincore.Span) (pdata.Traces, error) {
	traces := pdata.NewTraces()

	ocTraces, _ := v1ThriftBatchToOCProto(zSpans)

	for _, td := range ocTraces {
		tmp := internaldata.OCToTraceData(td)
		tmp.ResourceSpans().MoveAndAppendTo(traces.ResourceSpans())
	}
	return traces, nil
}
