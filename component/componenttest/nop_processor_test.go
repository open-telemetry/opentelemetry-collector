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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestNewNopProcessorFactory(t *testing.T) {
	factory := NewNopProcessorFactory()
	require.NotNil(t, factory)
	assert.Equal(t, configmodels.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &configmodels.ProcessorSettings{TypeVal: factory.Type()}, cfg)

	traces, err := factory.CreateTracesProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewTracesNop())
	require.NoError(t, err)
	assert.Equal(t, component.ProcessorCapabilities{MutatesConsumedData: false}, traces.GetCapabilities())
	assert.NoError(t, traces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, traces.ConsumeTraces(context.Background(), pdata.NewTraces()))
	assert.NoError(t, traces.Shutdown(context.Background()))

	metrics, err := factory.CreateMetricsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewMetricsNop())
	require.NoError(t, err)
	assert.Equal(t, component.ProcessorCapabilities{MutatesConsumedData: false}, metrics.GetCapabilities())
	assert.NoError(t, metrics.Start(context.Background(), NewNopHost()))
	assert.NoError(t, metrics.ConsumeMetrics(context.Background(), pdata.NewMetrics()))
	assert.NoError(t, metrics.Shutdown(context.Background()))

	logs, err := factory.CreateLogsProcessor(context.Background(), component.ProcessorCreateParams{}, cfg, consumertest.NewLogsNop())
	require.NoError(t, err)
	assert.Equal(t, component.ProcessorCapabilities{MutatesConsumedData: false}, logs.GetCapabilities())
	assert.NoError(t, logs.Start(context.Background(), NewNopHost()))
	assert.NoError(t, logs.ConsumeLogs(context.Background(), pdata.NewLogs()))
	assert.NoError(t, logs.Shutdown(context.Background()))
}
