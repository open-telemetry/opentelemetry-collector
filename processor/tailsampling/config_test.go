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

	p0 := cfg.Processors["tail-sampling"]
	assert.Equal(t, p0,
		&Config{
			ProcessorSettings: configmodels.ProcessorSettings{
				TypeVal: "tail-sampling",
				NameVal: "tail-sampling",
			},
			DecisionWait:            31 * time.Second,
			NumTraces:               20001,
			ExpectedNewTracesPerSec: 100,
		})
}
