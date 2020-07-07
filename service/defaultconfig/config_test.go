// Copyright 2020 The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package defaultconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func TestDefaultConfig(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	cfg := CreateDefaultConfig(factories, &configmodels.Config{Service: configmodels.Service{Pipelines: configmodels.Pipelines{"traces": &configmodels.Pipeline{}}}})
	err = config.ValidateConfig(cfg, zap.NewNop())
	require.Error(t, err, "no enabled exporters specified in config")
	assert.Equal(t, configmodels.Service{
		Pipelines: configmodels.Pipelines{
			string(configmodels.TracesDataType): {
				InputType:  configmodels.TracesDataType,
				Receivers:  []string{"otlp"},
				Processors: []string{"batch"},
			},
		},
	}, cfg.Service)
	assert.Equal(t, factories.Receivers["otlp"].CreateDefaultConfig(), cfg.Receivers["otlp"])
	assert.Equal(t, factories.Processors["batch"].CreateDefaultConfig(), cfg.Processors["batch"])
}

func TestDefaultConfig_nil(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	cfg := CreateDefaultConfig(factories, nil)
	assert.Nil(t, cfg)
}
