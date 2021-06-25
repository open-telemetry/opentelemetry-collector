// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkareceiver

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	jaegerproto "github.com/jaegertracing/jaeger/model"

	"go.opentelemetry.io/collector/model/pdata"
	jaegertranslator "go.opentelemetry.io/collector/translator/trace/jaeger"
)

type jaegerProtoSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*jaegerProtoSpanUnmarshaler)(nil)

func (j jaegerProtoSpanUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	span := &jaegerproto.Span{}
	err := span.Unmarshal(bytes)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return jaegerSpanToTraces(span), nil
}

func (j jaegerProtoSpanUnmarshaler) Encoding() string {
	return "jaeger_proto"
}

type jaegerJSONSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*jaegerJSONSpanUnmarshaler)(nil)

func (j jaegerJSONSpanUnmarshaler) Unmarshal(data []byte) (pdata.Traces, error) {
	span := &jaegerproto.Span{}
	err := jsonpb.Unmarshal(bytes.NewReader(data), span)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return jaegerSpanToTraces(span), nil
}

func (j jaegerJSONSpanUnmarshaler) Encoding() string {
	return "jaeger_json"
}

func jaegerSpanToTraces(span *jaegerproto.Span) pdata.Traces {
	batch := jaegerproto.Batch{
		Spans:   []*jaegerproto.Span{span},
		Process: span.Process,
	}
	return jaegertranslator.ProtoBatchToInternalTraces(batch)
}
