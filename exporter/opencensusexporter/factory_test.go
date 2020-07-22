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

package opencensusexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/receiver/opencensusreceiver"
	"go.opentelemetry.io/collector/testutil"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateMetricsExporter(t *testing.T) {
	factory := Factory{}
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.GRPCClientSettings.Endpoint = testutil.GetAvailableLocalAddress(t)

	oexp, err := factory.CreateMetricsExporter(zap.NewNop(), cfg)
	require.Nil(t, err)
	require.NotNil(t, oexp)
}

func TestCreateTraceExporter(t *testing.T) {
	// This test is about creating the exporter and stopping it. However, the
	// exporter keeps trying to update its connection state in the background
	// so unless there is a receiver enabled the stop call can return different
	// results. Standing up a receiver to ensure that stop don't report errors.
	rcvFactory := &opencensusreceiver.Factory{}
	require.NotNil(t, rcvFactory)
	rcvCfg := rcvFactory.CreateDefaultConfig().(*opencensusreceiver.Config)
	rcvCfg.NetAddr.Endpoint = testutil.GetAvailableLocalAddress(t)

	rcv, err := rcvFactory.CreateTraceReceiver(
		context.Background(),
		zap.NewNop(),
		rcvCfg,
		new(exportertest.SinkTraceExporterOld))
	require.NotNil(t, rcv)
	require.Nil(t, err)
	require.Nil(t, rcv.Start(context.Background(), componenttest.NewNopHost()))
	defer rcv.Shutdown(context.Background())

	tests := []struct {
		name     string
		config   Config
		mustFail bool
	}{
		{
			name: "NoEndpoint",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: "",
				},
			},
			mustFail: true,
		},
		{
			name: "UseSecure",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
					TLSSetting: configtls.TLSClientSetting{
						Insecure: false,
					},
				},
			},
		},
		{
			name: "ReconnectionDelay",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
				},
				ReconnectionDelay: 5 * time.Second,
			},
		},
		{
			name: "Keepalive",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
					Keepalive: &configgrpc.KeepaliveClientConfig{
						Time:                30 * time.Second,
						Timeout:             25 * time.Second,
						PermitWithoutStream: true,
					},
				},
			},
		},
		{
			name: "Compression",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    rcvCfg.NetAddr.Endpoint,
					Compression: configgrpc.CompressionGzip,
				},
			},
		},
		{
			name: "Headers",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
					Headers: map[string]string{
						"hdr1": "val1",
						"hdr2": "val2",
					},
				},
			},
		},
		{
			name: "NumConsumers",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
				},
				NumWorkers: 3,
			},
		},
		{
			name: "CompressionError",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint:    rcvCfg.NetAddr.Endpoint,
					Compression: "unknown compression",
				},
			},
			mustFail: true,
		},
		{
			name: "CaCert",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
							CAFile: "testdata/test_cert.pem",
						},
					},
				},
			},
		},
		{
			name: "CertPemFileError",
			config: Config{
				GRPCClientSettings: configgrpc.GRPCClientSettings{
					Endpoint: rcvCfg.NetAddr.Endpoint,
					TLSSetting: configtls.TLSClientSetting{
						TLSSetting: configtls.TLSSetting{
							CAFile: "nosuchfile",
						},
					},
				},
			},
			mustFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &Factory{}
			consumer, err := factory.CreateTraceExporter(zap.NewNop(), &tt.config)

			if tt.mustFail {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)

				err = consumer.Shutdown(context.Background())
				if err != nil {
					// Since the endpoint of opencensus exporter doesn't actually exist,
					// exporter may already stop because it cannot connect.
					assert.Equal(t, err.Error(), "rpc error: code = Canceled desc = grpc: the client connection is closing")
				}
			}
		})
	}
}
