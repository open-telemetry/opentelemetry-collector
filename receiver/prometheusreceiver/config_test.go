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

package prometheusreceiver

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers["prometheus"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["prometheus/customname"].(*Config)
	assert.Equal(t, r1.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: "prometheus/customname",
		})
	assert.Equal(t, r1.PrometheusConfig.ScrapeConfigs[0].JobName, "demo")
	assert.Equal(t, time.Duration(r1.PrometheusConfig.ScrapeConfigs[0].ScrapeInterval), 5*time.Second)
	assert.Equal(t, r1.UseStartTimeMetric, true)
	assert.Equal(t, r1.StartTimeMetricRegex, "^(.+_)*process_start_time_seconds$")
}

func TestLoadConfigWithEnvVar(t *testing.T) {
	const jobname = "JobName"
	const jobnamevar = "JOBNAME"
	os.Setenv(jobnamevar, jobname)

	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_env.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	r := cfg.Receivers["prometheus"].(*Config)
	assert.Equal(t, r.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: "prometheus",
		})
	assert.Equal(t, r.PrometheusConfig.ScrapeConfigs[0].JobName, jobname)
	os.Unsetenv(jobnamevar)
}

func TestLoadConfigK8s(t *testing.T) {
	const node = "node1"
	const nodenamevar = "NODE_NAME"
	os.Setenv(nodenamevar, node)
	defer os.Unsetenv(nodenamevar)

	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config_k8s.yaml"), factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	r := cfg.Receivers["prometheus"].(*Config)
	assert.Equal(t, r.ReceiverSettings,
		configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: "prometheus",
		})

	scrapeConfig := r.PrometheusConfig.ScrapeConfigs[0]
	kubeSDConfig := scrapeConfig.ServiceDiscoveryConfigs[0].(*kubernetes.SDConfig)
	assert.Equal(t,
		kubeSDConfig.Selectors[0].Field,
		fmt.Sprintf("spec.nodeName=%s", node))
	assert.Equal(t,
		scrapeConfig.RelabelConfigs[1].Replacement,
		"$1:$2")
}

func TestLoadConfigFailsOnUnknownSection(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(
		t,
		path.Join(".", "testdata", "invalid-config-section.yaml"), factories)

	require.Error(t, err)
	require.Nil(t, cfg)
}

// As one of the config parameters is consuming prometheus
// configuration as a subkey, ensure that invalid configuration
// within the subkey will also raise an error.
func TestLoadConfigFailsOnUnknownPrometheusSection(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(
		t,
		path.Join(".", "testdata", "invalid-config-prometheus-section.yaml"), factories)

	require.Error(t, err)
	require.Nil(t, cfg)
}
