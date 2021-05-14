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

	// Encoding return encoding for this converter
	Encoding() string
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

// Encoding return encoding for otlp_proto
func (mp *OTLPProtoConverter) Encoding() string {
	return "otlp_proto"
}

// JaegerProtoConverter is the converter for jaeger_proto and jaeger_json encoding
type JaegerProtoConverter struct {
}

// JaegerJSONConverter is the converter for jaeger_proto and jaeger_json encoding
type JaegerJSONConverter struct {
}

// ConvertMessages converter for jaeger_proto encoding
func (mp *JaegerProtoConverter) ConvertMessages(messages []Message, topic string) []*sarama.ProducerMessage {
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

// Encoding return encoding for jaeger_proto
func (mp *JaegerProtoConverter) Encoding() string {
	return "jaeger_proto"
}

// ConvertMessages converter for jaeger_json encoding
func (mp *JaegerJSONConverter) ConvertMessages(messages []Message, topic string) []*sarama.ProducerMessage {
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

// Encoding return encoding for jaeger_proto
func (mp *JaegerJSONConverter) Encoding() string {
	return "jaeger_json"
}

// getConverters returns all pre-configured converters for supported encodings
func getConverters() map[string]MessageConverter {
	otlppb := &OTLPProtoConverter{}
	jaegerProto := &JaegerProtoConverter{}
	jaegerJSON := &JaegerJSONConverter{}

	return map[string]MessageConverter{
		otlppb.Encoding():      otlppb,
		jaegerProto.Encoding(): jaegerProto,
		jaegerJSON.Encoding():  jaegerJSON,
	}
}
