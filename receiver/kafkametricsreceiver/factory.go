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

package kafkametricsreceiver

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

const (
	typeStr           = "kafkametrics"
	defaultBroker     = "localhost:9092"
	defaultGroupMatch = ".*"
	defaultTopicMatch = ".*"
	defaultClientID   = "otel-metrics-receiver"
)

// NewFactory creates kafkametrics receiver factory.
func NewFactory() component.ReceiverFactory {
	return receiverhelper.NewFactory(
		typeStr,
		createDefaultConfig,
		receiverhelper.WithMetrics(createMetricsReceiver))
}

func createDefaultConfig() configmodels.Receiver {
	return &Config{
		ScraperControllerSettings: scraperhelper.DefaultScraperControllerSettings(typeStr),
		Brokers:                   []string{defaultBroker},
		GroupMatch:                defaultGroupMatch,
		TopicMatch:                defaultTopicMatch,
		ClientID:                  defaultClientID,
	}
}

func createMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	nextConsumer consumer.MetricsConsumer) (component.MetricsReceiver, error) {
	c := cfg.(*Config)
	r, err := newMetricsReceiver(ctx, *c, params, nextConsumer)
	if err != nil {
		return nil, err
	}
	return r, nil
}
