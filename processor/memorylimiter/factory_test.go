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

package memorylimiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config/configcheck"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := &Factory{}
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := &Factory{}
	require.NotNil(t, factory)

	cfg := factory.CreateDefaultConfig()

	// This processor can't be created with the default config.
	tp, err := factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	assert.Nil(t, tp)
	assert.Error(t, err, "created processor with invalid settings")

	mp, err := factory.CreateMetricsProcessor(zap.NewNop(), exportertest.NewNopMetricsExporter(), cfg)
	assert.Nil(t, mp)
	assert.Error(t, err, "created processor with invalid settings")

	// Create processor with a valid config.
	pCfg := cfg.(*Config)
	pCfg.MemoryLimitMiB = 5722
	pCfg.MemorySpikeLimitMiB = 1907
	pCfg.BallastSizeMiB = 2048
	pCfg.CheckInterval = 100 * time.Millisecond

	tp, err = factory.CreateTraceProcessor(zap.NewNop(), exportertest.NewNopTraceExporter(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, tp)
	assert.NoError(t, tp.Shutdown())

	mp, err = factory.CreateMetricsProcessor(zap.NewNop(), exportertest.NewNopMetricsExporter(), cfg)
	assert.NoError(t, err)
	assert.NotNil(t, mp)
	assert.NoError(t, mp.Shutdown())
}
