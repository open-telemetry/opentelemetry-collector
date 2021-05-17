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

package protocols

import (
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/protocols/encoding"
	"go.opentelemetry.io/collector/protocols/models"
	"go.opentelemetry.io/collector/protocols/zipkinv2"
)

type Transcoder struct {
	models.TracesModelTranslator
	encoding.TracesEncoder
	encoding.TracesDecoder
}

func (t *Transcoder) UnmarshalTraces(data []byte) (pdata.Traces, error) {
	model, err := t.DecodeTraces(data)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return t.TracesFromModel(model)
}

func (t *Transcoder) MarshalTraces(td pdata.Traces) ([]byte, error) {
	out := t.Type()
	if err := t.TracesToModel(td, &out); err != nil {
		return nil, err
	}
	return t.EncodeTraces(out)
}

func test() {
	translator := &zipkinv2.Model{ParseStringTags: false}
	encodingDecoding := &zipkinv2.Encoder{
		Encoding: encoding.JSON,
	}

	tc := &Transcoder{
		TracesModelTranslator: translator,
		TracesEncoder:         encodingDecoding,
		TracesDecoder:         encodingDecoding,
	}
}
