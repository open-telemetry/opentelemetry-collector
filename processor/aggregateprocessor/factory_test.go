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

package aggregateprocessor

import (
	"context"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}


func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, factory.Type(), typeStr)
}

func TestFactory_CreateDefaultConfig(t *testing.T) {
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

func TestFactory_CreateTraceProcessor(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	config := cfg.(*Config)
	params := component.ProcessorCreateParams{Logger: zap.NewNop()}
	ap, err := factory.CreateTraceProcessor(context.Background(), params, config, exportertest.NewNopTraceExporter())
	assert.NotNil(t, ap)
	assert.Nil(t, err)
}

