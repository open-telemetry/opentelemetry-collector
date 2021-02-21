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
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/mock"
)

const (
	testBroker = "test_broker"
)

var newSaramaClient = sarama.NewClient

func mockNewSaramaClient([]string, *sarama.Config) (sarama.Client, error) {
	return getMockClient(), nil
}

type mockSaramaClient struct {
	mock.Mock
	sarama.Client

	close   func() error
	closed  func() bool
	brokers func() []*sarama.Broker
}

func (s *mockSaramaClient) Closed() bool {
	s.Called()
	return s.closed()
}

func (s *mockSaramaClient) Close() error {
	s.Called()
	return s.close()
}

func (s *mockSaramaClient) Brokers() []*sarama.Broker {
	s.Called()
	return s.brokers()
}

func getMockClient() *mockSaramaClient {
	client := new(mockSaramaClient)
	r := sarama.NewBroker(testBroker)
	testBrokers := make([]*sarama.Broker, 1)
	testBrokers[0] = r
	client.brokers = func() []*sarama.Broker {
		return testBrokers
	}
	return client
}
