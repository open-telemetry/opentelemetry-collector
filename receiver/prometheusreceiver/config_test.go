// Copyright 2019, OpenCensus Authors
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

package prometheusreceiver

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/pkg/configmodels"
	"github.com/open-telemetry/opentelemetry-service/pkg/configv2"
	"github.com/open-telemetry/opentelemetry-service/pkg/factories"
)

var _ = configv2.RegisterTestFactories()

func TestLoadConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)

	config, err := configv2.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"))

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, len(config.Receivers), 2)

	r0 := config.Receivers["prometheus"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := config.Receivers["prometheus/customname"].(*ConfigV2)
	assert.Equal(t, r1.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  "prometheus/customname",
			Endpoint: "1.2.3.4:456",
		})
	assert.Equal(t, r1.PrometheusConfig.ScrapeConfigs[0].JobName, "demo")
	assert.Equal(t, time.Duration(r1.PrometheusConfig.ScrapeConfigs[0].ScrapeInterval), 5*time.Second)
}
