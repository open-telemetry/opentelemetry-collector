// Copyright  The OpenTelemetry Authors
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

package kafkaexporter

import (
	"github.com/Shopify/sarama"
)

// MessageConverter converts our message type to sarama message type
type MessageConverter interface {
	// ConvertMessages generate Messages ready to be sent
	ConvertMessages([]Message, string) []*sarama.ProducerMessage
}

// OTLPProtoConverter is the converter for otlp_proto encoding
type OTLPProtoConverter struct {
}

// ConvertMessages converter for otlp_proto
func (mp *OTLPProtoConverter) ConvertMessages(messages []Message, topic string) []*sarama.ProducerMessage {
	producerMessages := make([]*sarama.ProducerMessage, len(messages))
	for i := range messages {
		producerMessages[i] = &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(messages[i].Value),
		}
	}
	return producerMessages
}

// JaegerConverter is the converter for jaeger_proto and jaeger_json encoding
type JaegerConverter struct {
}

// ConvertMessages converter for jaeger-ish encoding
func (mp *JaegerConverter) ConvertMessages(messages []Message, topic string) []*sarama.ProducerMessage {
	producerMessages := make([]*sarama.ProducerMessage, len(messages))
	for i := range messages {
		producerMessages[i] = &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(messages[i].Value),
			Key:   sarama.ByteEncoder(messages[i].Key),
		}
	}
	return producerMessages
}

// getConverters returns all pre-configured converters for supported encodings. For OTLP proto we use the encoding
// of traces marshaler. If future change make the encoding name different between traces/metrics/logs, then this
// may needs to adjust accordingly.
func getConverters() map[string]MessageConverter {
	otlppb := &otlpTracesPbMarshaler{}
	jaegerProto := &jaegerMarshaler{marshaler: jaegerProtoSpanMarshaler{}}
	jaegerJSON := &jaegerMarshaler{marshaler: newJaegerJSONMarshaler()}

	return map[string]MessageConverter{
		otlppb.Encoding():      &OTLPProtoConverter{},
		jaegerProto.Encoding(): &JaegerConverter{},
		jaegerJSON.Encoding():  &JaegerConverter{},
	}
}
