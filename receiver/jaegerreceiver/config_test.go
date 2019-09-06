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

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.Nil(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The receiver `jaeger/disabled` doesn't count because disabled receivers
	// are excluded from the final list.
	assert.Equal(t, len(cfg.Receivers), 3)

	r0 := cfg.Receivers["jaeger"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["jaeger/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/customname",
			Protocols: map[string]*receiver.SecureReceiverSettings{
				"grpc": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: "127.0.0.1:9876",
					},
				},
				"thrift-http": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: ":3456",
					},
				},
				"thrift-tchannel": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: "0.0.0.0:123",
					},
				},
			},
		})

	tlsConfig := cfg.Receivers["jaeger/tls"].(*Config)

	assert.Equal(t, tlsConfig,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/tls",
			Protocols: map[string]*receiver.SecureReceiverSettings{
				"grpc": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: "127.0.0.1:9876",
					},
					TLSCredentials: &receiver.TLSCredentials{
						CertFile: "/test.crt",
						KeyFile:  "/test.key",
					},
				},
				"thrift-http": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: ":3456",
					},
				},
				"thrift-tchannel": {
					ReceiverSettings: configmodels.ReceiverSettings{
						Endpoint: "0.0.0.0:123",
					},
				},
			},
		})
}
