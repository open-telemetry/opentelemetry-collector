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
	"errors"

	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"

	"go.opentelemetry.io/collector/internal/model/serializer"
	"go.opentelemetry.io/collector/internal/model/translator"
)

var (
	_ serializer.TracesMarshaler   = (*Protobuf)(nil)
	_ serializer.TracesUnmarshaler = (*Protobuf)(nil)
	_ serializer.TracesMarshaler   = (*JSON)(nil)
	_ serializer.TracesUnmarshaler = (*JSON)(nil)
)

type Protobuf struct {
}

type JSON struct {
}

func (z *Protobuf) UnmarshalTraces(data []byte) (interface{}, error) {
	return zipkin_proto3.ParseSpans(data, false)
}

func (z *Protobuf) MarshalTraces(model interface{}) ([]byte, error) {
	spans, ok := model.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, translator.NewErrIncompatibleType(spans, model)
	}

	// TODO: need to extract code from zipkinreceiver
	return nil, errors.New("Unimplemented")
}

func (z *JSON) UnmarshalTraces(data []byte) (interface{}, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(data, &spans); err != nil {
		return nil, err
	}
	return spans, nil
}

func (z *JSON) MarshalTraces(model interface{}) ([]byte, error) {
	spans, ok := model.([]*zipkinmodel.SpanModel)
	if !ok {
		return nil, translator.NewErrIncompatibleType(spans, model)
	}

	return json.Marshal(spans)
}
