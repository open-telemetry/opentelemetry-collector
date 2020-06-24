// Copyright The OpenTelemetry Authors
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

	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configprotocol"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	cfg, err := config.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 4)

	r1 := cfg.Receivers["jaeger/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/customname",
			Protocols: map[string]*configprotocol.ProtocolServerSettings{
				"grpc": {
					Endpoint: "localhost:9876",
				},
				"thrift_http": {
					Endpoint: ":3456",
				},
				"thrift_compact": {
					Endpoint: "0.0.0.0:456",
				},
				"thrift_binary": {
					Endpoint: "0.0.0.0:789",
				},
			},
			RemoteSampling: &RemoteSamplingConfig{
				HostEndpoint: "0.0.0.0:5778",
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "jaeger-collector:1234",
				},
				StrategyFile: "/etc/strategies.json",
			},
		})

	rDefaults := cfg.Receivers["jaeger/defaults"].(*Config)
	assert.Equal(t, rDefaults,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/defaults",
			Protocols: map[string]*configprotocol.ProtocolServerSettings{
				"grpc": {
					Endpoint: defaultGRPCBindEndpoint,
				},
				"thrift_http": {
					Endpoint: defaultHTTPBindEndpoint,
				},
				"thrift_compact": {
					Endpoint: defaultThriftCompactBindEndpoint,
				},
				"thrift_binary": {
					Endpoint: defaultThriftBinaryBindEndpoint,
				},
			},
		})

	rMixed := cfg.Receivers["jaeger/mixed"].(*Config)
	assert.Equal(t, rMixed,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/mixed",
			Protocols: map[string]*configprotocol.ProtocolServerSettings{
				"grpc": {
					Endpoint: "localhost:9876",
				},
				"thrift_compact": {
					Endpoint: defaultThriftCompactBindEndpoint,
				},
			},
		})

	tlsConfig := cfg.Receivers["jaeger/tls"].(*Config)

	assert.Equal(t, tlsConfig,
		&Config{
			TypeVal: typeStr,
			NameVal: "jaeger/tls",
			Protocols: map[string]*configprotocol.ProtocolServerSettings{
				"grpc": {
					Endpoint: "localhost:9876",
					TLSCredentials: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "/test.crt",
							KeyFile:  "/test.key",
						},
					},
				},
				"thrift_http": {
					Endpoint: ":3456",
				},
			},
		})
}

func TestFailedLoadConfig(t *testing.T) {
	factories, err := config.ExampleComponents()
	assert.NoError(t, err)

	factory := &Factory{}
	factories.Receivers[typeStr] = factory
	_, err = config.LoadConfigFile(t, path.Join(".", "testdata", "bad_proto_config.yaml"), factories)
	assert.EqualError(t, err, `error reading settings for receiver type "jaeger": unknown Jaeger protocol badproto`)

	_, err = config.LoadConfigFile(t, path.Join(".", "testdata", "bad_no_proto_config.yaml"), factories)
	assert.EqualError(t, err, `error reading settings for receiver type "jaeger": must specify at least one protocol when using the Jaeger receiver`)

	_, err = config.LoadConfigFile(t, path.Join(".", "testdata", "bad_empty_config.yaml"), factories)
	assert.EqualError(t, err, `error reading settings for receiver type "jaeger": empty config for Jaeger receiver`)
}
