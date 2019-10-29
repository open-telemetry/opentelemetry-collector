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

package zipkinexporter

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/config"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Exporters[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	e0 := cfg.Exporters["zipkin"]

	// URL doesn't have a default value so set it directly.
	defaultCfg := factory.CreateDefaultConfig().(*Config)
	defaultCfg.URL = "http://some.location.org:9411/api/v2/spans"
	assert.Equal(t, defaultCfg, e0)
	assert.Equal(t, "json", e0.(*Config).Format)

	e1 := cfg.Exporters["zipkin/2"]
	assert.Equal(t, "zipkin/2", e1.(*Config).Name())
	assert.Equal(t, "https://somedest:1234/api/v2/spans", e1.(*Config).URL)
	assert.Equal(t, "proto", e1.(*Config).Format)
	_, err = factory.CreateTraceExporter(zap.NewNop(), e1)
	require.NoError(t, err)
}
