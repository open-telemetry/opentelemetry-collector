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

package zipkinexporter

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Exporters[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	e0 := cfg.Exporters["zipkin"]

	// URL doesn't have a default value so set it directly.
	defaultCfg := factory.CreateDefaultConfig().(*Config)
	defaultCfg.Endpoint = "http://some.location.org:9411/api/v2/spans"
	assert.Equal(t, defaultCfg, e0)
	assert.Equal(t, "json", e0.(*Config).Format)

	e1 := cfg.Exporters["zipkin/2"]
	assert.Equal(t, &Config{
		ExporterSettings: configmodels.ExporterSettings{
			NameVal: "zipkin/2",
			TypeVal: "zipkin",
		},
		RetrySettings: exporterhelper.RetrySettings{
			Enabled:         true,
			InitialInterval: 10 * time.Second,
			MaxInterval:     1 * time.Minute,
			MaxElapsedTime:  10 * time.Minute,
		},
		QueueSettings: exporterhelper.QueueSettings{
			Enabled:      true,
			NumConsumers: 2,
			QueueSize:    10,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint:        "https://somedest:1234/api/v2/spans",
			WriteBufferSize: 524288,
			Timeout:         5 * time.Second,
		},
		Format:             "proto",
		DefaultServiceName: "test_name",
	}, e1)
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	_, err = factory.CreateTracesExporter(context.Background(), params, e1)
	require.NoError(t, err)
}
