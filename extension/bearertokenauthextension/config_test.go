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

package bearertokenauthextension

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	expected := factory.CreateDefaultConfig().(*Config)
	expected.BearerToken = "sometoken"

	ext0 := cfg.Extensions[config.NewID(typeStr)]
	assert.Equal(t, expected, ext0)

	ext1 := cfg.Extensions[config.NewIDWithName(typeStr, "1")]
	assert.Equal(t,
		&Config{
			ExtensionSettings: config.NewExtensionSettings(config.NewIDWithName(typeStr, "1")),
			BearerToken:       "sometesttoken",
		},
		ext1)

	assert.Equal(t, 1, len(cfg.Service.Extensions))
	assert.Equal(t, config.NewIDWithName(typeStr, "1"), cfg.Service.Extensions[0])
}

func TestLoadConfigError(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Extensions[typeStr] = factory
	_, err = configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config_missing_token.yaml"), factories)
	require.Error(t, err)
}
