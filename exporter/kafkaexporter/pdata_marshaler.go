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
	"github.com/Shopify/sarama"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/model"
)

type pdataLogsMarshaler struct {
	marshaler model.LogsMarshaler
	encoding  string
}

func (p pdataLogsMarshaler) Marshal(ld pdata.Logs, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := p.marshaler.Marshal(ld)
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

func (p pdataLogsMarshaler) Encoding() string {
	return p.encoding
}

func newPdataLogsMarshaler(marshaler model.LogsMarshaler, encoding string) LogsMarshaler {
	return pdataLogsMarshaler{
		marshaler: marshaler,
		encoding:  encoding,
	}
}

type pdataMetricsMarshaler struct {
	marshaler model.MetricsMarshaler
	encoding  string
}

func (p pdataMetricsMarshaler) Marshal(ld pdata.Metrics, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := p.marshaler.Marshal(ld)
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

func (p pdataMetricsMarshaler) Encoding() string {
	return p.encoding
}

func newPdataMetricsMarshaler(marshaler model.MetricsMarshaler, encoding string) MetricsMarshaler {
	return pdataMetricsMarshaler{
		marshaler: marshaler,
		encoding:  encoding,
	}
}

type pdataTracesMarshaler struct {
	marshaler model.TracesMarshaler
	encoding  string
}

func (p pdataTracesMarshaler) Marshal(td pdata.Traces, topic string) ([]*sarama.ProducerMessage, error) {
	bts, err := p.marshaler.Marshal(td)
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

func (p pdataTracesMarshaler) Encoding() string {
	return p.encoding
}

func newPdataTracesMarshaler(marshaler model.TracesMarshaler, encoding string) TracesMarshaler {
	return pdataTracesMarshaler{
		marshaler: marshaler,
		encoding:  encoding,
	}
}
