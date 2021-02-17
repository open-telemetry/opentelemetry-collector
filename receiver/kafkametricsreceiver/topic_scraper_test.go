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
	client.closed = func() bool {
		return false
	}
	client.close = func() error {
		return nil
	}
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
	bs := topicScraper{
		client: getMockClient(),
		logger: zap.NewNop(),
		config: Config{},
	}
	ms, err := bs.scrape(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, ms)
}
