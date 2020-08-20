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
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/kafkaexporter"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	typeStr         = "kafka"
	defaultTopic    = "otlp_spans"
	defaultBroker   = "localhost:9092"
	defaultClientID = "otel-collector"
	defaultGroupID  = defaultClientID

	// default from sarama.NewConfig()
	defaultMetadataRetryMax = 3
	// default from sarama.NewConfig()
	defaultMetadataRetryBackoff = time.Millisecond * 250
	// default from sarama.NewConfig()
	defaultMetadataFull = true
)

// NewFactory creates Kafka receiver factory.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithTraces(createTraceReceiver))
}

func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		Topic:    defaultTopic,
		Brokers:  []string{defaultBroker},
		ClientID: defaultClientID,
		GroupID:  defaultGroupID,
		Metadata: kafkaexporter.Metadata{
			Full: defaultMetadataFull,
			Retry: kafkaexporter.MetadataRetry{
				Max:     defaultMetadataRetryMax,
				Backoff: defaultMetadataRetryBackoff,
			},
		},
	}
}

func createTraceReceiver(
	_ context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	c := cfg.(*Config)
	r, err := newReceiver(*c, params, nextConsumer)
	if err != nil {
		return nil, err
	}
	return r, nil
}
