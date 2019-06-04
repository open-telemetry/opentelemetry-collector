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

package opencensusreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/census-instrumentation/opencensus-service/internal/factories"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
}

func TestCreateReceiver(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)
	cfg := factory.CreateDefaultConfig()

	tReceiver, err := factory.CreateTraceReceiver(context.Background(), cfg, nil)
	assert.NotNil(t, tReceiver)
	assert.Nil(t, err)

	// The default config does not provide scrape_config so we expect that metrics receiver
	// creation must also fail.
	mReceiver, err := factory.CreateMetricsReceiver(cfg, nil)
	assert.NotNil(t, mReceiver)
	assert.Nil(t, err)
}
