// Copyright The OpenTelemetry Authors
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

package exprfilterprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestType(t *testing.T) {
	factory := NewFactory()
	typ := factory.Type()
	require.Equal(t, typ, configmodels.Type("exprfilter"))
}

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: typeStr,
			TypeVal: typeStr,
		},
	})
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	pcfg := cfg.(*Config)
	pcfg.Query = "foo == 'bar'"
	ctx := context.Background()
	params := component.ProcessorCreateParams{Logger: zap.NewNop()}
	tp, err := factory.CreateTraceProcessor(
		ctx,
		params,
		cfg,
		exportertest.NewNopTraceExporter(),
	)
	assert.Nil(t, tp)
	assert.Error(t, err)

	lp, err := factory.CreateLogsProcessor(ctx, params, cfg, exportertest.NewNopLogsExporter())
	assert.Nil(t, lp)
	assert.Error(t, err)

	mp, err := factory.CreateMetricsProcessor(ctx, params, cfg, exportertest.NewNopMetricsExporter())
	assert.NotNil(t, mp)
	assert.NoError(t, err)
}
