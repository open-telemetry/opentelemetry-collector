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
	"context"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// kafkaProducer uses sarama to produce messages to Kafka.
type kafkaProducer struct {
	producer   sarama.SyncProducer
	topic      string
	marshaller marshaller
	logger     *zap.Logger
}

// newExporter creates Kafka exporter.
func newExporter(config Config, params component.ExporterCreateParams) (*kafkaProducer, error) {
	c := sarama.NewConfig()
	// These setting are required by the sarama.SyncProducer implementation.
	c.Producer.Return.Successes = true
	c.Producer.Return.Errors = true
	// Wait only the local commit to succeed before responding.
	c.Producer.RequiredAcks = sarama.WaitForLocal
	// Because sarama does not accept a Context for every message, set the Timeout here.
	c.Producer.Timeout = config.Timeout
	c.Metadata.Full = config.Metadata.Full
	c.Metadata.Retry.Max = config.Metadata.Retry.Max
	c.Metadata.Retry.Backoff = config.Metadata.Retry.Backoff
	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}
	producer, err := sarama.NewSyncProducer(config.Brokers, c)
	if err != nil {
		return nil, err
	}
	return &kafkaProducer{
		producer:   producer,
		topic:      config.Topic,
		marshaller: &protoMarshaller{},
		logger:     params.Logger,
	}, nil
}

func (e *kafkaProducer) traceDataPusher(_ context.Context, td pdata.Traces) (int, error) {
	bts, err := e.marshaller.Marshal(td)
	if err != nil {
		return td.SpanCount(), consumererror.Permanent(err)
	}
	m := &sarama.ProducerMessage{
		Topic: e.topic,
		Value: sarama.ByteEncoder(bts),
	}
	_, _, err = e.producer.SendMessage(m)
	if err != nil {
		return td.SpanCount(), err
	}
	return 0, nil
}

func (e *kafkaProducer) Close(context.Context) error {
	return e.producer.Close()
}
