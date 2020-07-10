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

package attributesprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configerror"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/internal/processor/attraction"
)

func TestFactory_Type(t *testing.T) {
	factory := Factory{}
	assert.Equal(t, factory.Type(), configmodels.Type(typeStr))
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.Equal(t, cfg, &Config{
		ProcessorSettings: configmodels.ProcessorSettings{
			NameVal: typeStr,
			TypeVal: typeStr,
		},
	})
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestFactoryCreateTraceProcessor_EmptyActions(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	ap, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateTraceProcessor_InvalidActions(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	// Missing key
	oCfg.Actions = []attraction.ActionKeyValue{
		{Key: "", Value: 123, Action: attraction.UPSERT},
	}
	ap, err := factory.CreateTraceProcessor(context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.Error(t, err)
	assert.Nil(t, ap)
}

func TestFactoryCreateTraceProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	oCfg := cfg.(*Config)
	oCfg.Actions = []attraction.ActionKeyValue{
		{Key: "a key", Action: attraction.DELETE},
	}

	tp, err := factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.NotNil(t, tp)
	assert.NoError(t, err)

	tp, err = factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, nil, cfg)
	assert.Nil(t, tp)
	assert.NotNil(t, err)

	oCfg.Actions = []attraction.ActionKeyValue{
		{Action: attraction.DELETE},
	}
	tp, err = factory.CreateTraceProcessor(
		context.Background(), component.ProcessorCreateParams{}, exportertest.NewNopTraceExporter(), cfg)
	assert.Nil(t, tp)
	assert.NotNil(t, err)
}

func TestFactory_CreateMetricsProcessor(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()

	mp, err := factory.CreateMetricsProcessor(
		context.Background(), component.ProcessorCreateParams{}, nil, cfg)
	require.Nil(t, mp)
	assert.Equal(t, err, configerror.ErrDataTypeIsNotSupported)
}
