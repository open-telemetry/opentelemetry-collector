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

package jaegerreceiver

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-service/config"
	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
	"github.com/open-telemetry/opentelemetry-service/receiver"
)

var _ = config.RegisterTestFactories()

func TestLoadConfig(t *testing.T) {
	factory := receiver.GetFactory(typeStr)

	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"))

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 2)

	r0 := cfg.Receivers["jaeger"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["jaeger/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/customname",
			Protocols: map[string]*configmodels.ReceiverSettings{
				"thrift-http": {
					Enabled:  false,
					Endpoint: "127.0.0.1:3456",
				},
				"thrift-tchannel": {
					Enabled:  true,
					Endpoint: "0.0.0.0:123",
				},
			},
		})
}
