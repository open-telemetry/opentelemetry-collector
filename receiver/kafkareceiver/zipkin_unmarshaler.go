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
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/trace/zipkinv1"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
)

type zipkinProtoSpanUnmarshaler struct {
}

var (
	_                         TracesUnmarshaler = (*zipkinProtoSpanUnmarshaler)(nil)
	v1ThriftUnmarshaler                         = zipkinv1.NewThriftTracesUnmarshaler()
	zipkinProtobufUnmarshaler                   = zipkinv2.NewProtobufTracesUnmarshaler(false, false)
	zipkinJSONUnmarshaler                       = zipkinv2.NewJSONTracesUnmarshaler(false)
)

func (z zipkinProtoSpanUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	return zipkinProtobufUnmarshaler.Unmarshal(bytes)
}

func (z zipkinProtoSpanUnmarshaler) Encoding() string {
	return "zipkin_proto"
}

type zipkinJSONSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*zipkinJSONSpanUnmarshaler)(nil)

func (z zipkinJSONSpanUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	return zipkinJSONUnmarshaler.Unmarshal(bytes)
}

func (z zipkinJSONSpanUnmarshaler) Encoding() string {
	return "zipkin_json"
}

type zipkinThriftSpanUnmarshaler struct {
}

var _ TracesUnmarshaler = (*zipkinThriftSpanUnmarshaler)(nil)

func (z zipkinThriftSpanUnmarshaler) Unmarshal(bytes []byte) (pdata.Traces, error) {
	return v1ThriftUnmarshaler.Unmarshal(bytes)
}

func (z zipkinThriftSpanUnmarshaler) Encoding() string {
	return "zipkin_thrift"
}
