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
	"fmt"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var errUnrecognizedEncoding = fmt.Errorf("unrecognized encoding")

// kafkaTracesProducer uses sarama to produce trace messages to Kafka.
type kafkaTracesProducer struct {
	producer   sarama.SyncProducer
	topic      string
	marshaller TracesMarshaller
	logger     *zap.Logger
}

func (e *kafkaTracesProducer) traceDataPusher(_ context.Context, td pdata.Traces) (int, error) {
	messages, err := e.marshaller.Marshal(td)
	if err != nil {
		return td.SpanCount(), consumererror.Permanent(err)
	}
	err = e.producer.SendMessages(producerMessages(messages, e.topic))
	if err != nil {
		return td.SpanCount(), err
	}
	return 0, nil
}

func (e *kafkaTracesProducer) Close(context.Context) error {
	return e.producer.Close()
}

// kafkaMetricsProducer uses sarama to produce metrics messages to kafka
type kafkaMetricsProducer struct {
	producer   sarama.SyncProducer
	topic      string
	marshaller MetricsMarshaller
	logger     *zap.Logger
}

func (e *kafkaMetricsProducer) metricsDataPusher(_ context.Context, md pdata.Metrics) (int, error) {
	messages, err := e.marshaller.Marshal(md)
	if err != nil {
		return md.MetricCount(), consumererror.Permanent(err)
	}
	err = e.producer.SendMessages(producerMessages(messages, e.topic))
	if err != nil {
		return md.MetricCount(), err
	}
	return 0, nil
}

func (e *kafkaMetricsProducer) Close(context.Context) error {
	return e.producer.Close()
}

func newSaramaProducer(config Config) (sarama.SyncProducer, error) {
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
	if err := ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}
	producer, err := sarama.NewSyncProducer(config.Brokers, c)
	if err != nil {
		return nil, err
	}
	return producer, nil
}

func newMetricsExporter(config Config, params component.ExporterCreateParams, marshallers map[string]MetricsMarshaller) (*kafkaMetricsProducer, error) {
	marshaller := marshallers[config.Encoding]
	if marshaller == nil {
		return nil, errUnrecognizedEncoding
	}
	producer, err := newSaramaProducer(config)
	if err != nil {
		return nil, err
	}

	return &kafkaMetricsProducer{
		producer:   producer,
		topic:      config.Topic,
		marshaller: marshaller,
		logger:     params.Logger,
	}, nil

}

// newTracesExporter creates Kafka exporter.
func newTracesExporter(config Config, params component.ExporterCreateParams, marshallers map[string]TracesMarshaller) (*kafkaTracesProducer, error) {
	marshaller := marshallers[config.Encoding]
	if marshaller == nil {
		return nil, errUnrecognizedEncoding
	}
	producer, err := newSaramaProducer(config)
	if err != nil {
		return nil, err
	}
	return &kafkaTracesProducer{
		producer:   producer,
		topic:      config.Topic,
		marshaller: marshaller,
		logger:     params.Logger,
	}, nil
}

func producerMessages(messages []Message, topic string) []*sarama.ProducerMessage {
	producerMessages := make([]*sarama.ProducerMessage, len(messages))
	for i := range messages {
		producerMessages[i] = &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(messages[i].Value),
		}
	}
	return producerMessages
}
