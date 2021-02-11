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
	"regexp"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
)

type topicScraper struct {
	client       sarama.Client
	logger       *zap.Logger
	topicFilter  *regexp.Regexp
	saramaConfig *sarama.Config
	config       Config
}

func (s *topicScraper) Name() string {
	return "topics"
}

func (s *topicScraper) shutdown(context.Context) error {
	if !s.client.Closed() {
		return s.client.Close()
	}
	return nil
}

func (s *topicScraper) scrape(context.Context) (pdata.MetricSlice, error) {
	// TODO
	return pdata.MetricSlice{}, nil
}

func createTopicsScraper(_ context.Context, config Config, saramaConfig *sarama.Config, logger *zap.Logger) (scraperhelper.MetricsScraper, error) {
	topicFilter := regexp.MustCompile(config.TopicMatch)
	client, err := newSaramaClient(config.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}
	s := topicScraper{
		client:       client,
		logger:       logger,
		topicFilter:  topicFilter,
		saramaConfig: saramaConfig,
		config:       config,
	}
	return scraperhelper.NewMetricsScraper(
		s.Name(),
		s.scrape,
		scraperhelper.WithShutdown(s.shutdown),
	), nil
}
