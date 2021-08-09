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

package zipkinv2

import (
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"

	"go.opentelemetry.io/collector/model/pdata"
)

type marshaler struct {
	serializer     zipkinreporter.SpanSerializer
	fromTranslator FromTranslator
}

// MarshalTraces to JSON bytes.
func (j marshaler) MarshalTraces(td pdata.Traces) ([]byte, error) {
	spans, err := j.fromTranslator.FromTraces(td)
	if err != nil {
		return nil, err
	}
	return j.serializer.Serialize(spans)
}
