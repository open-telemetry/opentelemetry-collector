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

package tailsamplingprocessor

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/config"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Processors[factory.Type()] = factory

	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "tail_sampling_config.yaml"), factories)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, cfg.Processors["tail_sampling"],
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail_sampling",
				NameVal: "tail_sampling",
			},
			DecisionWait:            10 * time.Second,
			NumTraces:               100,
			ExpectedNewTracesPerSec: 10,
			PolicyCfgs: []PolicyCfg{
				{
					Name: "test-policy-1",
					Type: AlwaysSample,
				},
				{
					Name:                "test-policy-2",
					Type:                NumericAttribute,
					NumericAttributeCfg: NumericAttributeCfg{Key: "key1", MinValue: 50, MaxValue: 100},
				},
				{
					Name:               "test-policy-3",
					Type:               StringAttribute,
					StringAttributeCfg: StringAttributeCfg{Key: "key2", Values: []string{"value1", "value2"}},
				},
				{
					Name:            "test-policy-4",
					Type:            RateLimiting,
					RateLimitingCfg: RateLimitingCfg{SpansPerSecond: 35},
				},
			},
		})
}
