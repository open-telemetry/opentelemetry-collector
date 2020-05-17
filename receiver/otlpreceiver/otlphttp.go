// Copyright The OpenTelemetry Authors
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

package otlpreceiver

import (
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// xProtobufMarshaler is a Marshaler which wraps runtime.ProtoMarshaller
// and sets ContentType to application/x-protobuf
type xProtobufMarshaler struct {
	marshaler runtime.ProtoMarshaller
}

// ContentType always returns "application/x-protobuf".
func (*xProtobufMarshaler) ContentType() string {
	return "application/x-protobuf"
}

// Marshal marshals "value" into Proto
func (m *xProtobufMarshaler) Marshal(value interface{}) ([]byte, error) {
	return m.marshaler.Marshal(value)
}

// Unmarshal unmarshals proto "data" into "value"
func (m *xProtobufMarshaler) Unmarshal(data []byte, value interface{}) error {
	return m.marshaler.Unmarshal(data, value)
}

// NewDecoder returns a Decoder which reads proto stream from "reader".
func (m *xProtobufMarshaler) NewDecoder(reader io.Reader) runtime.Decoder {
	return m.marshaler.NewDecoder(reader)
}

// NewEncoder returns an Encoder which writes proto stream into "writer".
func (m *xProtobufMarshaler) NewEncoder(writer io.Writer) runtime.Encoder {
	return m.marshaler.NewEncoder(writer)
}
