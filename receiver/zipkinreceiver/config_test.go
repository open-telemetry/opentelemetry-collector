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

package zipkinreceiver

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtest"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 3)

	r0 := cfg.Receivers[config.NewID(typeStr)]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers[config.NewIDWithName(typeStr, "customname")].(*Config)
	assert.Equal(t, r1,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "customname")),
			HTTPServerSettings: confighttp.HTTPServerSettings{
				Endpoint: "localhost:8765",
			},
		})

	r2 := cfg.Receivers[config.NewIDWithName(typeStr, "parse_strings")].(*Config)
	assert.Equal(t, r2,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "parse_strings")),
			HTTPServerSettings: confighttp.HTTPServerSettings{
				Endpoint: "0.0.0.0:9411",
			},
			ParseStringTags: true,
		})
}
