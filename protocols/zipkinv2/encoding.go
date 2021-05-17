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

	"go.opentelemetry.io/collector/protocols/encoding"
	"go.opentelemetry.io/collector/protocols/models"
)

var (
	_ encoding.TracesEncoder = (*Encoder)(nil)
	_ encoding.TracesDecoder = (*Encoder)(nil)
)

type Encoder struct {
	// Encoding is the format Zipkin is serialized to.
	Encoding encoding.Type
}

func (z *Encoder) DecodeTraces(bytes []byte) (interface{}, error) {
	switch z.Encoding {
	case encoding.Protobuf:
		return zipkin_proto3.ParseSpans(bytes, false)
	case encoding.JSON:
		var spans []*zipkinmodel.SpanModel
		if err := json.Unmarshal(bytes, &spans); err != nil {
			return nil, err
		}
		return spans, nil
	default:
		return nil, &encoding.ErrUnavailableEncoding{Encoding: z.Encoding}
	}
}

func (z *Encoder) EncodeTraces(model interface{}) ([]byte, error) {
	spans, ok := model.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, &models.ErrIncompatibleType{Model: spans}
	}

	switch z.Encoding {
	// TODO
	// case protocols.Protobuf:
	case encoding.JSON:
		return json.Marshal(spans)
	default:
		return nil, &encoding.ErrUnavailableEncoding{Encoding: z.Encoding}
	}
}
