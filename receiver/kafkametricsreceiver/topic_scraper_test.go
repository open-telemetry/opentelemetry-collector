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
	"fmt"
	"regexp"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTopicScraper_Name(t *testing.T) {
	s := topicScraper{}
	assert.Equal(t, s.Name(), "topics")
}

func TestBrokersScraper_createTopicScraper(t *testing.T) {
	sc := sarama.NewConfig()
	newSaramaClient = mockNewSaramaClient
	ms, err := createTopicsScraper(context.Background(), Config{}, sc, zap.NewNop())
	assert.Nil(t, err)
	assert.NotNil(t, ms)
}

func TestTopicScraper_createScraper_handles_client_error(t *testing.T) {
	newSaramaClient = func(addrs []string, conf *sarama.Config) (sarama.Client, error) {
		return nil, fmt.Errorf("new client failed")
	}
	sc := sarama.NewConfig()
	ms, err := createTopicsScraper(context.Background(), Config{}, sc, zap.NewNop())
	assert.NotNil(t, err)
	assert.Nil(t, ms)
}


func TestBrokerScraper_shutdown(t *testing.T) {
	client := getMockClient()
	client.Mock.
		On("Close").Return(nil).
		On("Closed").Return(false)
	scraper := topicScraper{
		client: client,
	}
	_ = scraper.shutdown(context.Background())
	client.AssertExpectations(t)
}

func TestBrokerScraper_shutdown_closed(t *testing.T) {
	client := getMockClient()
	client.closed = func() bool {
		return true
	}
	client.Mock.
		On("Closed").Return(true)
	scraper := topicScraper{
		client: client,
	}
	_ = scraper.shutdown(context.Background())
	client.AssertExpectations(t)
}


func TestTopicScraper_scrape(t *testing.T) {
	client := getMockClient()
	config := createDefaultConfig().(*Config)
	match := regexp.MustCompile(config.TopicMatch)
	scraper := topicScraper{
		client:      client,
		logger:      zap.NewNop(),
		topicFilter: match,
	}
	ms, err := scraper.scrape(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, ms.Len(), 5)
	pm := ms.At(0)
	assert.Equal(t, pm.Name(), partitionsName)
	assert.Equal(t, pm.IntGauge().DataPoints().At(0).Value(), int64(1), "number of partitions must match test value")
	cm := ms.At(1)
	assert.Equal(t, cm.Name(), currentOffsetName)
	assert.Equal(t, cm.IntGauge().DataPoints().At(0).Value(), int64(1), "current offset must match test value")
	om := ms.At(2)
	assert.Equal(t, om.Name(), oldestOffsetName)
	assert.Equal(t, om.IntGauge().DataPoints().At(0).Value(), int64(1), "oldest offset must match test value")
	rm := ms.At(3)
	assert.Equal(t, rm.Name(), replicasName)
	assert.Equal(t, rm.IntGauge().DataPoints().At(0).Value(), int64(1), "number of replicas should match test value")
	im := ms.At(4)
	assert.Equal(t, im.Name(), replicasInSyncName)
	assert.Equal(t, im.IntGauge().DataPoints().At(0).Value(), int64(1), "replicas in sync must test value")
}
