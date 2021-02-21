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
	topics, err := s.client.Topics()
	metrics := pdata.NewMetricSlice()
	if err != nil {
		s.logger.Error("Topics Scraper: Failed to refresh topics. Error: ", zap.Error(err))
		return metrics, err
	}

	allMetrics := initializeTopicMetrics(&metrics)
	partitionsMetric := allMetrics.partitions
	currentOffsetMetric := allMetrics.currentOffset
	oldestOffsetMetric := allMetrics.oldestOffset
	replicasMetric := allMetrics.replicas
	replicasInSyncMetric := allMetrics.replicasInSync

	var matchedTopics []string
	for _, t := range topics {
		if s.topicFilter.MatchString(t) {
			matchedTopics = append(matchedTopics, t)
		}
	}
	partitionsMetric.IntGauge().DataPoints().Resize(len(matchedTopics))

	for topicIdx, topic := range matchedTopics {
		partitions, err := s.client.Partitions(topic)

		if err != nil {
			s.logger.Error("topics scraper: failed to get topic partitions", zap.String("Topic", topic), zap.Error(err))
			continue
		}
		addPartitionsToMetric(topic, int64(len(partitions)), partitionsMetric, topicIdx)

		for _, partition := range partitions {
			currentOffset, err := s.client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				s.logger.Error("topics scraper: failed to get current offset", zap.String("topic ", topic), zap.String("partition ", string(partition)), zap.Error(err))
			} else {
				addPartitionDPToMetric(topic, partition, currentOffset, currentOffsetMetric)
			}

			oldestOffset, err := s.client.GetOffset(topic, partition, sarama.OffsetOldest)
			if err != nil {
				s.logger.Error("topics scraper: failed to get oldest offset", zap.String("topic ", topic), zap.String("partition ", string(partition)), zap.Error(err))
			} else {
				addPartitionDPToMetric(topic, partition, oldestOffset, oldestOffsetMetric)
			}

			replicas, err := s.client.Replicas(topic, partition)
			if err != nil {
				s.logger.Error("topics scraper: failed to get replicas", zap.String("topic ", topic), zap.String("partition ", string(partition)), zap.Error(err))
			} else {
				addPartitionDPToMetric(topic, partition, int64(len(replicas)), replicasMetric)
			}

			replicasInSync, err := s.client.InSyncReplicas(topic, partition)
			if err != nil {
				s.logger.Error("topics scraper: failed to get in-sync replicas", zap.String("topic ", topic), zap.String("partition ", string(partition)), zap.Error(err))

			} else {
				addPartitionDPToMetric(topic, partition, int64(len(replicasInSync)), replicasInSyncMetric)
			}
		}
	}
	return metrics, nil
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
