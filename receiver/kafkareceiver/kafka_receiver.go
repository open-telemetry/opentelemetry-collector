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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/obsreport"
)

const (
	transport = "kafka"
)

var errUnrecognizedEncoding = fmt.Errorf("unrecognized encoding")

// kafkaTracesConsumer uses sarama to consume and handle messages from kafka.
type kafkaTracesConsumer struct {
	id                config.ComponentID
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Traces
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       TracesUnmarshaler

	logger *zap.Logger
}

// kafkaLogsConsumer uses sarama to consume and handle messages from kafka.
type kafkaLogsConsumer struct {
	id                config.ComponentID
	consumerGroup     sarama.ConsumerGroup
	nextConsumer      consumer.Logs
	topics            []string
	cancelConsumeLoop context.CancelFunc
	unmarshaler       LogsUnmarshaler

	logger *zap.Logger
}

var _ component.Receiver = (*kafkaTracesConsumer)(nil)
var _ component.Receiver = (*kafkaLogsConsumer)(nil)

func newTracesReceiver(config Config, set component.ReceiverCreateSettings, unmarshalers map[string]TracesUnmarshaler, nextConsumer consumer.Traces) (*kafkaTracesConsumer, error) {
	unmarshaler := unmarshalers[config.Encoding]
	if unmarshaler == nil {
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
	return &kafkaTracesConsumer{
		id:            config.ID(),
		consumerGroup: client,
		topics:        []string{config.Topic},
		nextConsumer:  nextConsumer,
		unmarshaler:   unmarshaler,
		logger:        set.Logger,
	}, nil
}

func (c *kafkaTracesConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	consumerGroup := &tracesConsumerGroupHandler{
		id:           c.id,
		logger:       c.logger,
		unmarshaler:  c.unmarshaler,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{ReceiverID: c.id, Transport: transport}),
	}
	go c.consumeLoop(ctx, consumerGroup) // nolint:errcheck
	<-consumerGroup.ready
	return nil
}

func (c *kafkaTracesConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
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

func (c *kafkaTracesConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

func newLogsReceiver(config Config, set component.ReceiverCreateSettings, unmarshalers map[string]LogsUnmarshaler, nextConsumer consumer.Logs) (*kafkaLogsConsumer, error) {
	unmarshaler := unmarshalers[config.Encoding]
	if unmarshaler == nil {
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
	return &kafkaLogsConsumer{
		id:            config.ID(),
		consumerGroup: client,
		topics:        []string{config.Topic},
		nextConsumer:  nextConsumer,
		unmarshaler:   unmarshaler,
		logger:        set.Logger,
	}, nil
}

func (c *kafkaLogsConsumer) Start(context.Context, component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelConsumeLoop = cancel
	logsConsumerGroup := &logsConsumerGroupHandler{
		id:           c.id,
		logger:       c.logger,
		unmarshaler:  c.unmarshaler,
		nextConsumer: c.nextConsumer,
		ready:        make(chan bool),
		obsrecv:      obsreport.NewReceiver(obsreport.ReceiverSettings{ReceiverID: c.id, Transport: transport}),
	}
	go c.consumeLoop(ctx, logsConsumerGroup)
	<-logsConsumerGroup.ready
	return nil
}

func (c *kafkaLogsConsumer) consumeLoop(ctx context.Context, handler sarama.ConsumerGroupHandler) error {
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

func (c *kafkaLogsConsumer) Shutdown(context.Context) error {
	c.cancelConsumeLoop()
	return c.consumerGroup.Close()
}

type tracesConsumerGroupHandler struct {
	id           config.ComponentID
	unmarshaler  TracesUnmarshaler
	nextConsumer consumer.Traces
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *obsreport.Receiver
}

type logsConsumerGroupHandler struct {
	id           config.ComponentID
	unmarshaler  LogsUnmarshaler
	nextConsumer consumer.Logs
	ready        chan bool
	readyCloser  sync.Once

	logger *zap.Logger

	obsrecv *obsreport.Receiver
}

var _ sarama.ConsumerGroupHandler = (*tracesConsumerGroupHandler)(nil)
var _ sarama.ConsumerGroupHandler = (*logsConsumerGroupHandler)(nil)

func (c *tracesConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionStart.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.Name())}
	_ = stats.RecordWithTags(session.Context(), statsTags, statPartitionClose.M(1))
	return nil
}

func (c *tracesConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		session.MarkMessage(message, "")

		ctx := obsreport.ReceiverContext(session.Context(), c.id, transport)
		ctx = c.obsrecv.StartTracesOp(ctx)
		statsTags := []tag.Mutator{tag.Insert(tagInstanceName, c.id.String())}
		_ = stats.RecordWithTags(ctx, statsTags,
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		traces, err := c.unmarshaler.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshal message", zap.Error(err))
			return err
		}

		spanCount := traces.SpanCount()
		err = c.nextConsumer.ConsumeTraces(session.Context(), traces)
		c.obsrecv.EndTracesOp(ctx, c.unmarshaler.Encoding(), spanCount, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *logsConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	c.readyCloser.Do(func() {
		close(c.ready)
	})
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
		statPartitionStart.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	_ = stats.RecordWithTags(
		session.Context(),
		[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
		statPartitionClose.M(1))
	return nil
}

func (c *logsConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	c.logger.Info("Starting consumer group", zap.Int32("partition", claim.Partition()))
	for message := range claim.Messages() {
		c.logger.Debug("Kafka message claimed",
			zap.String("value", string(message.Value)),
			zap.Time("timestamp", message.Timestamp),
			zap.String("topic", message.Topic))
		session.MarkMessage(message, "")

		ctx := obsreport.ReceiverContext(session.Context(), c.id, transport)
		ctx = c.obsrecv.StartTracesOp(ctx)
		_ = stats.RecordWithTags(
			ctx,
			[]tag.Mutator{tag.Insert(tagInstanceName, c.id.String())},
			statMessageCount.M(1),
			statMessageOffset.M(message.Offset),
			statMessageOffsetLag.M(claim.HighWaterMarkOffset()-message.Offset-1))

		logs, err := c.unmarshaler.Unmarshal(message.Value)
		if err != nil {
			c.logger.Error("failed to unmarshal message", zap.Error(err))
			return err
		}

		err = c.nextConsumer.ConsumeLogs(session.Context(), logs)
		// TODO
		c.obsrecv.EndTracesOp(ctx, c.unmarshaler.Encoding(), logs.LogRecordCount(), err)
		if err != nil {
			return err
		}
	}
	return nil
}
