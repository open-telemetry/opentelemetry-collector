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
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	jaegerproto "github.com/jaegertracing/jaeger/model"

	"go.opentelemetry.io/collector/consumer/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

// JaegerProtobuf protocol type.
const JaegerProtobuf Type = "jaeger/protobuf"

type jaegerProtoSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*jaegerProtoSpanUnmarshaler)(nil)

func (j jaegerProtoSpanUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	batch := jaegerproto.Batch{}
	err := batch.Unmarshal(bytes)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return jaegertranslator.ProtoBatchToInternalTraces(batch), nil
}

func (j jaegerProtoSpanUnmarshaler) Type() Type {
	return JaegerProtobuf
}

// JaegerJSON protocol type.
const JaegerJSON Type = "jaeger/json"

type jaegerJSONSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*jaegerJSONSpanUnmarshaler)(nil)

func (j jaegerJSONSpanUnmarshaler) Unmarshal(data []byte) (pdata.Traces, error) {
	batch := jaegerproto.Batch{}
	err := jsonpb.Unmarshal(bytes.NewReader(data), &batch)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return jaegertranslator.ProtoBatchToInternalTraces(batch), nil
}

func (j jaegerJSONSpanUnmarshaler) Type() Type {
	return JaegerJSON
}
