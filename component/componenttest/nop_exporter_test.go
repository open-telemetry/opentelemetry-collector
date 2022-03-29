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
	"go.opentelemetry.io/collector/model/pdata/logs"
	"go.opentelemetry.io/collector/model/pdata/metrics"
	"go.opentelemetry.io/collector/model/pdata/traces"
)

func TestNewNopExporterFactory(t *testing.T) {
	factory := NewNopExporterFactory()
	require.NotNil(t, factory)
	assert.Equal(t, config.Type("nop"), factory.Type())
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, &nopExporterConfig{ExporterSettings: config.NewExporterSettings(config.NewComponentID("nop"))}, cfg)

	te, err := factory.CreateTracesExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, te.Start(context.Background(), NewNopHost()))
	assert.NoError(t, te.ConsumeTraces(context.Background(), traces.New()))
	assert.NoError(t, te.Shutdown(context.Background()))

	me, err := factory.CreateMetricsExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, me.Start(context.Background(), NewNopHost()))
	assert.NoError(t, me.ConsumeMetrics(context.Background(), metrics.New()))
	assert.NoError(t, me.Shutdown(context.Background()))

	le, err := factory.CreateLogsExporter(context.Background(), NewNopExporterCreateSettings(), cfg)
	require.NoError(t, err)
	assert.NoError(t, le.Start(context.Background(), NewNopHost()))
	assert.NoError(t, le.ConsumeLogs(context.Background(), logs.New()))
	assert.NoError(t, le.Shutdown(context.Background()))
}
