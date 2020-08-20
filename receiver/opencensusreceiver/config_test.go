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

package opencensusreceiver

import (
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
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

	assert.Equal(t, len(cfg.Receivers), 7)

	r0 := cfg.Receivers["opencensus"]
	assert.Equal(t, r0, factory.CreateDefaultConfig())

	r1 := cfg.Receivers["opencensus/customname"].(*Config)
	assert.Equal(t, r1,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/customname",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:9090",
					Transport: "tcp",
				},
				ReadBufferSize: 512 * 1024,
			},
		})

	r2 := cfg.Receivers["opencensus/keepalive"].(*Config)
	assert.Equal(t, r2,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/keepalive",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:55678",
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
		})

	r3 := cfg.Receivers["opencensus/msg-size-conc-connect-max-idle"].(*Config)
	assert.Equal(t, r3,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/msg-size-conc-connect-max-idle",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:55678",
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
		})

	// TODO(ccaraman): Once the config loader checks for the files existence, this test may fail and require
	// 	use of fake cert/key for test purposes.
	r4 := cfg.Receivers["opencensus/tlscredentials"].(*Config)
	assert.Equal(t, r4,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/tlscredentials",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:55678",
					Transport: "tcp",
				},
				ReadBufferSize: 512 * 1024,
				TLSSetting: &configtls.TLSServerSetting{
					TLSSetting: configtls.TLSSetting{
						CertFile: "test.crt",
						KeyFile:  "test.key",
					},
				},
			},
		})

	r5 := cfg.Receivers["opencensus/cors"].(*Config)
	assert.Equal(t, r5,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/cors",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "0.0.0.0:55678",
					Transport: "tcp",
				},
				ReadBufferSize: 512 * 1024,
			},
			CorsOrigins: []string{"https://*.test.com", "https://test.com"},
		})

	r6 := cfg.Receivers["opencensus/uds"].(*Config)
	assert.Equal(t, r6,
		&Config{
			ReceiverSettings: configmodels.ReceiverSettings{
				TypeVal: typeStr,
				NameVal: "opencensus/uds",
			},
			GRPCServerSettings: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  "/tmp/opencensus.sock",
					Transport: "unix",
				},
				ReadBufferSize: 512 * 1024,
			},
		})
}

func TestBuildOptions_TLSCredentials(t *testing.T) {
	cfg := Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			NameVal: "IncorrectTLS",
		},
		GRPCServerSettings: configgrpc.GRPCServerSettings{
			TLSSetting: &configtls.TLSServerSetting{
				TLSSetting: configtls.TLSSetting{
					CertFile: "willfail",
				},
			},
		},
	}
	_, err := cfg.buildOptions()
	assert.EqualError(t, err, `failed to load TLS config: for auth via TLS, either both certificate and key must be supplied, or neither`)

	cfg.TLSSetting = &configtls.TLSServerSetting{}
	opt, err := cfg.buildOptions()
	assert.NoError(t, err)
	assert.NotNil(t, opt)
}
