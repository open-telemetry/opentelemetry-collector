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

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/protocols/encodings"
	"go.opentelemetry.io/collector/protocols/models"
)

var (
	_ encodings.TracesEncoder = (*Encoder)(nil)
	_ encodings.TracesDecoder = (*Encoder)(nil)
)

type Encoder struct {
	models.TracesModelTranslator
	// Encoding is the format Zipkin is serialized to.
	Encoding encodings.Encoding
}

func (z *Encoder) UnmarshalTraces(bytes []byte) (pdata.Traces, error) {
	switch z.Encoding {
	case encodings.Protobuf:
		spans, err := zipkin_proto3.ParseSpans(bytes, false)
		if err != nil {
			return pdata.NewTraces(), err
		}
		return z.TracesFromModel(spans)
	case encodings.JSON:
		var spans []*zipkinmodel.SpanModel
		if err := json.Unmarshal(bytes, &spans); err != nil {
			return pdata.NewTraces(), err
		}
		return z.TracesFromModel(spans)
	default:
		return pdata.NewTraces(), &encodings.ErrInvalidEncoding{Encoding: z.Encoding}
	}
}

func (z *Encoder) MarshalTraces(td pdata.Traces) ([]byte, error) {
	switch z.Encoding {
	// TODO
	// case protocols.Protobuf:
	case encodings.JSON:
		var spans []*zipkinmodel.SpanModel
		if err := z.TracesToModel(td, &spans); err != nil {
			return nil, err
		}
		return json.Marshal(spans)
	default:
		return nil, &encodings.ErrInvalidEncoding{Encoding: z.Encoding}
	}
}
