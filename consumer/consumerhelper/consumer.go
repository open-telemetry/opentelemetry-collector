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

package consumerhelper

import (
	"go.opentelemetry.io/collector/consumer"
)

type baseConsumer struct {
	capabilities consumer.Capabilities
}

func (bc *baseConsumer) GetCapabilities() consumer.Capabilities {
	return bc.capabilities
}

// NewConsumer returns a consumer.Consumer that implements the basic consumer functionality.
func NewConsumer(capabilities consumer.Capabilities) consumer.Consumer {
	return &baseConsumer{
		capabilities: capabilities,
	}
}
