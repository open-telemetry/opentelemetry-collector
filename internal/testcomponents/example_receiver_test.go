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

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component/componenttest"
)

func TestExampleReceiverProducer(t *testing.T) {
	rcv := &ExampleReceiverProducer{}
	host := componenttest.NewNopHost()
	assert.False(t, rcv.Started)
	err := rcv.Start(context.Background(), host)
	assert.NoError(t, err)
	assert.True(t, rcv.Started)

	err = rcv.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, rcv.Started)
}
