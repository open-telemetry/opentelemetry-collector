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
	"go.opentelemetry.io/collector/model/pdata"
)

func TestNewNopExporterFactory(t *testing.T) {
	factory := NewNopExporterFactory()
	require.NotNil(t, factory)
	assert.Equal(t, config.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopExporterConfig{ExporterSettings: config.NewExporterSettings(config.NewID("nop"))}, cfg)

	traces, err := factory.CreateTracesExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, traces.Start(context.Background(), NewNopHost()))
	assert.NoError(t, traces.ConsumeTraces(context.Background(), pdata.NewTraces()))
	assert.NoError(t, traces.Shutdown(context.Background()))

	metrics, err := factory.CreateMetricsExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, metrics.Start(context.Background(), NewNopHost()))
	assert.NoError(t, metrics.ConsumeMetrics(context.Background(), pdata.NewMetrics()))
	assert.NoError(t, metrics.Shutdown(context.Background()))

	logs, err := factory.CreateLogsExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, logs.Start(context.Background(), NewNopHost()))
	assert.NoError(t, logs.ConsumeLogs(context.Background(), pdata.NewLogs()))
	assert.NoError(t, logs.Shutdown(context.Background()))
}
