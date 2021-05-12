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
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer"
)

func TestDefaultOptions(t *testing.T) {
	bp := newBaseConsumer()
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, bp.Capabilities())
}

func TestWithCapabilities(t *testing.T) {
	bpMutate := newBaseConsumer(WithCapabilities(consumer.Capabilities{MutatesData: true}))
	assert.Equal(t, consumer.Capabilities{MutatesData: true}, bpMutate.Capabilities())

	bpNotMutate := newBaseConsumer(WithCapabilities(consumer.Capabilities{MutatesData: false}))
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, bpNotMutate.Capabilities())
}
