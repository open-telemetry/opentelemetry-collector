// Copyright 2019, OpenTelemetry Authors
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

package opencensusreceiver

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-service/internal/testutils"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := receiver.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateReceiver(t *testing.T) {
	factory := receiver.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	config := cfg.(*ConfigV2)
	config.Endpoint = testutils.GetAvailableLocalAddress(t)

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), zap.NewNop(), cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.Nil(t, err)

	// The default config does not provide scrape_config so we expect that metrics receiver
	// creation must also fail.
	mReceiver, err := factory.CreateMetricsReceiver(zap.NewNop(), cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.Nil(t, err)
}
