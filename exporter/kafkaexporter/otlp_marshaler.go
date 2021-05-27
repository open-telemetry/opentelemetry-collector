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

package kafkaexporter

import (
	"github.com/Shopify/sarama"

	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ TracesMarshaler = (*otlpTracesPbMarshaler)(nil)
var _ MetricsMarshaler = (*otlpMetricsPbMarshaler)(nil)

type otlpTracesPbMarshaler struct {
}

func (m *otlpTracesPbMarshaler) Encoding() string {
	return defaultEncoding
}

func (m *otlpTracesPbMarshaler) Marshal(td pdata.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := td.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []*sarama.ProducerMessage{
		{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		},
	}, nil
}

type otlpMetricsPbMarshaler struct {
}

func (m *otlpMetricsPbMarshaler) Encoding() string {
	return defaultEncoding
}

func (m *otlpMetricsPbMarshaler) Marshal(md pdata.Metrics, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := md.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []*sarama.ProducerMessage{
		{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		},
	}, nil
}

type otlpLogsPbMarshaler struct {
}

func (m *otlpLogsPbMarshaler) Encoding() string {
	return defaultEncoding
}

func (m *otlpLogsPbMarshaler) Marshal(ld pdata.Logs, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := ld.ToOtlpProtoBytes()
	if err != nil {
		return nil, err
	}
	return []*sarama.ProducerMessage{
		{
			Topic: topic,
			Value: sarama.ByteEncoder(bts),
		},
	}, nil
}
