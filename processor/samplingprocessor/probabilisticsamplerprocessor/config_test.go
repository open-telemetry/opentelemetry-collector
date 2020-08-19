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

package probabilisticsamplerprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Processors[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.Nil(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors["probabilistic_sampler"]
	assert.Equal(t, p0,
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "probabilistic_sampler",
				NameVal: "probabilistic_sampler",
			},
			SamplingPercentage: 15.3,
			HashSeed:           22,
		})

}

func TestLoadConfigEmpty(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	require.NoError(t, err)
	factories.Processors, err = component.MakeProcessorFactoryMap(NewFactory())
	require.NotNil(t, factories.Processors)
	require.NoError(t, err)

	config, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "empty.yaml"), factories)

	require.Nil(t, err)
	require.NotNil(t, config)
	p0 := config.Processors["probabilistic_sampler"]
	assert.Equal(t, p0, createDefaultConfig())
}
