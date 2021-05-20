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
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"

	"go.opentelemetry.io/collector/internal/model/serializer"
	"go.opentelemetry.io/collector/internal/model/translator"
)

var (
	_ serializer.TracesMarshaler   = (*Encoder)(nil)
	_ serializer.TracesUnmarshaler = (*Encoder)(nil)
)

type Encoder struct {
	// Encoding is the format Zipkin is serialized to.
	Encoding serializer.Encoding
}

func (z *Encoder) UnmarshalTraces(data []byte) (interface{}, error) {
	switch z.Encoding {
	case serializer.Protobuf:
		return zipkin_proto3.ParseSpans(data, false)
	case serializer.JSON:
		var spans []*zipkinmodel.SpanModel
		if err := json.Unmarshal(data, &spans); err != nil {
			return nil, err
		}
		return spans, nil
	default:
		return nil, serializer.NewErrUnavailableEncoding(z.Encoding)
	}
}

func (z *Encoder) MarshalTraces(model interface{}) ([]byte, error) {
	spans, ok := model.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, translator.NewErrIncompatibleType(spans, model)
	}

	switch z.Encoding {
	// TODO: need to extract code from zipkinreceiver
	// case protocols.Protobuf:
	case serializer.JSON:
		return json.Marshal(spans)
	default:
		return nil, serializer.NewErrUnavailableEncoding(z.Encoding)
	}
}
