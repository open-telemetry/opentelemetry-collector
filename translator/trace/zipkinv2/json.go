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
	"encoding/json"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	zipkinreporter "github.com/openzipkin/zipkin-go/reporter"

	"go.opentelemetry.io/collector/model/pdata"
)

type jsonUnmarshaler struct {
	toTranslator ToTranslator
}

// UnmarshalTraces from JSON bytes.
func (j jsonUnmarshaler) UnmarshalTraces(buf []byte) (pdata.Traces, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(buf, &spans); err != nil {
		return pdata.Traces{}, err
	}
	return j.toTranslator.ToTraces(spans)
}

type jsonMarshaler struct {
	serializer     zipkinreporter.JSONSerializer
	fromTranslator FromTranslator
}

// MarshalTraces to JSON bytes.
func (j jsonMarshaler) MarshalTraces(td pdata.Traces) ([]byte, error) {
	spans, err := j.fromTranslator.FromTraces(td)
	if err != nil {
		return nil, err
	}
	return j.serializer.Serialize(spans)
}

// NewJSONTracesUnmarshaler returns an unmarshaler for JSON bytes.
func NewJSONTracesUnmarshaler(parseStringTags bool) pdata.TracesUnmarshaler {
	return jsonUnmarshaler{toTranslator: ToTranslator{ParseStringTags: parseStringTags}}
}

// NewJSONTracesMarshaler returns a marshaler to JSON bytes.
func NewJSONTracesMarshaler() pdata.TracesMarshaler {
	return jsonMarshaler{}
}
