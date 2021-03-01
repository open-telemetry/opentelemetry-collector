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

package prometheusexporter

import (
	"path"
	"testing"
	"time"

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
	factories.Exporters[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	e0 := cfg.Exporters["prometheus"]
	assert.Equal(t, e0, factory.CreateDefaultConfig())

	e1 := cfg.Exporters["prometheus/2"]
	assert.Equal(t, e1,
		&Config{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "prometheus/2",
				TypeVal: "prometheus",
			},
			Endpoint:  "1.2.3.4:1234",
			Namespace: "test-space",
			ConstLabels: map[string]string{
				"label1":        "value1",
				"another label": "spaced value",
			},
			SendTimestamps:   true,
			MetricExpiration: 60 * time.Minute,
		})
}
