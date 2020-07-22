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
	"go.opentelemetry.io/collector/component/componenterror"
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
	c.Producer.Return.Successes = true
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
	go func() {
		for range e.asyncProducer.Successes() {
			statsTags := []tag.Mutator{tag.Insert(tagInstanceName, name)}
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
	sendSpans := 0
	var errs []error
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rss := td.ResourceSpans().At(i)
		bts, err := e.marshaller.Marshal(rss)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		e.asyncProducer.Input() <- &sarama.ProducerMessage{
			Topic: e.topic,
			Value: sarama.ByteEncoder(bts),
		}
		sendSpans += resourceSpansCount(rss)
	}
	if len(errs) > 0 {
		err := componenterror.CombineErrors(errs)
		return td.SpanCount() - sendSpans, consumererror.Permanent(err)
	}
	return 0, nil
}

func (e *kafkaProducer) Close(context.Context) error {
	return e.asyncProducer.Close()
}

func resourceSpansCount(spans pdata.ResourceSpans) int {
	size := 0

	if spans.IsNil() {
		return 0
	}
	for i := 0; i < spans.InstrumentationLibrarySpans().Len(); i++ {
		size += spans.InstrumentationLibrarySpans().At(i).Spans().Len()
	}
	return size
}
