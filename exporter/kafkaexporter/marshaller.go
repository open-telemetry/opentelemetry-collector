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

package kafkaexporter

import (
	"go.opentelemetry.io/collector/consumer/pdata"
)

// Marshaller marshals traces into Message array.
type Marshaller interface {
	// Marshal serializes spans into Messages
	Marshal(traces pdata.Traces) ([]Message, error)

	// Encoding returns encoding name
	Encoding() string
}

// Message encapsulates Kafka's message payload.
type Message struct {
	Value []byte
}

// defaultMarshallers returns map of supported encodings with Marshaller.
func defaultMarshallers() map[string]Marshaller {
	otlp := &otlpProtoMarshaller{}
	jaegerProto := jaegerMarshaller{marshaller: jaegerProtoSpanMarshaller{}}
	jaegerJSON := jaegerMarshaller{marshaller: newJaegerJSONMarshaller()}
	return map[string]Marshaller{
		otlp.Encoding():        otlp,
		jaegerProto.Encoding(): jaegerProto,
		jaegerJSON.Encoding():  jaegerJSON,
	}
}
