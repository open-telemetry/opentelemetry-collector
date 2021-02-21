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

type consumerScraper struct {
	client       sarama.Client
	logger       *zap.Logger
	groupFilter  *regexp.Regexp
	topicFilter  *regexp.Regexp
	clusterAdmin sarama.ClusterAdmin
}

func (s *consumerScraper) Name() string {
	return "consumers"
}

func (s *consumerScraper) shutdown(_ context.Context) error {
	if !s.client.Closed() {
		return s.client.Close()
	}
	return nil
}

func (s *consumerScraper) scrape(context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()
	allMetrics := initializeConsumerMetrics(&metrics)
	cgs, listErr := s.clusterAdmin.ListConsumerGroups()
	if listErr != nil {
		return metrics, listErr
	}
	var matchedGrpIds []string
	for grpID := range cgs {
		if s.groupFilter.MatchString(grpID) {
			matchedGrpIds = append(matchedGrpIds, grpID)
		}
	}

	allTopics, listErr := s.clusterAdmin.ListTopics()
	if listErr != nil {
		return metrics, listErr
	}

	topics := make(map[string]sarama.TopicDetail)
	for t, d := range allTopics {
		if s.topicFilter.MatchString(t) {
			topics[t] = d
		}
	}
	// partitionIds for each topic
	topicPartitions := make(map[string][]int32)
	// currentOffset for each partition in each topic
	topicPartitionOffset := make(map[string]map[int32]int64)
	for topic := range topics {
		topicPartitions[topic] = make([]int32, 0)
		topicPartitionOffset[topic] = make(map[int32]int64)
		partitions, err := s.client.Partitions(topic)
		if err != nil {
			s.logger.Error("failed to fetch partitions for topic. ", zap.String("topic: ", topic), zap.Error(err))
			continue
		}
		for _, p := range partitions {
			o, err := s.client.GetOffset(topic, p, sarama.OffsetNewest)
			if err != nil {
				s.logger.Error("failed to fetch partition offset.", zap.String("topic", topic), zap.String("partition: ", int32ToStr(p)), zap.Error(err))
				continue
			}
			topicPartitions[topic] = append(topicPartitions[topic], p)
			topicPartitionOffset[topic][p] = o
		}
	}

	consumerGroups, listErr := s.clusterAdmin.DescribeConsumerGroups(matchedGrpIds)
	if listErr != nil {
		return metrics, listErr
	}

	for _, group := range consumerGroups {
		addGroupMembersToMetric(group.GroupId, int64(len(group.Members)), allMetrics.groupMembers)
		groupOffsetFetchResponse, err := s.clusterAdmin.ListConsumerGroupOffsets(group.GroupId, topicPartitions)
		if err != nil {
			s.logger.Error("failed to fetch consumer group offset.", zap.String("group", group.GroupId), zap.Error(err))
			continue
		}
		for topic, partitions := range groupOffsetFetchResponse.Blocks {
			// tracking topics consumed by consumer
			// checking if any of the blocks has an offset
			isConsumed := false
			for _, block := range partitions {
				if block.Offset != -1 {
					isConsumed = true
				}
			}
			if isConsumed {
				var lagSum int64 = 0
				var offsetSum int64 = 0
				for partition, block := range partitions {
					consumerOffset := block.Offset
					offsetSum += consumerOffset
					addConsumerPartitionDPToMetric(group.GroupId, topic, partition, consumerOffset, allMetrics.consumerOffset)

					var consumerLag int64 = -1
					if partitionOffset, ok := topicPartitionOffset[topic][partition]; ok {
						// for lag, only consider partitions with an offset
						if block.Offset != -1 {
							consumerLag = partitionOffset - consumerOffset
							lagSum += consumerLag
						}
					}
					addConsumerPartitionDPToMetric(group.GroupId, topic, partition, consumerLag, allMetrics.consumerLag)
				}
				addConsumerSumDPToMetric(group.GroupId, topic, lagSum, allMetrics.lagSum)
				addConsumerSumDPToMetric(group.GroupId, topic, offsetSum, allMetrics.offsetSum)
			}
		}
	}

	return metrics, nil
}

func createConsumerScraper(_ context.Context, config Config, saramaConfig *sarama.Config, logger *zap.Logger) (scraperhelper.MetricsScraper, error) {
	groupFilter := regexp.MustCompile(config.GroupMatch)
	topicFilter := regexp.MustCompile(config.TopicMatch)
	client, err := newSaramaClient(config.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}
	clusterAdmin, err := newClusterAdmin(config.Brokers, saramaConfig)
	if err != nil {
		return nil, err
	}
	s := consumerScraper{
		client:       client,
		logger:       logger,
		groupFilter:  groupFilter,
		topicFilter:  topicFilter,
		clusterAdmin: clusterAdmin,
	}
	return scraperhelper.NewMetricsScraper(
		s.Name(),
		s.scrape,
		scraperhelper.WithShutdown(s.shutdown),
	), nil
}
