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

package kafkareceiver

import (
	"go.opentelemetry.io/collector/translator/trace/zipkinv1"
	"go.opentelemetry.io/collector/translator/trace/zipkinv2"
)

const (
	zipkinProtobufEncoding = "zipkin_proto"
	zipkinJSONEncoding     = "zipkin_json"
	zipkinThriftEncoding   = "zipkin_thrift"
)

func newZipkinProtobufUnmarshaler() TracesUnmarshaler {
	return newPdataTracesUnmarshaler(zipkinv2.NewProtobufTracesUnmarshaler(false, false), zipkinProtobufEncoding)
}

func newZipkinJSONUnmarshaler() TracesUnmarshaler {
	return newPdataTracesUnmarshaler(zipkinv2.NewJSONTracesUnmarshaler(false), zipkinJSONEncoding)
}

func newZipkinThriftUnmarshaler() TracesUnmarshaler {
	return newPdataTracesUnmarshaler(zipkinv1.NewThriftTracesUnmarshaler(), zipkinThriftEncoding)
}
