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

var newSaramaClient = sarama.NewClient
var newClusterAdmin = sarama.NewClusterAdmin

func mockNewSaramaClient([]string, *sarama.Config) (sarama.Client, error) {
	return getMockClient(), nil
}

func mockNewClusterAdmin([]string, *sarama.Config) (sarama.ClusterAdmin, error) {
	return getMockClusterAdmin(), nil
}

type mockSaramaClient struct {
	mock.Mock
	sarama.Client
}

func (s *mockSaramaClient) Closed() bool {
	s.Called()
	return false
}

func (s *mockSaramaClient) Close() error {
	s.Called()
	return nil
}

func getMockClient() *mockSaramaClient {
	client := new(mockSaramaClient)
	return client
}

type mockClusterAdmin struct {
	mock.Mock
	sarama.ClusterAdmin
}

func getMockClusterAdmin() *mockClusterAdmin {
	clusterAdmin := new(mockClusterAdmin)
	return clusterAdmin
}
