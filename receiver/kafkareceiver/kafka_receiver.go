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

package kafkareceiver

import (
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	transport = "kafka"
)

var errUnrecognizedEncoding = fmt.Errorf("unrecognized encoding")

// kafkaConsumer uses sarama to consume and handle messages from kafka.
type kafkaConsumer struct {
	name              string
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.TracesConsumer
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaller      Unmarshaller

	logger *zap.Logger
}

var _ component.Receiver = (*kafkaConsumer)(nil)

func newReceiver(config Config, params component.ReceiverCreateParams, unmarshalers map[string]Unmarshaller, nextConsumer consumer.TracesConsumer) (*kafkaConsumer, error) {
	unmarshaller := unmarshalers[config.Encoding]
	if unmarshaller == nil {
		return nil, errUnrecognizedEncoding
	}

	c := sarama.NewConfig()
	c.ClientID = config.ClientID
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
	if err := kafkaexporter.ConfigureAuthentication(config.Authentication, c); err != nil {
		return nil, err
	}
	client, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, c)
	if err != nil {
		return nil, err
	}
	return &kafkaConsumer{
		name:          config.Name(),
		consumerGroup: client,
		topics:        []string{config.Topic},
		nextConsumer:  nextConsumer,
		unmarshaller:  unmarshaller,
		logger:        params.Logger,
	}, nil
}

func (c *kafkaConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	consumerGroup := &consumerGroupHandler{
		name:         c.name,
		logger:       c.logger,
		unmarshaller: c.unmarshaller,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
	}
	go c.consumeLoop(ctx, consumerGroup)
	<-consumerGroup.ready
	return nil
}

func (c *kafkaConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
			c.logger.Error("Error from consumer", zap.Error(err))
		}
		// check if context was cancelled, signaling that the consumer should stop
		if ctx.Err() != nil {
			c.logger.Info("Consumer stopped", zap.Error(ctx.Err()))
			return ctx.Err()
		}
	}
}

func (c *kafkaConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

type consumerGroupHandler struct {
	name         string
	unmarshaller Unmarshaller
	nextConsumer consumer.TracesConsumer
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger
}

var _ sarama.ConsumerGroupHandler = (*consumerGroupHandler)(nil)

func (c *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.name)}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.name)}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		session.MarkMessage(message, "")

		ctx := obsreport.ReceiverContext(session.Context(), c.name, transport)
		ctx = obsreport.StartTraceDataReceiveOp(ctx, c.name, transport)
		statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.name)}
		_ = stats.RecordWithTags(ctx, statsTags,
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		traces, err := c.unmarshaller.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshall message", zap.Error(err))
			return err
		}

		err = c.nextConsumer.ConsumeTraces(session.Context(), traces)
		obsreport.EndTraceDataReceiveOp(ctx, c.unmarshaller.Encoding(), traces.SpanCount(), err)
		if err != nil {
			return err
		}
	}
	return nil
}
