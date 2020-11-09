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
	"encoding/json"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	zipkinmodel "github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/proto/zipkin_proto3"

	"go.opentelemetry.io/collector/consumer/pdata"
	zipkintranslator "go.opentelemetry.io/collector/translator/trace/zipkin"
)

type zipkinProtoSpanUnmarshaller struct {
}

var _ Unmarshaller = (*zipkinProtoSpanUnmarshaller)(nil)

func (z zipkinProtoSpanUnmarshaller) Unmarshal(bytes []byte) (pdata.Traces, error) {
	parseSpans, err := zipkin_proto3.ParseSpans(bytes, false)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return zipkintranslator.V2SpansToInternalTraces(parseSpans, false)
}

func (z zipkinProtoSpanUnmarshaller) Encoding() string {
	return "zipkin_proto"
}

type zipkinJSONSpanUnmarshaller struct {
}

var _ Unmarshaller = (*zipkinJSONSpanUnmarshaller)(nil)

func (z zipkinJSONSpanUnmarshaller) Unmarshal(bytes []byte) (pdata.Traces, error) {
	var spans []*zipkinmodel.SpanModel
	if err := json.Unmarshal(bytes, &spans); err != nil {
		return pdata.NewTraces(), err
	}
	return zipkintranslator.V2SpansToInternalTraces(spans, false)
}

func (z zipkinJSONSpanUnmarshaller) Encoding() string {
	return "zipkin_json"
}

type zipkinThriftSpanUnmarshaller struct {
}

var _ Unmarshaller = (*zipkinThriftSpanUnmarshaller)(nil)

func (z zipkinThriftSpanUnmarshaller) Unmarshal(bytes []byte) (pdata.Traces, error) {
	spans, err := deserializeZipkinThrift(bytes)
	if err != nil {
		return pdata.NewTraces(), err
	}
	return zipkintranslator.V1ThriftBatchToInternalTraces(spans)

}

func (z zipkinThriftSpanUnmarshaller) Encoding() string {
	return "zipkin_thrift"
}

// deserializeThrift decodes Thrift bytes to a list of spans.
// This code comes from jaegertracing/jaeger, ideally we should have imported
// it but this was creating many conflicts so brought the code to here.
// https://github.com/jaegertracing/jaeger/blob/6bc0c122bfca8e737a747826ae60a22a306d7019/model/converter/thrift/zipkin/deserialize.go#L36
func deserializeZipkinThrift(b []byte) ([]*zipkincore.Span, error) {
	buffer := thrift.NewTMemoryBuffer()
	buffer.Write(b)

	transport := thrift.NewTBinaryProtocolTransport(buffer)
	_, size, err := transport.ReadListBegin() // Ignore the returned element type
	if err != nil {
		return nil, err
	}

	// We don't depend on the size returned by ReadListBegin to preallocate the array because it
	// sometimes returns a nil error on bad input and provides an unreasonably large int for size
	var spans []*zipkincore.Span
	for i := 0; i < size; i++ {
		zs := &zipkincore.Span{}
		if err = zs.Read(transport); err != nil {
			return nil, err
		}
		spans = append(spans, zs)
	}
	return spans, nil
}
