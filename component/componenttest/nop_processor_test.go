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

package componenttest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/model/pdata/logs"
	"go.opentelemetry.io/collector/model/pdata/metrics"
	"go.opentelemetry.io/collector/model/pdata/traces"
)

func TestNewNopProcessorFactory(t *testing.T) {
	factory := NewNopProcessorFactory()
	require.NotNil(t, factory)
	assert.Equal(t, config.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopProcessorConfig{ProcessorSettings: config.NewProcessorSettings(config.NewComponentID("nop"))}, cfg)

	tp, err := factory.CreateTracesProcessor(context.Background(), NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, tp.Capabilities())
	assert.NoError(t, tp.Start(context.Background(), NewNopHost()))
	assert.NoError(t, tp.ConsumeTraces(context.Background(), traces.New()))
	assert.NoError(t, tp.Shutdown(context.Background()))

	mp, err := factory.CreateMetricsProcessor(context.Background(), NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, mp.Capabilities())
	assert.NoError(t, mp.Start(context.Background(), NewNopHost()))
	assert.NoError(t, mp.ConsumeMetrics(context.Background(), metrics.New()))
	assert.NoError(t, mp.Shutdown(context.Background()))

	lp, err := factory.CreateLogsProcessor(context.Background(), NewNopProcessorCreateSettings(), cfg, consumertest.NewNop())
	require.NoError(t, err)
	assert.Equal(t, consumer.Capabilities{MutatesData: false}, lp.Capabilities())
	assert.NoError(t, lp.Start(context.Background(), NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), logs.New()))
	assert.NoError(t, lp.Shutdown(context.Background()))
}
