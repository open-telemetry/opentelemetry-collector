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

package probabilisticsamplerprocessor

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/processor"
)

func TestLoadConfig(t *testing.T) {
	receivers, processors, exporters, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	processors[typeStr] = factory
	cfg, err := config.LoadConfigFile(
		t, path.Join(".", "testdata", "config.yaml"), receivers, processors, exporters,
	)

	require.Nil(t, err)
	require.NotNil(t, cfg)

	p0 := cfg.Processors["probabilistic-sampler"]
	assert.Equal(t, p0,
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "probabilistic-sampler",
				NameVal: "probabilistic-sampler",
			},
			SamplingPercentage: 15.3,
			HashSeed:           22,
		})

}

func TestLoadConfigEmpty(t *testing.T) {
	receivers, _, exporters, err := config.ExampleComponents()
	require.NoError(t, err)
	processors, err := processor.Build(&Factory{})
	require.NotNil(t, processors)
	require.NoError(t, err)

	config, err := config.LoadConfigFile(
		t,
		path.Join(".", "testdata", "empty.yaml"),
		receivers,
		processors,
		exporters)

	require.Nil(t, err)
	require.NotNil(t, config)
	p0 := config.Processors["probabilistic-sampler"]
	factory := &Factory{}
	assert.Equal(t, p0, factory.CreateDefaultConfig())
}
