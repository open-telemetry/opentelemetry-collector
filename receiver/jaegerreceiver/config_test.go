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

package jaegerreceiver

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigAndValidate(path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 4)

	r1 := cfg.Receivers[config.NewIDWithName(typeStr, "customname")].(*Config)
	assert.Equal(t, r1,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "customname")),
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:9876",
						Transport: "tcp",
					},
				},
				ThriftHTTP: &confighttp.HTTPServerSettings{
					Endpoint: ":3456",
				},
				ThriftCompact: &ProtocolUDP{
					Endpoint: "0.0.0.0:456",
					ServerConfigUDP: ServerConfigUDP{
						QueueSize:        100_000,
						MaxPacketSize:    131_072,
						Workers:          100,
						SocketBufferSize: 65_536,
					},
				},
				ThriftBinary: &ProtocolUDP{
					Endpoint: "0.0.0.0:789",
					ServerConfigUDP: ServerConfigUDP{
						QueueSize:        1_000,
						MaxPacketSize:    65_536,
						Workers:          5,
						SocketBufferSize: 0,
					},
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

	rDefaults := cfg.Receivers[config.NewIDWithName(typeStr, "defaults")].(*Config)
	assert.Equal(t, rDefaults,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "defaults")),
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  defaultGRPCBindEndpoint,
						Transport: "tcp",
					},
				},
				ThriftHTTP: &confighttp.HTTPServerSettings{
					Endpoint: defaultHTTPBindEndpoint,
				},
				ThriftCompact: &ProtocolUDP{
					Endpoint:        defaultThriftCompactBindEndpoint,
					ServerConfigUDP: DefaultServerConfigUDP(),
				},
				ThriftBinary: &ProtocolUDP{
					Endpoint:        defaultThriftBinaryBindEndpoint,
					ServerConfigUDP: DefaultServerConfigUDP(),
				},
			},
		})

	rMixed := cfg.Receivers[config.NewIDWithName(typeStr, "mixed")].(*Config)
	assert.Equal(t, rMixed,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "mixed")),
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:9876",
						Transport: "tcp",
					},
				},
				ThriftCompact: &ProtocolUDP{
					Endpoint:        defaultThriftCompactBindEndpoint,
					ServerConfigUDP: DefaultServerConfigUDP(),
				},
			},
		})

	tlsConfig := cfg.Receivers[config.NewIDWithName(typeStr, "tls")].(*Config)

	assert.Equal(t, tlsConfig,
		&Config{
			ReceiverSettings: config.NewReceiverSettings(config.NewIDWithName(typeStr, "tls")),
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:9876",
						Transport: "tcp",
					},
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "/test.crt",
							KeyFile:  "/test.key",
						},
					},
				},
				ThriftHTTP: &confighttp.HTTPServerSettings{
					Endpoint: ":3456",
				},
			},
		})
}

func TestFailedLoadConfig(t *testing.T) {
	factories, err := componenttest.NopFactories()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	_, err = configtest.LoadConfigAndValidate(path.Join(".", "testdata", "bad_typo_default_proto_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for jaeger: 1 error(s) decoding:\n\n* 'protocols' has invalid keys: thrift_htttp")

	_, err = configtest.LoadConfigAndValidate(path.Join(".", "testdata", "bad_proto_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for jaeger: 1 error(s) decoding:\n\n* 'protocols' has invalid keys: thrift_htttp")

	_, err = configtest.LoadConfigAndValidate(path.Join(".", "testdata", "bad_no_proto_config.yaml"), factories)
	assert.EqualError(t, err, "receiver \"jaeger\" has invalid configuration: must specify at least one protocol when using the Jaeger receiver")

	_, err = configtest.LoadConfigAndValidate(path.Join(".", "testdata", "bad_empty_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for jaeger: empty config for Jaeger receiver")
}
