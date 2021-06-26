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

var _ pdata.TracesDecoder = (*jsonDecoder)(nil)

type jsonDecoder struct{}

// DecodeTraces from JSON bytes to zipkin model.
func (j jsonDecoder) DecodeTraces(buf []byte) (interface{}, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(buf, &spans); err != nil {
		return nil, err
	}
	return spans, nil
}

var _ pdata.TracesEncoder = (*jsonEncoder)(nil)

type jsonEncoder struct {
	serializer zipkinreporter.JSONSerializer
}

// EncodeTraces from zipkin model to bytes.
func (j jsonEncoder) EncodeTraces(mod interface{}) ([]byte, error) {
	spans, ok := mod.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, pdata.NewErrIncompatibleType([]*zipkinmodel.SpanModel{}, mod)
	}
	return j.serializer.Serialize(spans)
}

// NewJSONTracesUnmarshaler returns an unmarshaler for JSON bytes.
func NewJSONTracesUnmarshaler(parseStringTags bool) pdata.TracesUnmarshaler {
	return pdata.NewTracesUnmarshaler(jsonDecoder{}, ToTranslator{ParseStringTags: parseStringTags})
}

// NewJSONTracesMarshaler returns a marshaler to JSON bytes.
func NewJSONTracesMarshaler() pdata.TracesMarshaler {
	return pdata.NewTracesMarshaler(jsonEncoder{}, FromTranslator{})
}
