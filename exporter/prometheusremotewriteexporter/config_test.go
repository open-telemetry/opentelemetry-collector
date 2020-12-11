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

package prometheusremotewriteexporter

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// TestLoadConfig checks whether yaml configuration can be loaded correctly
func Test_loadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Exporters[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// From the default configurations -- checks if a correct exporter is instantiated
	e0 := cfg.Exporters["prometheusremotewrite"]
	assert.Equal(t, e0, factory.CreateDefaultConfig())

	// checks if the correct Config struct can be instantiated from testdata/config.yaml
	e1 := cfg.Exporters["prometheusremotewrite/2"]
	assert.Equal(t, e1,
		&Config{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "prometheusremotewrite/2",
				TypeVal: "prometheusremotewrite",
			},
			TimeoutSettings: exporterhelper.DefaultTimeoutSettings(),
			QueueSettings: exporterhelper.QueueSettings{
				Enabled:      true,
				NumConsumers: 2,
				QueueSize:    10,
			},
			RetrySettings: exporterhelper.RetrySettings{
				Enabled:         true,
				InitialInterval: 10 * time.Second,
				MaxInterval:     1 * time.Minute,
				MaxElapsedTime:  10 * time.Minute,
			},
			Namespace:      "test-space",
			ExternalLabels: map[string]string{"key1": "value1", "key2": "value2"},
			HTTPClientSettings: confighttp.HTTPClientSettings{
				Endpoint: "localhost:8888",
				TLSSetting: configtls.TLSClientSetting{
					TLSSetting: configtls.TLSSetting{
						CAFile: "/var/lib/mycert.pem", // This is subject to change, but currently I have no idea what else to put here lol
					},
					Insecure: false,
				},
				ReadBufferSize:  0,
				WriteBufferSize: 512 * 1024,
				Timeout:         5 * time.Second,
				Headers: map[string]string{
					"prometheus-remote-write-version": "0.1.0",
					"x-scope-orgid":                   "234"},
			},
		})
}
