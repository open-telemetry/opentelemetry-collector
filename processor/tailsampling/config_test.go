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

package tailsampling

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

func TestLoadConfig(t *testing.T) {
	receivers, processors, exporters, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	processors[factory.Type()] = factory

	cfg, err := config.LoadConfigFile(
		t, path.Join(".", "testdata", "tail_sampling_config.yaml"), receivers, processors, exporters,
	)
	require.Nil(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, cfg.Processors["tail-sampling"],
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail-sampling",
				NameVal: "tail-sampling",
			},
			DecisionWait:            10 * time.Second,
			NumTraces:               100,
			ExpectedNewTracesPerSec: 10,
			PolicyCfg: PolicyCfg{
				Name: "test-policy-1",
				Type: AlwaysSample,
			},
		})

	assert.Equal(t, cfg.Processors["tail-sampling/2"],
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail-sampling",
				NameVal: "tail-sampling/2",
			},
			DecisionWait:            20 * time.Second,
			NumTraces:               200,
			ExpectedNewTracesPerSec: 20,
			PolicyCfg: PolicyCfg{
				Name:                      "test-policy-2",
				Type:                      NumericAttributeFilter,
				NumericAttributeFilterCfg: NumericAttributeFilterCfg{Key: "key1", MinValue: 50, MaxValue: 100},
			},
		})

	assert.Equal(t, cfg.Processors["tail-sampling/3"],
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail-sampling",
				NameVal: "tail-sampling/3",
			},
			DecisionWait:            30 * time.Second,
			NumTraces:               300,
			ExpectedNewTracesPerSec: 30,
			PolicyCfg: PolicyCfg{
				Name:                     "test-policy-3",
				Type:                     StringAttributeFilter,
				StringAttributeFilterCfg: StringAttributeFilterCfg{Key: "key2", Values: []string{"value1", "value2"}},
			},
		})

	assert.Equal(t, cfg.Processors["tail-sampling/4"],
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail-sampling",
				NameVal: "tail-sampling/4",
			},
			DecisionWait:            40 * time.Second,
			NumTraces:               400,
			ExpectedNewTracesPerSec: 40,
			PolicyCfg: PolicyCfg{
				Name:                  "test-policy-4",
				Type:                  RateLimitingFilter,
				RateLimitingFilterCfg: RateLimitingFilterCfg{SpansPerSecond: 35},
			},
		})
}
