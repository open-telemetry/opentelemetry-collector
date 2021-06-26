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
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"

	"go.opentelemetry.io/collector/model/pdata"
)

var _ pdata.TracesDecoder = (*protobufDecoder)(nil)

type protobufDecoder struct {
	// DebugWasSet toggles the Debug field of each Span. It is usually set to true if
	// the "X-B3-Flags" header is set to 1 on the request.
	DebugWasSet bool
}

// DecodeTraces from protobuf bytes to zipkin model.
func (p protobufDecoder) DecodeTraces(buf []byte) (interface{}, error) {
	spans, err := zipkin_proto3.ParseSpans(buf, p.DebugWasSet)
	if err != nil {
		return nil, err
	}
	return spans, nil
}

var _ pdata.TracesEncoder = (*protobufEncoder)(nil)

type protobufEncoder struct {
	serializer zipkin_proto3.SpanSerializer
}

// EncodeTraces to protobuf bytes.
func (p protobufEncoder) EncodeTraces(mod interface{}) ([]byte, error) {
	spans, ok := mod.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, pdata.NewErrIncompatibleType([]*zipkinmodel.SpanModel{}, mod)
	}
	return p.serializer.Serialize(spans)
}

// NewProtobufTracesUnmarshaler returns an unmarshaler of protobuf bytes.
func NewProtobufTracesUnmarshaler(debugWasSet, parseStringTags bool) pdata.TracesUnmarshaler {
	return pdata.NewTracesUnmarshaler(
		protobufDecoder{DebugWasSet: debugWasSet},
		ToTranslator{ParseStringTags: parseStringTags},
	)
}

// NewProtobufTracesMarshaler returns a new marshaler to protobuf bytes.
func NewProtobufTracesMarshaler() pdata.TracesMarshaler {
	return pdata.NewTracesMarshaler(protobufEncoder{}, FromTranslator{})
}
