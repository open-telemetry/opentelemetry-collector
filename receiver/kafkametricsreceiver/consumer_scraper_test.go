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

func TestConsumerShutdown(t *testing.T) {
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
	scraper := consumerScraper{
		client: client,
	}
	_ = scraper.shutdown(context.Background())
	client.AssertExpectations(t)
}

func TestConsumerShutdown_closed(t *testing.T) {
	client := getMockClient()
	client.closed = func() bool {
		return true
	}
	client.Mock.
		On("Closed").Return(true)
	scraper := consumerScraper{
		client: client,
	}
	_ = scraper.shutdown(context.Background())
	client.AssertExpectations(t)
}

func TestConsumerScraper_Name(t *testing.T) {
	s := consumerScraper{}
	assert.Equal(t, s.Name(), "consumers")
}

func TestConsumerScraper_createConsumerScraper(t *testing.T) {
	sc := sarama.NewConfig()
	newSaramaClient = mockNewSaramaClient
	newClusterAdmin = mockNewClusterAdmin
	ms, err := createConsumerScraper(context.Background(), Config{}, sc, zap.NewNop())
	assert.Nil(t, err)
	assert.NotNil(t, ms)
}

func TestConsumerScraper_createScraper_handles_client_error(t *testing.T) {
	newSaramaClient = func(addrs []string, conf *sarama.Config) (sarama.Client, error) {
		return nil, fmt.Errorf("new client failed")
	}
	sc := sarama.NewConfig()
	ms, err := createConsumerScraper(context.Background(), Config{}, sc, zap.NewNop())
	assert.NotNil(t, err)
	assert.Nil(t, ms)
}

func TestConsumerScraper_createScraper_handles_cluserAdmin_error(t *testing.T) {
	newSaramaClient = mockNewSaramaClient
	newClusterAdmin = func(addrs []string, conf *sarama.Config) (sarama.ClusterAdmin, error) {
		return nil, fmt.Errorf("new cluster failed")
	}
	sc := sarama.NewConfig()
	ms, err := createConsumerScraper(context.Background(), Config{}, sc, zap.NewNop())
	assert.NotNil(t, err)
	assert.Nil(t, ms)
}

func TestConsumerScraper_scrape(t *testing.T) {
	bs := consumerScraper{
		client: getMockClient(),
		logger: zap.NewNop(),
	}
	ms, err := bs.scrape(context.Background())
	assert.Nil(t, err)
	assert.NotNil(t, ms)
}
