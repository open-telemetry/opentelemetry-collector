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

package jaegerexporter

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
	"go.opentelemetry.io/collector/config/configgrpc"
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

	e0 := cfg.Exporters["jaeger"]
	assert.Equal(t, e0, factory.CreateDefaultConfig())

	e1 := cfg.Exporters["jaeger/2"]
	assert.Equal(t, e1,
		&Config{
			ExporterSettings: configmodels.ExporterSettings{
				NameVal: "jaeger/2",
				TypeVal: "jaeger",
			},
			TimeoutSettings: exporterhelper.TimeoutSettings{
				Timeout: 10 * time.Second,
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
			GRPCClientSettings: configgrpc.GRPCClientSettings{
				Endpoint:        "a.new.target:1234",
				WriteBufferSize: 512 * 1024,
				BalancerName:    "round_robin",
			},
		})

	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	te, err := factory.CreateTracesExporter(context.Background(), params, e1)
	require.NoError(t, err)
	require.NotNil(t, te)
}
