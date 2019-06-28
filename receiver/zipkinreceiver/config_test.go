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

package zipkinreceiver

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/configv2"
	"github.com/open-telemetry/opentelemetry-service/factories"
	"github.com/open-telemetry/opentelemetry-service/models"
)

var _ = configv2.RegisterTestFactories()

func TestLoadConfig(t *testing.T) {
	factory := factories.GetReceiverFactory(typeStr)

	config, err := configv2.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"))

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, len(config.Receivers), 2)

	r0 := config.Receivers["zipkin"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := config.Receivers["zipkin/customname"].(*ConfigV2)
	assert.Equal(t, r1,
		&ConfigV2{
			ReceiverSettings: models.ReceiverSettings{
				TypeVal:  typeStr,
				NameVal:  "zipkin/customname",
				Endpoint: "127.0.0.1:8765",
				Enabled:  true,
			},
		})
}
