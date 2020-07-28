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
	"context"

	"github.com/Shopify/sarama"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/pdata"
)

// kafkaProducer uses sarama to produce messages to Kafka.
type kafkaProducer struct {
	asyncProducer sarama.AsyncProducer
	topic         string
	marshaller    marshaller
	logger        *zap.Logger
}

// newExporter creates Kafka exporter.
func newExporter(config Config, params component.ExporterCreateParams) (*kafkaProducer, error) {
	c := sarama.NewConfig()
	// deliver confirmations to the success channel
	c.Producer.Return.Successes = true
	// deliver errors to the error channel
	c.Producer.Return.Errors = true
	if config.ProtocolVersion != "" {
		version, err := sarama.ParseKafkaVersion(config.ProtocolVersion)
		if err != nil {
			return nil, err
		}
		c.Version = version
	}
	producer, err := sarama.NewAsyncProducer(config.Brokers, c)
	if err != nil {
		return nil, err
	}
	kp := &kafkaProducer{
		asyncProducer: producer,
		topic:         config.Topic,
		marshaller:    &protoMarshaller{},
		logger:        params.Logger,
	}
	kp.processSendResults(config.Name())
	return kp, nil
}

func (e *kafkaProducer) processSendResults(name string) {
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, name)}
	go func() {
		for range e.asyncProducer.Successes() {
			_ = stats.RecordWithTags(context.Background(), statsTags, statSendSuccess.M(1))
		}
	}()
	go func() {
		for err := range e.asyncProducer.Errors() {
			e.logger.Error("Sending a message to Kafka failed", zap.Error(err.Err))
			statsTags := []tag.Mutator{tag.Insert(tagInstanceName, name)}
			_ = stats.RecordWithTags(context.Background(), statsTags, statSendErr.M(1))
		}
	}()
}

func (e *kafkaProducer) traceDataPusher(_ context.Context, td pdata.Traces) (int, error) {
	bts, err := e.marshaller.Marshal(td)
	if err != nil {
		return td.SpanCount(), consumererror.Permanent(err)
	}
	e.asyncProducer.Input() <- &sarama.ProducerMessage{
		Topic: e.topic,
		Value: sarama.ByteEncoder(bts),
	}
	return 0, nil
}

func (e *kafkaProducer) Close(context.Context) error {
	return e.asyncProducer.Close()
}
