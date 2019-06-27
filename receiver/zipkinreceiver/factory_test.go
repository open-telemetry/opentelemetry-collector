// Copyright 2019, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zipkinreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/data"
	"github.com/open-telemetry/opentelemetry-service/pkg/factories"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

type mockTraceConsumer struct {
}

func (m *mockTraceConsumer) ConsumeTraceData(ctx context.Context, td data.TraceData) error { return nil }

func TestCreateReceiver(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), cfg, &mockTraceConsumer{})
	assert.Nil(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(cfg, nil)
	assert.Equal(t, err, factories.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}
