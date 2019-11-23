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

package prometheusreceiver

import (
	"os"
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
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers["prometheus"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["prometheus/customname"].(*Config)
	assert.Equal(t, r1.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  "prometheus/customname",
			Endpoint: "1.2.3.4:456",
		})
	assert.Equal(t, r1.PrometheusConfig.ScrapeConfigs[0].JobName, "demo")
	assert.Equal(t, time.Duration(r1.PrometheusConfig.ScrapeConfigs[0].ScrapeInterval), 5*time.Second)
	assert.Equal(t, r1.UseStartTimeMetric, true)
}

func TestLoadConfigWithEnvVar(t *testing.T) {
	const jobname = "JobName"
	const jobnamevar = "JOBNAME"
	os.Setenv(jobnamevar, jobname)

	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config_env.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	r := cfg.Receivers["prometheus"].(*Config)
	assert.Equal(t, r.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal:  typeStr,
			NameVal:  "prometheus",
			Endpoint: "1.2.3.4:456",
		})
	assert.Equal(t, r.PrometheusConfig.ScrapeConfigs[0].JobName, jobname)
	os.Unsetenv(jobnamevar)
}

func TestLoadConfigFailsOnUnknownSection(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(
		t,
		path.Join(".", "testdata", "invalid-config-section.yaml"), factories)

	require.Error(t, err)
	require.Nil(t, cfg)
}
