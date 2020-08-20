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

package otlpreceiver

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configtls"
)

func TestLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	cfg, err := configtest.LoadConfigFile(t, path.Join(".", "testdata", "config.yaml"), factories)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, len(cfg.Receivers), 9)

	assert.Equal(t, cfg.Receivers["otlp"], factory.CreateDefaultConfig())

	defaultOnlyGRPC := factory.CreateDefaultConfig().(*Config)
	defaultOnlyGRPC.SetName("otlp/only_grpc")
	defaultOnlyGRPC.HTTP = nil
	assert.Equal(t, cfg.Receivers["otlp/only_grpc"], defaultOnlyGRPC)

	defaultOnlyHTTP := factory.CreateDefaultConfig().(*Config)
	defaultOnlyHTTP.SetName("otlp/only_http")
	defaultOnlyHTTP.GRPC = nil
	assert.Equal(t, cfg.Receivers["otlp/only_http"], defaultOnlyHTTP)

	assert.Equal(t, cfg.Receivers["otlp/customname"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/customname",
			},
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:9090",
						Transport: "tcp",
					},
					ReadBufferSize: 512 * 1024,
				},
			},
		})

	assert.Equal(t, cfg.Receivers["otlp/keepalive"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/keepalive",
			},
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "0.0.0.0:55680",
						Transport: "tcp",
					},
					ReadBufferSize: 512 * 1024,
					Keepalive: &configgrpc.KeepaliveServerConfig{
						ServerParameters: &configgrpc.KeepaliveServerParameters{
							MaxConnectionIdle:     11 * time.Second,
							MaxConnectionAge:      12 * time.Second,
							MaxConnectionAgeGrace: 13 * time.Second,
							Time:                  30 * time.Second,
							Timeout:               5 * time.Second,
						},
						EnforcementPolicy: &configgrpc.KeepaliveEnforcementPolicy{
							MinTime:             10 * time.Second,
							PermitWithoutStream: true,
						},
					},
				},
			},
		})

	assert.Equal(t, cfg.Receivers["otlp/msg-size-conc-connect-max-idle"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/msg-size-conc-connect-max-idle",
			},
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "0.0.0.0:55680",
						Transport: "tcp",
					},
					MaxRecvMsgSizeMiB:    32,
					MaxConcurrentStreams: 16,
					ReadBufferSize:       1024,
					WriteBufferSize:      1024,
					Keepalive: &configgrpc.KeepaliveServerConfig{
						ServerParameters: &configgrpc.KeepaliveServerParameters{
							MaxConnectionIdle: 10 * time.Second,
						},
					},
				},
			},
		})

	// NOTE: Once the config loader checks for the files existence, this test may fail and require
	// 	use of fake cert/key for test purposes.
	assert.Equal(t, cfg.Receivers["otlp/tlscredentials"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/tlscredentials",
			},
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "0.0.0.0:55680",
						Transport: "tcp",
					},
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "test.crt",
							KeyFile:  "test.key",
						},
					},
					ReadBufferSize: 512 * 1024,
				},
				HTTP: &confighttp.HTTPServerSettings{
					Endpoint: "0.0.0.0:55681",
					TLSSetting: &configtls.TLSServerSetting{
						TLSSetting: configtls.TLSSetting{
							CertFile: "test.crt",
							KeyFile:  "test.key",
						},
					},
				},
			},
		})

	assert.Equal(t, cfg.Receivers["otlp/cors"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/cors",
			},
			Protocols: Protocols{
				HTTP: &confighttp.HTTPServerSettings{
					Endpoint:    "0.0.0.0:55681",
					CorsOrigins: []string{"https://*.test.com", "https://test.com"},
				},
			},
		})

	assert.Equal(t, cfg.Receivers["otlp/uds"],
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "otlp/uds",
			},
			Protocols: Protocols{
				GRPC: &configgrpc.GRPCServerSettings{
					NetAddr: confignet.NetAddr{
						Endpoint:  "/tmp/grpc_otlp.sock",
						Transport: "unix",
					},
					ReadBufferSize: 512 * 1024,
				},
				HTTP: &confighttp.HTTPServerSettings{
					Endpoint: "/tmp/http_otlp.sock",
					// Transport: "unix",
				},
			},
		})
}

func TestFailedLoadConfig(t *testing.T) {
	factories, err := componenttest.ExampleComponents()
	assert.NoError(t, err)

	factory := NewFactory()
	factories.Receivers[typeStr] = factory
	_, err = configtest.LoadConfigFile(t, path.Join(".", "testdata", "typo_default_proto_config.yaml"), factories)
	assert.EqualError(t, err, `error reading receivers configuration for otlp: unknown protocols in the OTLP receiver`)

	_, err = configtest.LoadConfigFile(t, path.Join(".", "testdata", "bad_proto_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for otlp: 1 error(s) decoding:\n\n* 'protocols' has invalid keys: thrift")

	_, err = configtest.LoadConfigFile(t, path.Join(".", "testdata", "bad_no_proto_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for otlp: must specify at least one protocol when using the OTLP receiver")

	_, err = configtest.LoadConfigFile(t, path.Join(".", "testdata", "bad_empty_config.yaml"), factories)
	assert.EqualError(t, err, "error reading receivers configuration for otlp: empty config for OTLP receiver")
}
